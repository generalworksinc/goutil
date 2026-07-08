package gw_gorm

import (
	"errors"
	"reflect"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

const (
	tenantScopeKey     = "gw_gorm:tenant_scope"
	tenantScopeSkipKey = "gw_gorm:tenant_scope_skip"
)

// Scope はテナント境界と参照可能 organization を表す。
type Scope struct {
	TenantId string
	OrgIds   []string // 可視 organization。nil/空 = 何も見えない
}

func (s *Scope) CanSeeOrg(orgId string) bool {
	if s == nil || orgId == "" {
		return false
	}
	for _, id := range s.OrgIds {
		if id == orgId {
			return true
		}
	}
	return false
}

type TenantScopedModel interface{ TenantScoped() }
type OrgScopedModel interface{ OrgScoped() }

// WithScope は GORM セッションにテナントスコープを載せる。
// Raw()/Exec() の生 SQL は GORM コールバックを通らないため、このガードの対象外。
func WithScope(db *gorm.DB, scope *Scope) *gorm.DB {
	return db.Set(tenantScopeKey, scope)
}

// WithoutTenantScope はスコープ解決処理・管理バッチ・シードなどで明示的にガードを外す。
// Raw()/Exec() の生 SQL は GORM コールバックを通らないため、このガードの対象外。
func WithoutTenantScope(db *gorm.DB) *gorm.DB {
	return db.Set(tenantScopeSkipKey, true)
}

// UseTenantGuard は TenantScopedModel/OrgScopedModel への GORM 操作にスコープ条件を強制する。
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
	if !implementsTenantScoped(stmt.Schema) {
		return
	}
	scope, ok := getScope(db)
	if !ok || scope == nil || scope.TenantId == "" {
		stmt.AddError(errors.New("tenant scope is required"))
		return
	}
	stmt.AddClause(clause.Where{Exprs: []clause.Expression{
		clause.Eq{Column: clause.Column{Table: clause.CurrentTable, Name: "tenant_id"}, Value: scope.TenantId},
	}})
	if !implementsOrgScoped(stmt.Schema) {
		return
	}
	if len(scope.OrgIds) == 0 {
		stmt.AddClause(clause.Where{Exprs: []clause.Expression{clause.Expr{SQL: "1 = 0"}}})
		return
	}
	values := make([]interface{}, len(scope.OrgIds))
	for i, id := range scope.OrgIds {
		values[i] = id
	}
	stmt.AddClause(clause.Where{Exprs: []clause.Expression{
		clause.IN{Column: clause.Column{Table: clause.CurrentTable, Name: "organization_id"}, Values: values},
	}})
}

func tenantGuardCreate(db *gorm.DB) {
	if shouldSkip(db) || db.Statement == nil || db.Statement.Schema == nil {
		return
	}
	stmt := db.Statement
	if !implementsTenantScoped(stmt.Schema) {
		return
	}
	scope, ok := getScope(db)
	if !ok || scope == nil || scope.TenantId == "" {
		stmt.AddError(errors.New("tenant scope is required"))
		return
	}
	applyToReflectValues(stmt, func(v reflect.Value) {
		setStringIfEmpty(db, v, "tenant_id", scope.TenantId)
		if implementsOrgScoped(stmt.Schema) {
			orgId := getStringValue(db, v, "organization_id")
			if !scope.CanSeeOrg(orgId) {
				stmt.AddError(errors.New("organization is out of scope"))
			}
		}
	})
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

func implements[T any](s *schema.Schema) bool {
	if s == nil || s.ModelType == nil {
		return false
	}
	v := reflect.New(s.ModelType).Interface()
	_, ok := v.(T)
	return ok
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
