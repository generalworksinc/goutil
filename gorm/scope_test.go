package gw_gorm

import (
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

func (guardedOrganization) TenantScoped() {}
func (guardedOrganization) OrgSelfScoped() {}

type plainUser struct {
	Id       string
	TenantId string
	Email    string
}

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

// WithScope の返り値を変数に取って複数クエリに使い回しても、
// 前のクエリの条件が次のクエリに残留しない（ステートメント汚染防止）。
func TestWithScopeReuseDoesNotPolluteStatement(t *testing.T) {
	db := openTestDB(t)
	scoped := WithScope(db, singleScope())

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
func TestWithScopePropagatesAcrossReuse(t *testing.T) {
	db := openTestDB(t)
	scoped := WithScope(db, &Scope{TenantIds: []string{"t1"}, OrgIds: []string{"o1", "o2"}})

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

// WithoutTenantScope も再利用可能な起点として振る舞い、skip 指定が使い回し後も効く。
func TestWithoutTenantScopeReuse(t *testing.T) {
	db := openTestDB(t)
	free := WithoutTenantScope(db)

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

// スコープ未設定のままガード対象モデルを触ると拒否される。
func TestGuardRejectsWithoutScope(t *testing.T) {
	db := openTestDB(t)
	var list []guardedTodo
	tx := db.Find(&list)
	if tx.Error == nil || !strings.Contains(tx.Error.Error(), "tenant scope is required") {
		t.Fatalf("expected 'tenant scope is required', got %v", tx.Error)
	}
}

// TenantIds が空のスコープも「スコープなし」として拒否される（AllTenants でない限り）。
func TestGuardRejectsEmptyTenantIds(t *testing.T) {
	db := openTestDB(t)
	scoped := WithScope(db, &Scope{})
	var list []guardedTodo
	tx := scoped.Find(&list)
	if tx.Error == nil || !strings.Contains(tx.Error.Error(), "tenant scope is required") {
		t.Fatalf("expected 'tenant scope is required', got %v", tx.Error)
	}
}

// AllTenants（システム管理者）はテナント/organization 条件が一切注入されない。
func TestAllTenantsBypassesInjection(t *testing.T) {
	db := openTestDB(t)
	scoped := WithScope(db, &Scope{AllTenants: true})

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
	scoped := WithScope(db, &Scope{TenantIds: []string{"t1", "t2"}, OrgIds: []string{"o1"}})

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
	scoped := WithScope(db, &Scope{TenantIds: []string{"t1"}, OrgIds: []string{"o1", "o2"}})

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
	none := WithScope(db, &Scope{TenantIds: []string{"t1"}})
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
	scoped := WithScope(db, singleScope())

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

	multi := WithScope(db, &Scope{TenantIds: []string{"t1", "t2"}, OrgIds: []string{"o1"}})
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

	all := WithScope(db, &Scope{AllTenants: true})
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
	if err := AssertScopedModels(nil, &guardedTodo{}, &guardedOrganization{}); err != nil {
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
