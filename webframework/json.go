package gw_web

import "encoding/json/v2"

// newJSONCodec は、Fiber の JSON の読み書きを json v2 で行う関数の組を作る。
// json.DefaultOptionsV2 は Deterministic(false) を含むので、Deterministic(true) は最後に置く。
func newJSONCodec(opts []json.Options) (func(any) ([]byte, error), func([]byte, any) error) {
	joined := json.JoinOptions(append(append([]json.Options{}, opts...), json.Deterministic(true))...)
	marshal := func(v any) ([]byte, error) {
		return json.Marshal(v, joined)
	}
	unmarshal := func(data []byte, v any) error {
		return json.Unmarshal(data, v, joined)
	}
	return marshal, unmarshal
}
