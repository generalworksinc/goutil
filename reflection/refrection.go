package gw_reflection

import (
	gw_strings "github.com/generalworksinc/goutil/strings"
	"reflect"
)

func GetStructFields(st interface{}, isSnakeCase bool) []string {
	fields := []string{}
	v := reflect.Indirect(reflect.ValueOf(st))
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		// フィールド名の取得
		//フィールドがstructだった場合、再帰的に取得
		if t.Field(i).Type.Kind() == reflect.Struct {
			f := v.Field(i)
			fields = append(fields, GetStructFields(f.Interface(), isSnakeCase)...)
		} else {
			if isSnakeCase {
				fields = append(fields, gw_strings.ToSnakeCase(t.Field(i).Name))
			} else {
				fields = append(fields, t.Field(i).Name)
			}
		}
	}
	return fields
}
