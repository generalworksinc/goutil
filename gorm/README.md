# gw_gorm — GORM 共通ユーティリティ

generalworks 標準の GORM 設定・ベースモデル・マルチテナントガード・クエリヘルパーを提供する。
DB ドライバ（Dialector）と DSN の組み立てはアプリ側の責務であり、**このパッケージは特定の DB に依存しない**。

実運用の設計・規約の全体像は Go API テンプレート（generalworks-template-go-api-project）の
`src/docs/orm.md` / `src/docs/authorization.md` を参照。

## DefaultConfig / ReCreateTable

```go
db, err := gorm.Open(postgres.Open(dsn), gw_gorm.DefaultConfig(debug))
```

- テーブル名は単数形 snake_case（`SingularTable: true`。`TableName()` の実装は不要）
- FK 制約を DDL に含めない（データを pure に保つ方針。リレーションはタグの論理 FK で扱う）
- タイムスタンプは UTC。debug 時のみ SQL ログ出力
- `ReCreateTable(db, models...)` は開発・デモ用の破壊的な DROP → AutoMigrate

## ベースモデル

`BaseModel` / `BaseModelLogicalDel`（gorm.DeletedAt）/ `BaseModelByManualId`（±LogicalDel）/
`BaseModelSimple` と、それぞれの Ulid 版（時刻ソート可能な ULID を BeforeCreate で自動採番）。
UUID/ULID は **Id が空のときだけ** BeforeCreate フックで自動セットされる。

**`BaseModelByManualId` は Id を含まない**（CreatedAt/UpdatedAt のみ）。「Id はモデル側が自前定義する」
バリエーション。使い分けの目安:
- ID を必ず明示セットする運用でも、通常は `BaseModel`（自動採番は Id が空のときだけの no-op）で足りる
- Id のセット忘れを「ランダム UUID で黙って埋まる」のではなく「空PKエラーで即検出」したい特殊なモデルや、
  Id の型・タグを変えたいモデルだけ `BaseModelByManualId` + 自前 Id 定義を使う

## テナントガード

対象モデルは**マーカーインターフェース**（中身のない宣言メソッド1行）で明示する:

| マーカー | 対象 | 自動注入される条件 |
| --- | --- | --- |
| `TenantScoped()` | tenant 境界で分離するテーブル | `tenant_id = ?` / `IN (TenantIds)` |
| `OrgScoped()` | organization 可視範囲で絞るテーブル | `organization_id IN (OrgIds)` |
| `OrgSelfScoped()` | organization テーブル自身 | 主キー `IN (OrgIds)` |

```go
gw_gorm.UseTenantGuard(db)                          // 起動時に1回登録
scoped := gw_gorm.ApplyScope(db, &gw_gorm.Scope{...}) // リクエスト毎にスコープを載せる（返り値は再利用可能な起点）
```

- `Scope{TenantIds, OrgIds, AllTenants}`。`AllTenants` はシステム管理者用の全条件スキップ（BYPASSRLS 相当）
- スコープ未設定でガード対象モデルに触ると `tenant scope is required` で**拒否される（fail-closed）**
- Create は TenantIds が1件なら tenant_id を自動セット。複数/AllTenants は明示必須。organization_id はスコープ内検証
- Update は対象行をScopeで絞り、tenant_id / organization_idの変更先もScope内であることを検証
- 条件なしUpdate/Deleteは、Guardが条件を追加してもGORMの`ErrMissingWhereClause`で拒否
- `BypassTenantGuard(db)` はスコープ解決・シード・管理バッチ等の明示的なガード除外（使用箇所は grep で監査可能に保つ）
- `ApplyScope`と`BypassTenantGuard`は後勝ち。最後に呼んだ設定を最終状態とする
- `Raw()` / `Exec()`の生SQLと、Schemaを持たない`Table()` + map/primitive結果はコールバックによるGuard対象外
- `AssertScopedModels(exceptions, models...)` を起動時に呼ぶと「tenant_id カラムがあるのにマーカー未実装」を検出できる（マーカー付け忘れ対策）

## contextとtransactionへのScope伝搬

HTTP middlewareなどで確定したScopeをrepositoryまで伝搬する場合は、Scopeを個別引数にせずcontextへ保存できる。

```go
ctx = gw_gorm.WithScopeContext(ctx, &gw_gorm.Scope{
    TenantIds: []string{"tenant-a"},
    OrgIds:    []string{"org-a"},
})

db := defaultDB.WithContext(ctx)
err := db.Transaction(func(tx *gorm.DB) error {
    return tx.Create(&row).Error
})
```

- `WithScopeContext` はScopeを複製して保存し、外部からのslice変更による認可範囲の変化を防ぐ。取得処理はTenant Guard内部に限定する
- `AttachScope` は `*gw_web.WebCtx` 専用のFiber連携関数。HTTP middlewareで確定したScopeをリクエストのcontextへ設定する
- HTTPを使わない処理では `WithScopeContext` と `ApplyScope` を利用し、`gw_web` の型を持ち込む必要はない
- Scopeの保存場所は `context.Context` の1箇所だけで、Tenant GuardはGORMの `Statement.Context` から取得する
- 明示DBと既定DBの選択はアプリケーション側の責務。transactionなどの明示DBにはcontextを再設定しない
- transaction wrapperが必要ならアプリケーション側に置き、既定DBへcontextを設定して`Transaction`を呼ぶ
- Scopeを使わないアプリはマーカーを実装せず、Tenantだけを使うアプリは `TenantScoped()` と `TenantIds` だけを利用できる
- `Raw()` / `Exec()` / Schemaなし`Table()`はtransaction内でもTenant Guard対象外なので、利用箇所をgrep・レビューで監査する

## FindOne — 「不在はエラーではない」検索

`First` は 0 件を `ErrRecordNotFound`（合成エラー）にするため、不在があり得る検索では
毎回 `errors.Is` の3分岐が必要になる。`FindOne` は不在を `(nil, nil)` で返し、
**エラーが返るのは接続断・制約違反・スコープ未設定などの予期しない異常だけ**になる。

```go
user, err := gw_gorm.FindOne[models.User](db.Where("email = ?", email))
if err != nil {
    return nil, err // 予期せぬエラーのみ
}
if user == nil {
    return nil, nil // 不在（正常系の分岐）
}
```

使い分け: **不在があり得る検索は `FindOne`、存在しなければバグという検索は `First`**。
