# common codes for golang (Generalworks inc)

Go 1.27 以降が必要です。

## 型付きリクエストローカル値

`webframework.NewLocalKey[T]()` でキーを一度作り、ミドルウェアとハンドラで共有します。`WebCtx.SetLocal` と `WebCtx.Local` は値の型をキーから推論します。

```go
var userIDKey = gw_web.NewLocalKey[string]()

ctx.SetLocal(userIDKey, "user-123")
userID, ok := ctx.Local(userIDKey)
```

### `WebCtx.Locals` からの移行

1. 既存のキーごとに `NewLocalKey[T]()` を使い、パッケージレベルでキーを一度だけ作成します。書き込み側と読み取り側が別パッケージなら、共有パッケージから同じキーを参照します。
2. `ctx.Locals("Auth", auth)` を `ctx.SetLocal(authKey, auth)` に、`ctx.Locals("Auth")` とその型アサーションを `auth, ok := ctx.Local(authKey)` に置き換えます。値が未設定の場合は `ok` が `false` です。
3. 同じ文字列キーを使う書き込み側と読み取り側をまとめて移行します。型付きキーと文字列キーは別物なので、片側だけ変更すると値を受け渡せません。

`WebCtx.Locals` は非推奨ですが、既存コードとの互換性のため引き続き動作します。`WebSocketConn.Locals` は今回の移行対象ではありません。

## Mongo クエリの派生

共有する `Database[T]` から条件を付ける場合は `WithID`、`WithObjectID`、`WithSession` を使用します。これらは元の値を書き換えません。従来の `Id`、`Oid`、`Session` は既存コードとの互換性のため、引き続き受信側を書き換えます。

```go
user, err := db.WithID(id).FindOne(nil)
```

## Mongo integration tests

- `go test ./mongo -v` で Mongo 統合テストが実行されます。
- デフォルトでは `testcontainers` で `mongo:7` を自動起動します（Docker が必要）。
- 既存 Mongo を使う場合は `GOUTIL_IT_MONGO_URI` と `GOUTIL_IT_MONGO_DB` を設定してください。
- `testcontainers` を無効化する場合は `GOUTIL_IT_USE_TESTCONTAINERS=0` を設定してください。
