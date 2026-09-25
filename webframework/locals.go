package gw_web

// LocalKey identifies one request-local value of type T. Create a key once
// and reuse it in the middleware and handlers that share the value.
type LocalKey[T any] struct {
	token *int
}

// NewLocalKey creates a distinct key, even when another key has the same type.
func NewLocalKey[T any]() LocalKey[T] {
	return LocalKey[T]{token: new(int)}
}

// SetLocal stores a typed request-local value.
func (ctx WebCtx) SetLocal[T any](key LocalKey[T], value T) {
	if key.token == nil {
		panic("gw_web: zero LocalKey")
	}
	ctx.Locals(key.token, value)
}

// Local returns a typed request-local value. ok is false when the key is unset.
func (ctx WebCtx) Local[T any](key LocalKey[T]) (value T, ok bool) {
	if key.token == nil {
		panic("gw_web: zero LocalKey")
	}
	value, ok = ctx.Locals(key.token).(T)
	return value, ok
}
