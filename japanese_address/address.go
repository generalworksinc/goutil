package gw_japanese_address

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/ktnyt/go-moji"
	"github.com/kurehajime/cjk2num"
)

var kansujiChars = []string{"京", "兆", "億", "万", "萬", "만", "千", "仟", "천", "百", "佰", "백", "十", "拾", "십", "廿", "念", "입", "卅", "삽", "卌", "십", "〇", "一", "二", "三", "四", "五", "六", "七", "八", "九", "零", "壱", "弐", "参", "壹", "貳", "叁", "肆", "伍", "陸", "柒", "捌", "玖", "영", "령", "일", "이", "삼", "사", "오", "육", "륙", "칠", "팔", "구"}
var kansujiRe = regexp.MustCompile(`([` + strings.Join(kansujiChars, "") + `]+)`)
var NormalizeAddressBanchiRe = regexp.MustCompile(`(\d+)(番地?|丁目)(\d)`)

// 数値と数値の間のスペース
var SpaceBetweenNumberRe = regexp.MustCompile(`(\d+)\s(\d+)`)

func NormalizeAddress(address string) string {
	// 不要な文字を削除/置換
	address = strings.ReplaceAll(address, "ー", "-") // 全角ハイフンを半角に
	address = strings.ReplaceAll(address, "－", "-") // 全角ハイフンを半角に

	// Convert Hankaku Katakana to Zenkaku Katakana
	address = moji.Convert(address, moji.HK, moji.ZK)
	// Convert HiraGana to KataKana
	address = moji.Convert(address, moji.HG, moji.KK)
	// Convert Zenkaku Eisuu to Hankaku Eisuu
	address = moji.Convert(address, moji.ZE, moji.HE)
	// Convert Zenkaku Space to Hankaku Space
	address = moji.Convert(address, moji.ZS, moji.HS)

	// 漢数字部分を、数字に変換
	//漢数字のみを抽出し、sliceに分割する
	// 住所文字列を漢数字とそれ以外で分割し、漢数字部分のみを変換
	parts := kansujiRe.Split(address, -1)
	matches := kansujiRe.FindAllString(address, -1)

	var result string
	for i, part := range parts {
		result += part
		if i < len(matches) {
			if num, err := cjk2num.Convert(matches[i]); err == nil {
				result += strconv.FormatInt(num, 10)
			} else {
				result += matches[i]
			}
		}
	}
	address = result

	// 丁目、番地、号、階を統一

	// 番や番地の後に数値が続く場合のみハイフンに変換
	address = NormalizeAddressBanchiRe.ReplaceAllString(address, "$1-$3")

	// 残りの番、番地、丁目を削除（数値が続かない場合は変換しない）
	address = strings.ReplaceAll(address, "号", "")
	address = strings.ReplaceAll(address, "階", "F")

	// address = strings.ReplaceAll(address, "番", "-")
	// address = strings.ReplaceAll(address, "番地", "-")
	// address = strings.ReplaceAll(address, "丁目", "-")

	//数値と数値の間のスペースは、文字,として残す
	address = SpaceBetweenNumberRe.ReplaceAllString(address, "$1,$2")
	address = strings.ReplaceAll(address, " ", "") // 半角スペース削除
	return address
}
