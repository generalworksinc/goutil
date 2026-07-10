# gw_authz — casbin による行為認可

「**この行為をしてよいか**」の Yes/No 判定を提供する。RBAC with domains（domain=テナントID）。

## 責務境界（最重要）

| 問い | 担当 |
| --- | --- |
| どの行が見えるか（tenant/organization の絞り込み） | **gw_gorm のテナントガード**（このパッケージではない） |
| この行為をしてよいか | **gw_authz** |

- 可視範囲の計算・フィルタを casbin に入れてはならない（ポリシーと組織ツリーの二重管理になる）
- ユーザー→ロールの割当ても casbin に持たせない（アプリの User.Role 等が真実の源）。
  casbin の sub には**ロール名文字列**を渡す。g ルールは**ロール階層のみ**に使う

## セットアップ

```go
// 起動時（DB初期化後）に1回
gw_authz.Init(db)  // casbin_rule テーブルからロード。以後の判定はインメモリ

// リクエストからロール名/テナントIDを取り出す関数を登録（Require が使う）
gw_authz.SetRoleResolver(func(c *gw_web.WebCtx) (string, string, bool) {
    scope, ok := c.Locals("Scope").(*access.AccessScope)
    if !ok { return "", "", false }
    tenantId := ""
    if len(scope.TenantIds) > 0 { tenantId = scope.TenantIds[0] }
    return scope.Role.Name(), tenantId, true
})
```

テーブルはマイグレーションで `gw_authz.CasbinRule` を作成する。永続化は gorm.io/gorm だけに依存する
自前アダプタで行う（公式 gorm-adapter は全 DB ドライバを import するため不採用）。

## ポリシー

```go
// ロール階層（上位が下位の許可を継承）
gw_authz.AddRoleInheritance("admin", "manager")

// p ルール: role, dom("*"=全テナント), obj, act — マッチしなければデフォルト拒否
gw_authz.AddPolicy("user",  "*", "todo", "update_own")
gw_authz.AddPolicy("admin", "*", "todo", "update_any")

// テナント個別の上書き（実行時に追加/削除可能。DBへ自動保存）
gw_authz.AddPolicy("manager", tenantId, "todo", "delete_any")
```

## 判定の3つの入口

```go
// 1) ルート/グループのゲート（リソース実体に依存しない行為）
systemGroup := app.Group("/api/system", AuthSystemAdmin(), gw_authz.Require("user", "manage"))

// 2) リソース実体に依存する own/any 合成判定（モデルに OwnedBy/TenantIdOf の1行メソッドを実装）
todo, err := gw_authz.FindEditable[models.Todo](tx, scope.Role.Name(), userId, id, "todo", "update")
// 不在 → (nil, nil) / 権限なし → gw_authz.ErrForbidden（403に変換）

// 3) 任意の場所での直接判定
if gw_authz.Can(role, tenantId, "todo", "create") { ... }
```

## 運用ノート

- 単一インスタンス前提。複数インスタンスでポリシーを動的変更する場合は casbin watcher で同期すること
- 属性条件（時間・金額など）が主役になったら cerbos への乗り換えを検討する（判断基準はテンプレートの docs/authorization.md 参照）

## TODO

- ポーリング自動リロードの標準ヘルパー化: 現在はテンプレート側で `Enforcer().StartAutoLoadPolicy(interval)` を
  フラグ定数つきで呼んでいる（generalworks-template-go-api-project の constant/authz.go + infrastructure/db.go 参照）。
  複数プロダクトで同じ形が繰り返されたら `gw_authz.Init` のオプション（例: `WithAutoReload(interval)`）として取り込む
