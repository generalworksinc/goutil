package gw_gorm

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

const (
	tenantScopeKey     = "gw_gorm:tenant_scope"
	tenantScopeSkipKey = "gw_gorm:tenant_scope_skip"
)

// Scope はテナント境界と参照可能 organization を表す。
// - TenantIds: 参照可能テナント。通常ユーザーは所属テナントの1件。複数テナント権限者はN件
// - OrgIds: 参照可能 organization。nil/空 = 何も見えない
// - AllTenants: システム管理者用。テナント/organization 条件の注入を全てスキップする（RLS の BYPASSRLS 相当）。
//   これを true にしてよいのはアプリのスコープ解決処理（Role 判定）1箇所のみ、という規約で運用すること
type Scope struct {
	TenantIds  []string
	OrgIds     []string
	AllTenants bool
}

// CanSeeTenant は tenantId が参照可能テナントに含まれるかを返す。AllTenants なら常に true。
func (s *Scope) CanSeeTenant(tenantId string) bool {
	if s == nil || tenantId == "" {
		return false
	}
	if s.AllTenants {
		return true
	}
	for _, id := range s.TenantIds {
		if id == tenantId {
			return true
		}
	}
	return false
}

// CanSeeOrg は orgId が参照可能 organization に含まれるかを返す。AllTenants なら常に true。
func (s *Scope) CanSeeOrg(orgId string) bool {
	if s == nil || orgId == "" {
		return false
	}
	if s.AllTenants {
		return true
	}
	for _, id := range s.OrgIds {
		if id == orgId {
			return true
		}
	}
	return false
}

// マーカーインターフェース。モデルに中身のない宣言メソッドを1行書くことでガード対象になる。
// - TenantScopedModel: tenant_id 条件（= / IN）を自動注入
// - OrgScopedModel: organization_id IN (OrgIds) を自動注入（Todo など organization に紐づくデータ）
// - OrgSelfScopedModel: 主キー IN (OrgIds) を自動注入（organization テーブル自身の可視範囲）
type TenantScopedModel interface{ TenantScoped() }
type OrgScopedModel interface{ OrgScoped() }
type OrgSelfScopedModel interface{ OrgSelfScoped() }

// WithScope は GORM セッションにテナントスコープを載せる。
// Raw()/Exec() の生 SQL は GORM コールバックを通らないため、このガードの対象外。
// 返り値は Session() 済みの「再利用可能な起点」。変数に取って複数クエリに使い回しても、
// finisher 実行後の条件が次のクエリへ残留しない（スコープは Statement 複製時に伝搬する）。
func WithScope(db *gorm.DB, scope *Scope) *gorm.DB {
	return db.Set(tenantScopeKey, scope).Session(&gorm.Session{})
}

// WithoutTenantScope はスコープ解決処理・管理バッチ・シードなどで明示的にガードを外す。
// Raw()/Exec() の生 SQL は GORM コールバックを通らないため、このガードの対象外。
// WithScope と同様、返り値は再利用可能な起点として扱える。
func WithoutTenantScope(db *gorm.DB) *gorm.DB {
	return db.Set(tenantScopeSkipKey, true).Session(&gorm.Session{})
}

// ScopeFrom は WithScope でセッションに載せたスコープを取り出す。
// repository が業務判定（CanSeeOrg 等）にスコープの中身を使いたい場合に、引数で別途受け取らずに済む。
func ScopeFrom(db *gorm.DB) (*Scope, bool) {
	return getScope(db)
}

// UseTenantGuard は TenantScopedModel/OrgScopedModel/OrgSelfScopedModel への GORM 操作にスコープ条件を強制する。
// Raw()/Exec() の生 SQL は GORM コールバックを通らないため、このガードの対象外。
func UseTenantGuard(db *gorm.DB) error {
	if db.Callback().Query().Get("gw_gorm:tenant_guard_query") == nil {
		if err := db.Callback().Query().Before("gorm:query").Register("gw_gorm:tenant_guard_query", tenantGuardQuery); err != nil {
			return err
		}
	}
	if db.Callback().Update().Get("gw_gorm:tenant_guard_update") == nil {
		if err := db.Callback().Update().Before("gorm:update").Register("gw_gorm:tenant_guard_update", tenantGuardQuery); err != nil {
			return err
		}
	}
	if db.Callback().Delete().Get("gw_gorm:tenant_guard_delete") == nil {
		if err := db.Callback().Delete().Before("gorm:delete").Register("gw_gorm:tenant_guard_delete", tenantGuardQuery); err != nil {
			return err
		}
	}
	if db.Callback().Row().Get("gw_gorm:tenant_guard_row") == nil {
		if err := db.Callback().Row().Before("gorm:row").Register("gw_gorm:tenant_guard_row", tenantGuardQuery); err != nil {
			return err
		}
	}
	if db.Callback().Create().Get("gw_gorm:tenant_guard_create") == nil {
		if err := db.Callback().Create().Before("gorm:create").Register("gw_gorm:tenant_guard_create", tenantGuardCreate); err != nil {
			return err
		}
	}
	return nil
}

