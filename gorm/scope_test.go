package gw_gorm

import (
	"database/sql"
	"errors"
	"strings"
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
)

type guardedTodo struct {
	Id             string
	TenantId       string
	OrganizationId string
	Title          string
}

func (guardedTodo) TenantScoped() {}
func (guardedTodo) OrgScoped()    {}

type guardedOrganization struct {
	Id       string
	TenantId string
	Name     string
}

func (guardedOrganization) TenantScoped()  {}
func (guardedOrganization) OrgSelfScoped() {}

type plainUser struct {
	Id       string
	TenantId string
	Email    string
}

type plainNote struct {
	Id   string
	Text string
}

type tenantOnlyRecord struct {
	Id       string
	TenantId string
	Name     string
}

func (tenantOnlyRecord) TenantScoped() {}

// DryRun + DummyDialector で SQL 組み立てのみ検証する（実 DB 不要）。
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(tests.DummyDialector{}, &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := UseTenantGuard(db); err != nil {
		t.Fatalf("UseTenantGuard: %v", err)
	}
	return db
}

func singleScope() *Scope {
	return &Scope{TenantIds: []string{"t1"}, OrgIds: []string{"o1"}}
}

func containsVar(vars []interface{}, want interface{}) bool {
	for _, v := range vars {
		if v == want {
			return true
		}
	}
	return false
}

// ApplyScope の返り値を変数に取って複数クエリに使い回しても、
// 前のクエリの条件が次のクエリに残留しない（ステートメント汚染防止）。
func TestApplyScopeReuseDoesNotPolluteStatement(t *testing.T) {
	db := openTestDB(t)
	scoped := ApplyScope(db, singleScope())

	var a, b []guardedTodo
	first := scoped.Where("id = ?", "id-1").Find(&a)
	if first.Error != nil {
		t.Fatalf("first query: %v", first.Error)
	}
	sql1 := first.Statement.SQL.String()
	if !strings.Contains(sql1, "tenant_id") {
		t.Fatalf("first query must contain tenant guard: %s", sql1)
	}

	second := scoped.Where("title = ?", "title-2").Find(&b)
	if second.Error != nil {
		t.Fatalf("second query: %v", second.Error)
	}
	sql2 := second.Statement.SQL.String()
	if containsVar(second.Statement.Vars, "id-1") {
		t.Fatalf("second query polluted by first query's condition: %s vars=%v", sql2, second.Statement.Vars)
	}
	if !containsVar(second.Statement.Vars, "title-2") {
		t.Fatalf("second query lost its own condition: %s vars=%v", sql2, second.Statement.Vars)
	}
	if c := strings.Count(sql2, "tenant_id"); c != 1 {
		t.Fatalf("tenant guard must be applied exactly once, got %d: %s", c, sql2)
	}
}

// スコープ（Set した値）は Session 後の Statement 複製にも伝搬し、
// 使い回した 2 回目以降のクエリにもガードが効き続ける。
func TestApplyScopePropagatesAcrossReuse(t *testing.T) {
	db := openTestDB(t)
	scoped := ApplyScope(db, &Scope{TenantIds: []string{"t1"}, OrgIds: []string{"o1", "o2"}})

	for i := 0; i < 3; i++ {
		var list []guardedTodo
		tx := scoped.Find(&list)
		if tx.Error != nil {
			t.Fatalf("query %d: %v", i, tx.Error)
		}
		sql := tx.Statement.SQL.String()
		if !strings.Contains(sql, "tenant_id") || !strings.Contains(sql, "organization_id") {
			t.Fatalf("query %d lost tenant guard: %s", i, sql)
		}
		if !containsVar(tx.Statement.Vars, "t1") {
			t.Fatalf("query %d lost tenant id var: %v", i, tx.Statement.Vars)
		}
	}
}

// BypassTenantGuard も再利用可能な起点として振る舞い、skip 指定が使い回し後も効く。
func TestBypassTenantGuardReuse(t *testing.T) {
	db := openTestDB(t)
	free := BypassTenantGuard(db)

	for i := 0; i < 2; i++ {
		var list []guardedTodo
		tx := free.Find(&list)
		if tx.Error != nil {
			t.Fatalf("query %d: %v", i, tx.Error)
		}
		if sql := tx.Statement.SQL.String(); strings.Contains(sql, "tenant_id") {
			t.Fatalf("query %d must not contain tenant guard: %s", i, sql)
		}
	}
}

