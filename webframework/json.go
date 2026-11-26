package gw_web

import "encoding/json/v2"

// newJSONCodec は、Fiber の JSON の読み書きを、渡されたオプションのまま json v2 で行う関数の組を作る。
func newJSONCodec(opts []json.Options) (func(any) ([]byte, error), func([]byte, any) error) {
	joined := json.JoinOptions(opts...)
	marshal := func(v any) ([]byte, error) {
		return json.Marshal(v, joined)
	}
	unmarshal := func(data []byte, v any) error {
		return json.Unmarshal(data, v, joined)
	}
	return marshal, unmarshal
}
