# gw_log — slog 構造化ロギング + リクエストID伝搬

`log/slog` の JSON 出力に、context 経由で `request_id` / `user_id` を**自動付与**する仕組みを提供する。
一度セットアップすれば、アクセスログ・SQL ログ・業務ログ・エラーログを 1 つの request_id で横ぐし検索できる。

運用規約・検索例はテンプレート（generalworks-template-go-api-project）の `src/docs/logging.md` を参照。

## セットアップ（アプリ起動時）

```go
gw_log.Init(slog.LevelInfo)      // JSONハンドラ + context注入ハンドラを slog.SetDefault
app.Use(gw_web.RequestId())      // 最初のミドルウェア。X-Request-Id → X-Amzn-Trace-Id → ULID
app.Use(gw_web.AccessLog())      // 1リクエスト1行のアクセスログ
```

- `Init` の時刻は UTC・RFC3339Nano に正規化。`Options{AddSource: true}` で file:line 付与
- SQL ログは `gw_gorm.DefaultConfig(debug)` が `SlogLogger` を使うため自動で構造化される。
  リクエストの request_id を SQL に乗せるには **DB ハンドルに `db.WithContext(ctx)` を通す**こと

## 仕組み

`Init` は JSON ハンドラを contextHandler でラップする。contextHandler は `Handle(ctx, record)` の際に
context から request_id / user_id を読み取り属性として追加する。したがって:

```go
slog.InfoContext(ctx, "user updated", slog.String("target", id))
// → {"msg":"user updated","target":"...","request_id":"...","user_id":"..."}
```

**`slog.Info` ではなく `slog.InfoContext`（ctx 付き）を使うこと**。ctx なしの呼び出しには何も付与されない。

`With` / `WithGroup` を挟んだ logger からのログでも、`request_id` / `user_id` は**常にトップレベル**に出る
（グループ配下に沈んで検索フィールドが分散しないことを保証。テストで固定済み）。

### 内部構造: contextHandler の base / buildSteps / assembled

**前提となる slog の性質**: ハンドラは `WithAttrs` / `WithGroup` で加工されたチェーンであり、
`WithGroup("g")` より後に追加された属性はすべて `g` 配下にネストする。
Handle 時に注入する request_id も「後から追加」なので、素朴なラッパーではグループ配下に沈む:

```jsonc
{"msg": "...", "g": {"request_id": "01J...", "inner": "x"}}   // 素朴なラッパー（検索キーが分散）
{"msg": "...", "request_id": "01J...", "g": {"inner": "x"}}   // この実装（常にトップレベル）
```

トップレベルに出すには「WithGroup が適用される前の地点」に注入する必要がある。
そのために contextHandler はチェーンを 3 つの形で保持する:

```text
contextHandler
├── base       slog.Handler                    // 生成時のハンドラ。WithAttrs/WithGroup 未適用
├── buildSteps []func(slog.Handler) slog.Handler // 適用された WithAttrs/WithGroup の操作列
└── assembled  slog.Handler                    // base に buildSteps を順適用した結果（事前計算）

不変条件: assembled == buildSteps を base に順適用したもの
```

処理は「構築時」と「出力時」の2フェーズ:

```text
[構築時] logger.With(...) / logger.WithGroup(...) が呼ばれるたび
    buildSteps ← buildSteps + [op]      // 履歴に追記
    assembled  ← op(assembled)          // 事前計算を更新（Handle 時には組み立てない）

[出力時] Handle(ctx, record)
    ctx に request_id / user_id が
    ├── 無い → assembled.Handle(ctx, r)             // 事前計算済みをそのまま使用。コスト増ゼロ
    └── 有る → base.WithAttrs([request_id, user_id]) // グループ適用「前」の base に注入
               → buildSteps を順に再適用
               → .Handle(ctx, r)                     // 注入した ID だけがトップレベルに出る
```

ID 有りパスの再組み立てコストは WithAttrs + buildSteps 再適用ぶんのアロケーションのみ。
WithGroup を使わない通常構成では buildSteps は空なので、実質 `base.WithAttrs(ids).Handle(r)` と同等。

## API

- `WithRequestId(ctx, id)` / `RequestIdFromContext(ctx)` — リクエストIDの context 載せ替え/取得
- `WithUserId(ctx, id)` / `UserIdFromContext(ctx)` — 認証ミドルウェアがユーザーIDを載せる
- `NewHandler(inner)` — 任意の slog.Handler を context 注入でラップする低レベルAPI

## 関連コンポーネント

- `gw_web.RequestId()` — リクエストID解決ミドルウェア（レスポンスヘッダ `X-Request-Id` にもエコー）
- `gw_web.AccessLog()` — method/path/status/latency_ms/ip/ua を出力（4xx=WARN, 5xx=ERROR）
- `gw_web.CustomHTTPErrorHandler` — エラー詳細+スタックトレースを slog で出力（5xx=ERROR, その他=WARN）
- `gw_gorm.SlogLogger` — GORM の SQL ログ（elapsed_ms/rows/sql。非 debug はスロークエリ+エラーのみ）