// ApplyScopeとBypassTenantGuardは、常に後から呼んだ設定を最終状態とする。
func TestScopeAndBypassUseLastCallWins(t *testing.T) {
	db := openTestDB(t)

	guarded := ApplyScope(BypassTenantGuard(db), singleScope())
	var guardedRows []guardedTodo
	guardedQuery := guarded.Find(&guardedRows)
	if guardedQuery.Error != nil {
		t.Fatalf("ApplyScope after bypass: %v", guardedQuery.Error)
	}
	if sql := guardedQuery.Statement.SQL.String(); !strings.Contains(sql, "tenant_id") || !strings.Contains(sql, "organization_id") {
		t.Fatalf("last ApplyScope must enable guard: %s", sql)
	}

	bypassed := BypassTenantGuard(ApplyScope(db, singleScope()))
	var bypassedRows []guardedTodo
	bypassedQuery := bypassed.Find(&bypassedRows)
	if bypassedQuery.Error != nil {
		t.Fatalf("BypassTenantGuard after scope: %v", bypassedQuery.Error)
	}
	if sql := bypassedQuery.Statement.SQL.String(); strings.Contains(sql, "tenant_id") || strings.Contains(sql, "organization_id") {
		t.Fatalf("last BypassTenantGuard must disable guard: %s", sql)
	}
}

func TestScopeAndBypassLastCallWinsInsideTransaction(t *testing.T) {
	db := openTransactionTestDB(t)
	seedGuardedTodos(t, db)

	if err := ApplyScope(BypassTenantGuard(db), singleScope()).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&guardedTodo{}).Count(&count).Error; err != nil {
			return err
		}
		if count != 1 {
			t.Fatalf("last ApplyScope transaction count=%d", count)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if err := BypassTenantGuard(ApplyScope(db, singleScope())).Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&guardedTodo{}).Count(&count).Error; err != nil {
			return err
		}
		if count != 2 {
			t.Fatalf("last BypassTenantGuard transaction count=%d", count)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// スコープ未設定のままガード対象モデルを触ると拒否される。
func TestGuardRejectsWithoutScope(t *testing.T) {
	db := openTestDB(t)
	var list []guardedTodo
	tx := db.Find(&list)
	if tx.Error == nil || !strings.Contains(tx.Error.Error(), "tenant scope is required") {
		t.Fatalf("expected 'tenant scope is required', got %v", tx.Error)
	}
}

// Scopeを導入しないモデルは、Tenant Guardを登録したDBでも通常どおり利用できる。
func TestGuardAllowsUnscopedModelWithoutScope(t *testing.T) {
	db := openTestDB(t)
	var list []plainNote
	tx := db.Find(&list)
	if tx.Error != nil {
		t.Fatalf("unscoped model must work without scope: %v", tx.Error)
	}
	if sql := tx.Statement.SQL.String(); strings.Contains(sql, "tenant_id") || strings.Contains(sql, "organization_id") {
		t.Fatalf("unscoped model must not receive scope conditions: %s", sql)
	}
}

// Tenantだけを採用するアプリではOrgIdsを設定せず、TenantScopedだけで利用できる。
func TestTenantOnlyModelDoesNotRequireOrganizationScope(t *testing.T) {
	db := openTestDB(t)
	scoped := ApplyScope(db, &Scope{TenantIds: []string{"t1"}})
	var list []tenantOnlyRecord
	tx := scoped.Find(&list)
	if tx.Error != nil {
		t.Fatalf("tenant-only query: %v", tx.Error)
	}
	sql := tx.Statement.SQL.String()
	if !strings.Contains(sql, "tenant_id") || strings.Contains(sql, "organization_id") {
		t.Fatalf("tenant-only query has unexpected conditions: %s", sql)
	}

	row := &tenantOnlyRecord{Id: "id-1", Name: "example"}
	tx = scoped.Create(row)
	if tx.Error != nil {
		t.Fatalf("tenant-only create: %v", tx.Error)
	}
	if row.TenantId != "t1" {
		t.Fatalf("tenant_id must be auto-set, got %q", row.TenantId)
	}
}

func TestGuardedUpdateRejectsScopeBoundaryChanges(t *testing.T) {
	db := openTransactionTestDB(t)
	seedGuardedTodos(t, db)
	scoped := ApplyScope(db, singleScope())

	if result := scoped.Model(&guardedTodo{}).Where("id = ?", "t1-row").Update("title", "updated"); result.Error != nil || result.RowsAffected != 1 {
		t.Fatalf("in-scope update error=%v rows=%d", result.Error, result.RowsAffected)
	}
	if result := scoped.Model(&guardedTodo{}).Where("id = ?", "t2-row").Update("title", "blocked"); result.Error != nil || result.RowsAffected != 0 {
		t.Fatalf("out-of-scope row update error=%v rows=%d", result.Error, result.RowsAffected)
	}
	if result := scoped.Model(&guardedTodo{}).Where("id = ?", "t1-row").Updates(map[string]interface{}{
		"tenant_id":       "t1",
		"organization_id": "o1",
	}); result.Error != nil || result.RowsAffected != 1 {
		t.Fatalf("same-scope boundary update error=%v rows=%d", result.Error, result.RowsAffected)
	}
	if result := scoped.Model(&guardedTodo{}).Where("id = ?", "t1-row").Omit("tenant_id").Updates(map[string]interface{}{
		"tenant_id": "t2",
		"title":     "omitted-boundary",
	}); result.Error != nil || result.RowsAffected != 1 {
		t.Fatalf("omitted boundary field must not be validated error=%v rows=%d", result.Error, result.RowsAffected)
	}

	updates := []struct {
		name   string
		update func() *gorm.DB
		want   string
	}{
		{"tenant column", func() *gorm.DB {
			return scoped.Model(&guardedTodo{}).Where("id = ?", "t1-row").Update("tenant_id", "t2")
		}, "tenant is out of scope"},
		{"tenant field map", func() *gorm.DB {
			return scoped.Model(&guardedTodo{}).Where("id = ?", "t1-row").Updates(map[string]interface{}{"TenantId": "t2"})
		}, "tenant is out of scope"},
		{"organization map", func() *gorm.DB {
			return scoped.Model(&guardedTodo{}).Where("id = ?", "t1-row").Updates(map[string]interface{}{"organization_id": "o2"})
		}, "organization is out of scope"},
		{"update columns", func() *gorm.DB {
			return scoped.Model(&guardedTodo{}).Where("id = ?", "t1-row").UpdateColumns(map[string]interface{}{"tenant_id": "t2"})
		}, "tenant is out of scope"},
		{"tenant struct", func() *gorm.DB {
			return scoped.Model(&guardedTodo{}).Where("id = ?", "t1-row").Updates(guardedTodo{TenantId: "t2"})
		}, "tenant is out of scope"},
		{"organization struct", func() *gorm.DB {
			return scoped.Model(&guardedTodo{}).Where("id = ?", "t1-row").Updates(guardedTodo{OrganizationId: "o2"})
		}, "organization is out of scope"},
		{"expression", func() *gorm.DB {
			return scoped.Model(&guardedTodo{}).Where("id = ?", "t1-row").Update("tenant_id", gorm.Expr("tenant_id"))
		}, "tenant is out of scope"},
	}
	for _, test := range updates {
		t.Run(test.name, func(t *testing.T) {
			result := test.update()
			if result.Error == nil || !strings.Contains(result.Error.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, result.Error)
			}
		})
	}

	var row guardedTodo
	if err := scoped.Where("id = ?", "t1-row").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	row.TenantId = "t2"
	if err := scoped.Save(&row).Error; err == nil || !strings.Contains(err.Error(), "tenant is out of scope") {
		t.Fatalf("Save must reject tenant move: %v", err)
	}
}

