# common codes for golang (Generalworks inc)

## Mongo integration tests

- `go test ./mongo -v` で Mongo 統合テストが実行されます。
- デフォルトでは `testcontainers` で `mongo:7` を自動起動します（Docker が必要）。
- 既存 Mongo を使う場合は `GOUTIL_IT_MONGO_URI` と `GOUTIL_IT_MONGO_DB` を設定してください。
- `testcontainers` を無効化する場合は `GOUTIL_IT_USE_TESTCONTAINERS=0` を設定してください。
