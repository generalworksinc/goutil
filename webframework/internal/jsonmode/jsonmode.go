// Package jsonmode は、jsontest.Build が app の JSON のモードを上書きする値を持つ。gw_web と jsontest だけが使う。
package jsonmode

import "sync/atomic"

var override atomic.Pointer[int]

// Set は上書きを mode にし、前の状態へ戻す関数を返す。
func Set(mode int) (restore func()) {
	prev := override.Swap(&mode)
	return func() { override.Store(prev) }
}

// Override は今の上書きを返す。上書きが無ければ ok は false。
func Override() (mode int, ok bool) {
	p := override.Load()
	if p == nil {
		return 0, false
	}
	return *p, true
}
