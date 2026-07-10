package gw_authz

// defaultModelText は RBAC with domains（domain=テナントID）の標準モデル。
//
// - sub: ロール名（"admin" 等）。ユーザー→ロールの割当ては casbin に持たせない
//   （アプリ側の User.Role 等が真実の源。二重管理を避ける）
// - dom: テナントID。ポリシー側の "*" は全テナント共通ルールを表し、
//   具体的なテナントIDの行を足すことでテナント個別の上書きができる
// - g: ロール階層のみに使う（例: g, admin, manager = admin は manager の許可を継承）
// - マッチしないリクエストはデフォルト拒否
const defaultModelText = `
[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act

[role_definition]
g = _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = (r.dom == p.dom || p.dom == "*") && g(r.sub, p.sub) && r.obj == p.obj && r.act == p.act
`
