# common codes for golang (Generalworks inc)

Go 1.27.3 以降が必要です。

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

## JSON を json v2 で読み書きする

`AppSettings.JSONOptions` にオプションを渡すと、`BindJSON` とレスポンスの `JSON` を `encoding/json/v2` で、そのオプション付きで行います。渡さなければ Fiber の既定(`encoding/json`、v1 の挙動)のままです。

```go
import (
	jsonv1 "encoding/json"
	"encoding/json/v2"
)

// v2 の挙動にそろえ、map をキーの順に並べる
gw_web.NewAppWithSettings(errorHandler, &gw_web.AppSettings{JSONOptions: []json.Options{json.DefaultOptionsV2(), json.Deterministic(true)}})

// v1 の挙動から 1 つずつ外す(この例は nil の slice を [] で書き出す)
gw_web.NewAppWithSettings(errorHandler, &gw_web.AppSettings{JSONOptions: []json.Options{jsonv1.DefaultOptionsV1(), json.FormatNilSliceAsNull(false)}})
```

オプションは加工せずにそのまま、読み込みと書き出しの両方に渡します。片方にだけ効くオプションは、もう片方では無視されます。`StringifyNumbers` のように両方に効くオプションは両方向を変えます(数値を文字列で受けるために足すと、応答の数値も文字列になります)。v2 の既定では map の順番が書き出すたびに変わります。Go の map をキーの順に並べたいなら `json.Deterministic(true)` を足します。`DefaultOptionsV2()` は `Deterministic(false)` を含み、後ろに置いたオプションが勝つので、`Deterministic(true)` は `DefaultOptionsV2()` より後ろに置きます。`DefaultOptionsV1()` は `Deterministic(true)` を含みます。

効くのは、Fiber の `JSONEncoder`・`JSONDecoder` を使う経路です(`BindJSON`・JSON の content type の `BodyParser`・`JSON`・JSONP・SSE のイベント・SharedState の `SetJSON`/`GetJSON`)。SharedStorage に保存済みのデータがあるなら、読み戻せるかを確かめてから切り替えます。利用側が `encoding/json` を直接呼ぶところ、生の `Send`、共通のエラーの応答(平文)は変わりません。

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
