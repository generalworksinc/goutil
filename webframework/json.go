package gw_web

import (
	"bytes"
	jsonv1 "encoding/json"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"slices"

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

const jsonStateKey = "github.com/generalworksinc/goutil/webframework.json"

// jsonState は app を作る時に決まった JSON の設定。nil は v1。
type jsonState struct{ opts []json.Options }

// resolveJSONOptions は、app が使う JSON の設定を決める。nil は v1。
// 試験の中で jsontest.Build が上書きしていれば、AppSettings の JSON・JSONOptions に関わらずそのモードにする。
func resolveJSONOptions(settings *AppSettings) []json.Options {
	if mode, ok := jsonmode.Override(); ok {
		return optionsForMode(JSONMode(mode), nil)
	}
	if settings == nil {
		return nil
	}
	return optionsForMode(settings.JSON, settings.JSONOptions)
}

func optionsForMode(mode JSONMode, custom []json.Options) []json.Options {
	switch mode {
	case JSONV1:
		if len(custom) == 0 {
			return nil
		}
		return slices.Clone(custom)
	case JSONV2:
		if len(custom) > 0 {
			panic("gw_web: AppSettings の JSON: JSONV2 と JSONOptions は同時に指定できない(細かく決めるなら JSONOptions だけを使う)")
		}
		return []json.Options{json.DefaultOptionsV2()}
	default:
		panic(fmt.Sprintf("gw_web: 未知の JSONMode %d", mode))
	}
}

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

// DecodeJSON は data を、要求を受けた app の JSON の設定で out に読む。BindJSON と同じ decoder を使う
// (BindJSON の後で行う誤りの整形と struct の validator は行わない)。
// 要求の本文に入れ子になった JSON の文字列など、HTTP の入力を BindJSON を通さずに読む時に使う。
// DB の列や外の API の応答など、HTTP の外の JSON には使わない(app のモードに従ってしまうため)。
func (ctx WebCtx) DecodeJSON(data []byte, out any) error {
	return ctx.Ctx.(fiber.Ctx).App().Config().JSONDecoder(data, out)
}

// DecodeJSONStrict は DecodeJSON と同じ設定で、知らないキーと、後ろに続く値を誤りにして読む。
// v1 では encoding/json の Decoder に DisallowUnknownFields を付けて読み、後ろに値が残れば誤りにする。
// v2(と JSONOptions)では、その設定に json.RejectUnknownMembers(true) を足して読む。
func (ctx WebCtx) DecodeJSONStrict(data []byte, out any) error {
	opts := ctx.jsonOptions()
	if len(opts) == 0 {
		decoder := jsonv1.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(out); err != nil {
			return err
		}
		if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
			if err == nil {
				return errors.New("unexpected trailing JSON value")
			}
			return err
		}
		return nil
	}
	return json.Unmarshal(data, out, append(slices.Clone(opts), json.RejectUnknownMembers(true))...)
}

func (ctx WebCtx) jsonOptions() []json.Options {
	value, ok := ctx.Ctx.(fiber.Ctx).App().State().Get(jsonStateKey)
	if !ok {
		panic("gw_web: この app は goutil の NewApp* で作られていない(JSON の設定が無い)")
	}
	return value.(jsonState).opts
}
