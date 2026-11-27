// Package jsonmode は、jsontest.Build が app の JSON のモードを上書きする値を持つ。gw_web と jsontest だけが使う。
package jsonmode

import "sync"

var (
	mu       sync.Mutex
	override *int
)

// Set は上書きを mode にし、前の状態へ戻す関数を返す。
func Set(mode int) (restore func()) {
	mu.Lock()
	prev := override
	override = &mode
	mu.Unlock()
	return func() {
		mu.Lock()
		override = prev
		mu.Unlock()
	}
}

// Override は今の上書きを返す。上書きが無ければ ok は false。
func Override() (mode int, ok bool) {
	mu.Lock()
	defer mu.Unlock()
	if override == nil {
		return 0, false
	}
	return *override, true
}
