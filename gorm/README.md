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
scoped := gw_gorm.WithScope(db, &gw_gorm.Scope{...}) // リクエスト毎にスコープを載せる（返り値は再利用可能な起点）
```

- `Scope{TenantIds, OrgIds, AllTenants}`。`AllTenants` はシステム管理者用の全条件スキップ（BYPASSRLS 相当）
- スコープ未設定でガード対象モデルに触ると `tenant scope is required` で**拒否される（fail-closed）**
- Create は TenantIds が1件なら tenant_id を自動セット。複数/AllTenants は明示必須。organization_id はスコープ内検証
- `WithoutTenantScope(db)` はスコープ解決・シード・管理バッチ等の明示的なガード除外（使用箇所は grep で監査可能に保つ）
- `ScopeFrom(db)` でセッションからスコープを取り出せる（repository の業務判定用）
- `Raw()` / `Exec()` の生 SQL はコールバックを通らないためガード対象外
- `AssertScopedModels(exceptions, models...)` を起動時に呼ぶと「tenant_id カラムがあるのにマーカー未実装」を検出できる（マーカー付け忘れ対策）

## contextとtransactionへのScope伝搬

HTTP middlewareなどで確定したScopeをrepositoryまで伝搬する場合は、Scopeを個別引数にせずcontextへ保存できる。

```go
ctx = gw_gorm.WithScopeContext(ctx, &gw_gorm.Scope{
    TenantIds: []string{"tenant-a"},
    OrgIds:    []string{"org-a"},
})

db := gw_gorm.PickConnectionIfEmpty(ctx, explicitDB, defaultDB)
err := gw_gorm.WithTx(ctx, defaultDB, func(tx *gorm.DB) error {
    return tx.Create(&row).Error
})
```

- `WithScopeContext` と `ScopeFromContext` はScopeを複製し、外部からのslice変更による認可範囲の変化を防ぐ
- `PickConnectionIfEmpty` は明示DBを優先する。明示DBは既にtransactionや意図的な管理用DBである可能性があるため、contextのScopeを暗黙に上書きしない
- 明示DBがnilなら既定DBへcontextのScopeを適用する。Scopeがなければ通常DBとなり、ガード対象モデルへの操作は `UseTenantGuard` が拒否する
- `WithTx` はScope付きの既定DBからtransactionを開始する。Scopeなしでも開始自体は許可するが、ガード対象操作はfail-closedになる
- goutilはアプリケーションのグローバル接続を保持しない。`defaultDB`は呼び出し側が管理する
- `Raw()` / `Exec()` はtransaction内でもTenant Guard対象外なので、ユーザー入力に基づく処理では使用しない

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
