package gw_xorm

import (
	gw_strings "github.com/generalworksinc/goutil/strings"
	"reflect"
	"strings"
)

func getStructFields(v reflect.Value, isSnakeCase bool) []string {
	fields := []string{}
	for i := 0; i < v.NumField(); i++ {
		structValue := reflect.Indirect(v)
		t := structValue.Type()

		// フィールド名の取得
		//フィールドがstructだった場合、再帰的に取得
		if t.Field(i).Type.Kind() == reflect.Struct && strings.Contains(t.Field(i).Tag.Get("xorm"), "extends") {
			f := v.Field(i)
			fields = append(fields, getStructFields(f, isSnakeCase)...)
		} else {
			t := reflect.Indirect(v).Type()
			if isSnakeCase {
				fields = append(fields, gw_strings.ToSnakeCase(t.Field(i).Name))
			} else {
				fields = append(fields, t.Field(i).Name)
			}
		}
	}
	return fields
}
func GetStructFields(st interface{}, isSnakeCase bool) []string {
	fields := []string{}
	val := reflect.ValueOf(st)
	v := reflect.Indirect(val)
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		// フィールド名の取得
		//フィールドがstructだった場合、再帰的に取得
		if t.Field(i).Type.Kind() == reflect.Struct && strings.Contains(t.Field(i).Tag.Get("xorm"), "extends") {
			f := v.Field(i)
			fields = append(fields, getStructFields(f, isSnakeCase)...)
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