func TestGuardedMutationsPreserveMissingWhereProtection(t *testing.T) {
	db := openTransactionTestDB(t)
	seedGuardedTodos(t, db)
	scoped := ApplyScope(db, singleScope())

	if err := scoped.Model(&guardedTodo{}).Update("title", "all").Error; !errors.Is(err, gorm.ErrMissingWhereClause) {
		t.Fatalf("scope injection must not permit conditionless update: %v", err)
	}
	if err := scoped.Delete(&guardedTodo{}).Error; !errors.Is(err, gorm.ErrMissingWhereClause) {
		t.Fatalf("scope injection must not permit conditionless delete: %v", err)
	}

	deleteInScope := scoped.Delete(&guardedTodo{Id: "t1-row"})
	if deleteInScope.Error != nil || deleteInScope.RowsAffected != 1 {
		t.Fatalf("primary-key delete error=%v rows=%d", deleteInScope.Error, deleteInScope.RowsAffected)
	}
	deleteOutOfScope := scoped.Delete(&guardedTodo{Id: "t2-row"})
	if deleteOutOfScope.Error != nil || deleteOutOfScope.RowsAffected != 0 {
		t.Fatalf("out-of-scope delete error=%v rows=%d", deleteOutOfScope.Error, deleteOutOfScope.RowsAffected)
	}
}

