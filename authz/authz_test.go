package gw_authz

import (
	"testing"

	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
)

// DryRun の DummyDialector を adapter に使う（LoadPolicy は空、AddPolicy の書き込みは
// 実行されないが enforcer のインメモリ状態には反映されるため、判定ロジックはフルにテストできる）。
func setupEnforcer(t *testing.T) {
	t.Helper()
	db, err := gorm.Open(tests.DummyDialector{}, &gorm.Config{DryRun: true})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := Init(db); err != nil {
		t.Fatalf("Init: %v", err)
	}

	// テンプレートと同じシード内容
	for _, g := range [][2]string{{"system_admin", "admin"}, {"admin", "manager"}, {"manager", "user"}} {
		if err := AddRoleInheritance(g[0], g[1]); err != nil {
			t.Fatalf("AddRoleInheritance: %v", err)
		}
	}
	for _, p := range [][4]string{
		{"user", "*", "todo", "create"},
		{"user", "*", "todo", "update_own"},
		{"user", "*", "todo", "delete_own"},
		{"admin", "*", "todo", "update_any"},
		{"admin", "*", "todo", "delete_any"},
		{"system_admin", "*", "user", "manage"},
	} {
		if err := AddPolicy(p[0], p[1], p[2], p[3]); err != nil {
			t.Fatalf("AddPolicy: %v", err)
		}
	}
}

// 受け入れ基準A: ロール×行為×own/any
func TestCanRoleActionMatrix(t *testing.T) {
	setupEnforcer(t)

	cases := []struct {
		role, dom, obj, act string
		want                bool
	}{
		{"user", "t1", "todo", "create", true},        // A-1系: userは作成可
		{"user", "t1", "todo", "update_own", true},    // A-1: 自分のは更新可
		{"user", "t1", "todo", "update_any", false},   // A-2: 他人のは不可
		{"admin", "t1", "todo", "update_any", true},   // A-3: adminは他人のも可
		{"manager", "t1", "todo", "update_any", false},// A-4: managerは不可（階層は下位方向のみ）
		{"admin", "t1", "todo", "update_own", true},   // A-5: 階層継承（admin⊃user）
		{"manager", "t1", "todo", "create", true},     // 階層継承（manager⊃user）
		{"system_admin", "t1", "todo", "delete_any", true}, // 階層で admin の許可も継承
		{"system_admin", "*", "user", "manage", true},
		{"admin", "*", "user", "manage", false},       // 管理APIはsystem_adminのみ
	}
	for _, c := range cases {
		if got := Can(c.role, c.dom, c.obj, c.act); got != c.want {
			t.Errorf("Can(%s, %s, %s, %s) = %v, want %v", c.role, c.dom, c.obj, c.act, got, c.want)
		}
	}
}

// 受け入れ基準C: テナント個別の上書き（domain）と実行時変更
func TestCanTenantOverride(t *testing.T) {
	setupEnforcer(t)

	if Can("manager", "tenant-1", "todo", "delete_any") {
		t.Fatal("前提: managerはdelete_any不可のはず")
	}
	// C-1: tenant-1 だけ manager に delete_any を付与
	if err := AddPolicy("manager", "tenant-1", "todo", "delete_any"); err != nil {
		t.Fatalf("AddPolicy: %v", err)
	}
	if !Can("manager", "tenant-1", "todo", "delete_any") {
		t.Error("C-1: tenant-1のmanagerはdelete_any可になるはず")
	}
	// C-2: 他テナントは不変
	if Can("manager", "tenant-2", "todo", "delete_any") {
		t.Error("C-2: tenant-2のmanagerは不可のまま")
	}
	// C-3: 取り消しで元に戻る
	if err := RemovePolicy("manager", "tenant-1", "todo", "delete_any"); err != nil {
		t.Fatalf("RemovePolicy: %v", err)
	}
	if Can("manager", "tenant-1", "todo", "delete_any") {
		t.Error("C-3: 削除後は403に戻るはず")
	}
}

// 受け入れ基準D: デフォルト拒否
func TestCanDefaultDeny(t *testing.T) {
	setupEnforcer(t)

	if Can("admin", "t1", "todo", "export") { // D-1: 未定義の行為
		t.Error("未定義の行為は拒否")
	}
	if Can("viewer", "t1", "todo", "create") { // D-2: 未定義のロール
		t.Error("未定義のロールは拒否")
	}
	if Can("user", "t1", "report", "create") { // 未定義のリソース種
		t.Error("未定義のリソース種は拒否")
	}
}

// Init前はfail-closed
func TestCanBeforeInitIsDenied(t *testing.T) {
	enforcer = nil
	if Can("admin", "t1", "todo", "update_any") {
		t.Error("Init前は常にfalse")
	}
}
