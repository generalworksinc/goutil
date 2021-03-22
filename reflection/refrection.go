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
		// フィールド名
		if isSnakeCase {
			fields = append(fields, t.Field(i).Name)
		} else {
			fields = append(fields, gw_strings.ToSnakeCase(t.Field(i).Name))
		}
	}
	return fields
}