func TestGuardAppliesToRowQuery(t *testing.T) {
	db := openTransactionTestDB(t)
	seedGuardedTodos(t, db)
	scoped := ApplyScope(db, singleScope()).Model(&guardedTodo{})

	var title string
	if err := scoped.Select("title").Where("id = ?", "t1-row").Row().Scan(&title); err != nil || title != "tenant-one" {
		t.Fatalf("in-scope Row title=%q error=%v", title, err)
	}
	if err := scoped.Select("title").Where("id = ?", "t2-row").Row().Scan(&title); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("out-of-scope Row must be hidden: %v", err)
	}
}

func seedGuardedTodos(t *testing.T, db *gorm.DB) {
	t.Helper()
	rows := []guardedTodo{
		{Id: "t1-row", TenantId: "t1", OrganizationId: "o1", Title: "tenant-one"},
		{Id: "t2-row", TenantId: "t2", OrganizationId: "o2", Title: "tenant-two"},
	}
	if err := BypassTenantGuard(db).Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
}

// TenantIds が空のスコープも「スコープなし」として拒否される（AllTenants でない限り）。
func TestGuardRejectsEmptyTenantIds(t *testing.T) {
	db := openTestDB(t)
	scoped := ApplyScope(db, &Scope{})
	var list []guardedTodo
	tx := scoped.Find(&list)
	if tx.Error == nil || !strings.Contains(tx.Error.Error(), "tenant scope is required") {
		t.Fatalf("expected 'tenant scope is required', got %v", tx.Error)
	}
}

// AllTenants（システム管理者）はテナント/organization 条件が一切注入されない。
func TestAllTenantsBypassesInjection(t *testing.T) {
	db := openTestDB(t)
	scoped := ApplyScope(db, &Scope{AllTenants: true})

	var todos []guardedTodo
	tx := scoped.Find(&todos)
	if tx.Error != nil {
		t.Fatalf("query: %v", tx.Error)
	}
	sql := tx.Statement.SQL.String()
	if strings.Contains(sql, "tenant_id") || strings.Contains(sql, "organization_id") || strings.Contains(sql, "1 = 0") {
		t.Fatalf("AllTenants must not inject conditions: %s", sql)
	}

	var orgs []guardedOrganization
	tx = scoped.Find(&orgs)
	if tx.Error != nil {
		t.Fatalf("org query: %v", tx.Error)
	}
	if sql := tx.Statement.SQL.String(); strings.Contains(sql, "tenant_id") || strings.Contains(sql, "1 = 0") {
		t.Fatalf("AllTenants must not inject conditions on OrgSelfScoped: %s", sql)
	}
}

// 複数テナント権限は tenant_id IN (...) が注入される。
func TestMultiTenantInjectsInClause(t *testing.T) {
	db := openTestDB(t)
	scoped := ApplyScope(db, &Scope{TenantIds: []string{"t1", "t2"}, OrgIds: []string{"o1"}})

	var list []guardedTodo
	tx := scoped.Find(&list)
	if tx.Error != nil {
		t.Fatalf("query: %v", tx.Error)
	}
	sql := tx.Statement.SQL.String()
	if !strings.Contains(sql, "tenant_id") || !strings.Contains(strings.ToUpper(sql), "IN") {
		t.Fatalf("expected tenant_id IN clause: %s", sql)
	}
	if !containsVar(tx.Statement.Vars, "t1") || !containsVar(tx.Statement.Vars, "t2") {
		t.Fatalf("expected both tenant ids in vars: %v", tx.Statement.Vars)
	}
}

// OrgSelfScopedModel（organization テーブル自身）は主キー IN (OrgIds) が注入される。
func TestOrgSelfScopedInjectsIdFilter(t *testing.T) {
	db := openTestDB(t)
	scoped := ApplyScope(db, &Scope{TenantIds: []string{"t1"}, OrgIds: []string{"o1", "o2"}})

	var orgs []guardedOrganization
	tx := scoped.Find(&orgs)
	if tx.Error != nil {
		t.Fatalf("query: %v", tx.Error)
	}
	sql := tx.Statement.SQL.String()
	if !strings.Contains(sql, "tenant_id") {
		t.Fatalf("expected tenant filter: %s", sql)
	}
	if !containsVar(tx.Statement.Vars, "o1") || !containsVar(tx.Statement.Vars, "o2") {
		t.Fatalf("expected org ids injected for OrgSelfScoped: %s vars=%v", sql, tx.Statement.Vars)
	}

	// OrgIds が空なら 1 = 0 で何も見えない
	none := ApplyScope(db, &Scope{TenantIds: []string{"t1"}})
	tx = none.Find(&orgs)
	if tx.Error != nil {
		t.Fatalf("query: %v", tx.Error)
	}
	if sql := tx.Statement.SQL.String(); !strings.Contains(sql, "1 = 0") {
		t.Fatalf("expected 1 = 0 for empty OrgIds: %s", sql)
	}
}

