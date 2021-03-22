package reflection

import (
	"reflect"
	"regexp"
	"strings"
)

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func ToSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func GetStructFields(st interface{}, isSnakeCase bool) []string {
	fields := []string{}
	v := reflect.Indirect(reflect.ValueOf(st))
	t := v.Type()

	for i := 0; i < t.NumField(); i++ {
		// フィールド名
		if isSnakeCase {
			fields = append(fields, t.Field(i).Name)
		} else {
			fields = append(fields, ToSnakeCase(t.Field(i).Name))
		}
	}
	return fields
}
