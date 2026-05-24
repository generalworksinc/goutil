Fiber v3 migration チェックリスト

- [ ] 前提確認（環境/依存）
  - [ ] Go バージョンが Fiber v3 要件（Go 1.24+）を満たす
  - [ ] 参照ドキュメントの確認（[What's New in v3](https://docs.gofiber.io/next/whats_new/), [v3 ドキュメント](https://docs.gofiber.io/next/))

- [ ] goutil（`github.com/generalworksinc/goutil`）側の移行
  - [ ] `go.mod` の Fiber を `github.com/gofiber/fiber/v3` に更新し、`go mod tidy` 実行
  - [ ] knowle_api 側の Fiber バージョンと揃える（例: `v3.0.0-beta.5`）
  - [ ] Static 提供方法の変更（app.Static -> 静的ミドルウェア）
    - [ ] `webframework/webframework.go` の `app.Static("/static", "static")` を削除
    - [ ] `github.com/gofiber/fiber/v3/middleware/static` を import
    - [ ] `app.Use(static.New("/static", static.Config{FS: os.DirFS("static")}))` を追加
    - [ ] `/static` 配信の動作確認
    - 参考: [What's New in v3 > App > Static](https://docs.gofiber.io/next/whats_new/)
  - [ ] Session ミドルウェア/ストアの更新
    - [ ] `var store = session.New()` を `session.NewStore()` へ変更
    - [ ] `store.Get(ctx)` で取得したセッションの利用箇所を確認
    - [ ] `sess.Save()` の後に `sess.Release()` を呼ぶよう変更
    - [ ] Cookie/ヘッダ取得の Extractor はストア直利用では不要（必要なら設計）
    - 参考: [What's New in v3 > Middlewares > Session](https://docs.gofiber.io/next/whats_new/)
  - [ ] MIME 定数の非推奨対応
    - [ ] `MIMETextJavaScript`, `MIMETextJavaScriptCharsetUTF8` を追加定義
    - [ ] `MIMEApplicationJavaScript*` の実使用箇所がないか検索して、あれば置換
    - 参考: [What's New in v3 > MIME Constants](https://docs.gofiber.io/next/whats_new/)
  - [ ] 既存 API の互換確認（必要に応じて修正）
    - [ ] `Ctx` 周りで利用中のメソッドが v3 と互換であることを確認（`Type/Send/SendString/Set/Redirect/Cookie/Query/Params/Locals/Next/QueryParser/BodyParser/FormFile/FormValue/Get/JSON/Cookies/Response().StatusCode()/BaseURL/OriginalURL/Method/Protocol/IP/Context().UserAgent()/SendStream/BodyWriter`）
  - [ ] ミドルウェアの動作確認（`compress`, `cors`, `logger`）

- [ ] knowle_api（`github.com/generalworks/knowle_api`）側の対応
  - [ ] goutil の更新取り込み（`go.mod` の参照更新 or `replace`）
  - [ ] `go build` が成功すること
  - [ ] アプリ起動が成功すること
  - [ ] 主要 API の疎通（ログイン/投稿/ファイル配信など）

- [ ] 回帰確認（重要機能）
  - [ ] エラーハンドラのレスポンスヘッダ・ステータスコードが想定通り
  - [ ] ファイルアップロード（`FormFile`/`FormValue`）が成功
  - [ ] CORS/圧縮/ログの挙動が従来通り
  - [ ] セッション読み書き（ログイン/サインアウト）で `Save`/`Release` の問題がない

- 参考リンク
  - v3 変更点: https://docs.gofiber.io/next/whats_new/
  - v3 ドキュメント: https://docs.gofiber.io/next/