// Create: 単一テナントスコープなら tenant_id を自動セット。スコープ外 org は拒否。
func TestCreateAutoSetsTenantForSingleScope(t *testing.T) {
	db := openTestDB(t)
	scoped := ApplyScope(db, singleScope())

	ent := &guardedTodo{Id: "id-1", OrganizationId: "o1", Title: "x"}
	tx := scoped.Create(ent)
	if tx.Error != nil {
		t.Fatalf("create: %v", tx.Error)
	}
	if ent.TenantId != "t1" {
		t.Fatalf("tenant_id must be auto-set, got %q", ent.TenantId)
	}

	bad := &guardedTodo{Id: "id-2", OrganizationId: "o-out", Title: "x"}
	tx = scoped.Create(bad)
	if tx.Error == nil || !strings.Contains(tx.Error.Error(), "organization is out of scope") {
		t.Fatalf("expected org out of scope, got %v", tx.Error)
	}
}

// Create: 複数テナント/AllTenants では tenant_id の明示指定が必須。スコープ外テナントは拒否。
func TestCreateRequiresExplicitTenantWhenAmbiguous(t *testing.T) {
	db := openTestDB(t)

	multi := ApplyScope(db, &Scope{TenantIds: []string{"t1", "t2"}, OrgIds: []string{"o1"}})
	tx := multi.Create(&guardedTodo{Id: "id-1", OrganizationId: "o1"})
	if tx.Error == nil || !strings.Contains(tx.Error.Error(), "tenant_id is required") {
		t.Fatalf("expected tenant_id required, got %v", tx.Error)
	}
	tx = multi.Create(&guardedTodo{Id: "id-2", TenantId: "t9", OrganizationId: "o1"})
	if tx.Error == nil || !strings.Contains(tx.Error.Error(), "tenant is out of scope") {
		t.Fatalf("expected tenant out of scope, got %v", tx.Error)
	}
	tx = multi.Create(&guardedTodo{Id: "id-3", TenantId: "t2", OrganizationId: "o1"})
	if tx.Error != nil {
		t.Fatalf("explicit in-scope tenant must pass: %v", tx.Error)
	}

	all := ApplyScope(db, &Scope{AllTenants: true})
	tx = all.Create(&guardedTodo{Id: "id-4", OrganizationId: "o1"})
	if tx.Error == nil || !strings.Contains(tx.Error.Error(), "tenant_id is required") {
		t.Fatalf("AllTenants create without tenant must fail, got %v", tx.Error)
	}
	tx = all.Create(&guardedTodo{Id: "id-5", TenantId: "t9", OrganizationId: "o9"})
	if tx.Error != nil {
		t.Fatalf("AllTenants create with explicit tenant must pass: %v", tx.Error)
	}
}

// AssertScopedModels: マーカー付け忘れ（tenant_id あり・宣言なし）を検出し、例外指定でスキップできる。
func TestAssertScopedModels(t *testing.T) {
	if err := AssertScopedModels(nil, &guardedTodo{}, &guardedOrganization{}, &tenantOnlyRecord{}); err != nil {
		t.Fatalf("valid models must pass: %v", err)
	}
	err := AssertScopedModels(nil, &guardedTodo{}, &plainUser{})
	if err == nil || !strings.Contains(err.Error(), "plainUser") {
		t.Fatalf("expected plainUser to be reported, got %v", err)
	}
	if err := AssertScopedModels([]any{&plainUser{}}, &guardedTodo{}, &plainUser{}); err != nil {
		t.Fatalf("exception must skip check: %v", err)
	}
}

// CanSeeTenant / CanSeeOrg のヘルパー挙動。
func TestScopeHelpers(t *testing.T) {
	s := &Scope{TenantIds: []string{"t1"}, OrgIds: []string{"o1"}}
	if !s.CanSeeTenant("t1") || s.CanSeeTenant("t2") || s.CanSeeTenant("") {
		t.Fatal("CanSeeTenant single scope")
	}
	if !s.CanSeeOrg("o1") || s.CanSeeOrg("o2") {
		t.Fatal("CanSeeOrg single scope")
	}
	all := &Scope{AllTenants: true}
	if !all.CanSeeTenant("t9") || !all.CanSeeOrg("o9") || all.CanSeeTenant("") {
		t.Fatal("AllTenants helpers")
	}
	var nilScope *Scope
	if nilScope.CanSeeTenant("t1") || nilScope.CanSeeOrg("o1") {
		t.Fatal("nil scope must deny")
	}
}
