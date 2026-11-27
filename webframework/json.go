package gw_web

import (
	"bytes"
	jsonv1 "encoding/json"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"

	"github.com/generalworksinc/goutil/webframework/internal/jsonmode"
	"github.com/gofiber/fiber/v3"
)

// JSONMode は、app が HTTP の JSON を v1(encoding/json)と v2(encoding/json/v2)のどちらで読み書きするか。
// ゼロ値は JSONV1。
type JSONMode int

const (
	// JSONV1 は Fiber の既定の encoding/json で読み書きする。
	JSONV1 JSONMode = iota
	// JSONV2 は encoding/json/v2 を json.DefaultOptionsV2() で使う。map のキーの順は固定しない。
	JSONV2
)

// ErrTrailingJSONValue は、DecodeJSONStrict(v1)で、本文の値の後ろに別の JSON の値が続いた時の誤り。
// 後ろが JSON として壊れている時は、encoding/json の SyntaxError を返す。v2 では、どちらも jsontext.SyntacticError になる。
var ErrTrailingJSONValue = errors.New("unexpected trailing JSON value")

const jsonStateKey = "github.com/generalworksinc/goutil/webframework.json"

// resolveJSONMode は、app が使うモードを決める。
// 試験の中で jsontest.Build が上書きしていれば、AppSettings.JSON に関わらずそのモードにする。
func resolveJSONMode(settings *AppSettings) JSONMode {
	mode := JSONV1
	if settings != nil {
		mode = settings.JSON
	}
	// 上書きの前に確かめる(試験で上書きしていても、本番の設定の誤りを隠さない)
	if mode != JSONV1 && mode != JSONV2 {
		panic(fmt.Sprintf("gw_web: 未知の JSONMode %d", mode))
	}
	if m, ok := jsonmode.Override(); ok {
		mode = JSONMode(m)
	}
	return mode
}

// newJSONV2Codec は、Fiber の JSON の読み書きを json v2 の既定で行う関数の組を作る。
func newJSONV2Codec() (func(any) ([]byte, error), func([]byte, any) error) {
	marshal := func(v any) ([]byte, error) {
		return json.Marshal(v, json.DefaultOptionsV2())
	}
	unmarshal := func(data []byte, v any) error {
		return json.Unmarshal(data, v, json.DefaultOptionsV2())
	}
	return marshal, unmarshal
}

// DecodeJSON は data を、要求を受けた app の JSON の設定で out に読む。BindJSON と同じ decoder を使う
// (BindJSON の後で行う誤りの整形と struct の validator は行わない)。
// 要求の本文に入れ子になった JSON の文字列など、HTTP の入力を BindJSON を通さずに読む時に使う。
// DB の列や外の API の応答など、HTTP の外の JSON には使わない(app のモードに従ってしまうため)。
func (ctx WebCtx) DecodeJSON(data []byte, out any) error {
	return ctx.Ctx.(fiber.Ctx).App().Config().JSONDecoder(data, out)
}

// DecodeJSONStrict は DecodeJSON と同じ設定で、知らないキーと、後ろに続く値を誤りにして読む。
// v1 では encoding/json の Decoder に DisallowUnknownFields を付けて読み、後ろに値が残れば誤りにする。
// v2 では json.RejectUnknownMembers(true) を足して読む。
// goutil の NewApp* で作った app の要求でだけ使える(モードを app に置いているため。そうでない app では panic する)。
func (ctx WebCtx) DecodeJSONStrict(data []byte, out any) error {
	if ctx.jsonMode() == JSONV1 {
		decoder := jsonv1.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(out); err != nil {
			return err
		}
		if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
			if err == nil {
				return ErrTrailingJSONValue
			}
			return err
		}
		return nil
	}
	return json.Unmarshal(data, out, json.DefaultOptionsV2(), json.RejectUnknownMembers(true))
}

func (ctx WebCtx) jsonMode() JSONMode {
	value, ok := ctx.Ctx.(fiber.Ctx).App().State().Get(jsonStateKey)
	if !ok {
		panic("gw_web: この app は goutil の NewApp* で作られていない(JSON のモードが無い)")
	}
	return value.(JSONMode)
}