func tenantGuardQuery(db *gorm.DB) {
	if shouldSkip(db) || db.Statement == nil || db.Statement.Schema == nil {
		return
	}
	stmt := db.Statement
	tenantScoped := implementsTenantScoped(stmt.Schema)
	orgScoped := implementsOrgScoped(stmt.Schema)
	orgSelfScoped := implementsOrgSelfScoped(stmt.Schema)
	if !tenantScoped && !orgScoped && !orgSelfScoped {
		return
	}
	scope, ok := getScope(db)
	if !ok || scope == nil || (!scope.AllTenants && len(scope.TenantIds) == 0) {
		stmt.AddError(errors.New("tenant scope is required"))
		return
	}
	if scope.AllTenants {
		// システム管理者: 条件注入なし（全テナント横断）
		return
	}
	if tenantScoped {
		if len(scope.TenantIds) == 1 {
			stmt.AddClause(clause.Where{Exprs: []clause.Expression{
				clause.Eq{Column: clause.Column{Table: clause.CurrentTable, Name: "tenant_id"}, Value: scope.TenantIds[0]},
			}})
		} else {
			stmt.AddClause(clause.Where{Exprs: []clause.Expression{
				clause.IN{Column: clause.Column{Table: clause.CurrentTable, Name: "tenant_id"}, Values: toAnySlice(scope.TenantIds)},
			}})
		}
	}
	if orgScoped {
		addOrgInClause(stmt, "organization_id", scope.OrgIds)
	}
	if orgSelfScoped {
		addOrgInClause(stmt, primaryColumnName(stmt.Schema), scope.OrgIds)
	}
}

func addOrgInClause(stmt *gorm.Statement, column string, orgIds []string) {
	if len(orgIds) == 0 {
		stmt.AddClause(clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "1 = 0"}}})
		return
	}
	stmt.AddClause(clause.Where{Exprs: []clause.Expression{
		clause.IN{Column: clause.Column{Table: clause.CurrentTable, Name: column}, Values: toAnySlice(orgIds)},
	}})
}

func tenantGuardCreate(db *gorm.DB) {
	if shouldSkip(db) || db.Statement == nil || db.Statement.Schema == nil {
		return
	}
	stmt := db.Statement
	tenantScoped := implementsTenantScoped(stmt.Schema)
	orgScoped := implementsOrgScoped(stmt.Schema)
	if !tenantScoped && !orgScoped {
		return
	}
	scope, ok := getScope(db)
	if !ok || scope == nil || (!scope.AllTenants && len(scope.TenantIds) == 0) {
		stmt.AddError(errors.New("tenant scope is required"))
		return
	}
	applyToReflectValues(stmt, func(v reflect.Value) {
		if tenantScoped {
			// 参照可能テナントが1件に確定する場合のみ自動セット。
			// 複数テナント権限者や AllTenants（システム管理者）は「どのテナントに作るのか」が
			// 確定しないため、呼び出し側での明示指定を必須にする。
			if !scope.AllTenants && len(scope.TenantIds) == 1 {
				setStringIfEmpty(db, v, "tenant_id", scope.TenantIds[0])
			}
			tenantId := getStringValue(db, v, "tenant_id")
			if tenantId == "" {
				stmt.AddError(errors.New("tenant_id is required"))
			} else if !scope.CanSeeTenant(tenantId) {
				stmt.AddError(errors.New("tenant is out of scope"))
			}
		}
		if orgScoped {
			orgId := getStringValue(db, v, "organization_id")
			if !scope.CanSeeOrg(orgId) {
				stmt.AddError(errors.New("organization is out of scope"))
			}
		}
	})
}

