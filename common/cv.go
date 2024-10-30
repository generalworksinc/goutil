package gw_common

func Ip(i int) *int {
	return &i
}

func Pointer[T any](value T) *T {
	return &value
}
