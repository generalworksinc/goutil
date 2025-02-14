package gw_numeric

import (
	"encoding/json"
	"strconv"
	"strings"
)

func CNullUInt(num json.Number) uint {

	if i, err := num.Int64(); err == nil {
		return uint(i)
	} else {
		return 0
	}
}

func CNulStrlUInt(num string) uint {

	if i, err := strconv.Atoi(num); err == nil {
		return uint(i)
	} else {
		return 0
	}
}
func CNullStrIntByJson(json map[string]interface{}, key string) *int {
	data := json[key]
	if data == nil {
		return nil
	}
	return CNullStrInt(strings.TrimSpace(data.(string)))
}
func CNullFloatByJson(json map[string]interface{}, key string) *float64 {
	defer func() {
		if e := recover(); e != nil {
			panic(e)
		}
	}()
	data := json[key]
	if data == nil {
		return nil
	}

	num := data.(float64)
	return &num
}
func CNullFloatToIntByJson(json map[string]interface{}, key string) *int {
	defer func() {
		if e := recover(); e != nil {
			panic(e)
		}
	}()
	data := json[key]
	if data == nil {
		return nil
	}
	num := int(data.(float64))
	return &num
}

// breaking change
//
//	func CNullStrInt(num string) int {
//			if i, err := strconv.Atoi(num); err == nil {
//				return i
//			} else {
//				return 0
//			}
//		}
func CNullStrInt(num string) *int {

	num = strings.Replace(num, ",", "", -1)
	if i, err := strconv.Atoi(num); err == nil {
		return &i
	} else {
		return nil
	}
}