// AssertScopedModels はガードのマーカー付け忘れを起動時に検出する。
// 「tenant_id カラムを持つのに TenantScopedModel を実装していない」モデル（およびその逆）をエラーにする。
// exceptions には意図的にガード対象外とするモデル（例: ログイン時にスコープ確立前へ検索が必要な User）を渡す。
// アプリの初期化（InitDB 等)で全モデルを渡して呼び、エラーなら起動を中断すること。
func AssertScopedModels(exceptions []any, models ...any) error {
	exceptionTypes := map[reflect.Type]bool{}
	for _, e := range exceptions {
		exceptionTypes[indirectType(reflect.TypeOf(e))] = true
	}
	cache := &sync.Map{}
	namer := schema.NamingStrategy{SingularTable: true}
	var problems []string
	for _, m := range models {
		s, err := schema.Parse(m, cache, namer)
		if err != nil {
			return err
		}
		if exceptionTypes[s.ModelType] {
			continue
		}
		hasTenantCol := s.LookUpField("tenant_id") != nil
		isTenantScoped := implementsTenantScoped(s)
		if hasTenantCol && !isTenantScoped {
			problems = append(problems, fmt.Sprintf("%s: has tenant_id column but does not implement TenantScopedModel (missing marker?)", s.ModelType.Name()))
		}
		if !hasTenantCol && isTenantScoped {
			problems = append(problems, fmt.Sprintf("%s: implements TenantScopedModel but has no tenant_id column", s.ModelType.Name()))
		}
	}
	if len(problems) > 0 {
		return errors.New("scoped model assertion failed:\n  " + strings.Join(problems, "\n  "))
	}
	return nil
}

func getScope(db *gorm.DB) (*Scope, bool) {
	v, ok := db.Get(tenantScopeKey)
	if !ok {
		return nil, false
	}
	scope, ok := v.(*Scope)
	return scope, ok
}

func shouldSkip(db *gorm.DB) bool {
	v, ok := db.Get(tenantScopeSkipKey)
	if !ok {
		return false
	}
	skip, _ := v.(bool)
	return skip
}

func implementsTenantScoped(s *schema.Schema) bool {
	return implements[TenantScopedModel](s)
}

func implementsOrgScoped(s *schema.Schema) bool {
	return implements[OrgScopedModel](s)
}

func implementsOrgSelfScoped(s *schema.Schema) bool {
	return implements[OrgSelfScopedModel](s)
}

func implements[T any](s *schema.Schema) bool {
	if s == nil || s.ModelType == nil {
		return false
	}
	v := reflect.New(s.ModelType).Interface()
	_, ok := v.(T)
	return ok
}

func primaryColumnName(s *schema.Schema) string {
	if s != nil && s.PrioritizedPrimaryField != nil {
		return s.PrioritizedPrimaryField.DBName
	}
	return "id"
}

func toAnySlice(values []string) []interface{} {
	out := make([]interface{}, len(values))
	for i, v := range values {
		out[i] = v
	}
	return out
}

func indirectType(t reflect.Type) reflect.Type {
	for t != nil && t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func applyToReflectValues(stmt *gorm.Statement, fn func(reflect.Value)) {
	v := stmt.ReflectValue
	if !v.IsValid() {
		return
	}
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			item := v.Index(i)
			for item.Kind() == reflect.Ptr {
				if item.IsNil() {
					break
				}
				item = item.Elem()
			}
			if item.IsValid() && item.Kind() == reflect.Struct {
				fn(item)
			}
		}
	case reflect.Struct:
		fn(v)
	}
}

func setStringIfEmpty(db *gorm.DB, v reflect.Value, dbName string, value string) {
	field := db.Statement.Schema.LookUpField(dbName)
	if field == nil {
		return
	}
	_, zero := field.ValueOf(db.Statement.Context, v)
	if zero {
		db.Statement.AddError(field.Set(db.Statement.Context, v, value))
	}
}

func getStringValue(db *gorm.DB, v reflect.Value, dbName string) string {
	field := db.Statement.Schema.LookUpField(dbName)
	if field == nil {
		return ""
	}
	value, _ := field.ValueOf(db.Statement.Context, v)
	str, _ := value.(string)
	return str
}
