// Package jsontest は、JSON の v1 と v2 の差を比べる試験で、本番と同じ組み立ての app を指定のモードで作る。
// 本番を v2 へ替えたら、差の試験と一緒に使う所を消す前提の道具。
package jsontest

import (
	"fmt"
	"testing"

	gw_web "github.com/generalworksinc/goutil/webframework"
	"github.com/generalworksinc/goutil/webframework/internal/jsonmode"
)

// Build は build を呼び、その間に goutil の NewApp* で作った app を、AppSettings の JSON・JSONOptions に関わらず mode にする。
// 設定は app を作る時に決まるので、Build の後もその app は mode のまま。Build の外で作った app には効かない。
// 上書きはプロセス全体で 1 つなので、build の間に別の goroutine が作った app にも効く。並行に走る試験では使わない。
func Build[T any](mode gw_web.JSONMode, build func() T) T {
	if !testing.Testing() {
		panic("jsontest: 試験の中だけで使う")
	}
	if mode != gw_web.JSONV1 && mode != gw_web.JSONV2 {
		panic(fmt.Sprintf("jsontest: 未知の JSONMode %d", mode))
	}
	restore := jsonmode.Set(int(mode))
	defer restore()
	return build()
}
