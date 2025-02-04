package gw_common

func Ip(i int) *int {
	return &i
}

func Pointer[T any](value T) *T {
	return &value
}

func Clone[T any](value *T) *T {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func CloneSliceP[T any](value []*T) []*T {
	if value == nil {
		return nil
	}
	clonedSlice := make([]*T, len(value))
	for i, v := range value {
		clonedSlice[i] = Clone(v)
	}
	return clonedSlice
}
func CloneSlice[T any](value []T) []T {
	if value == nil {
		return nil
	}
	clonedSlice := make([]T, len(value))
	copy(clonedSlice, value)
	return clonedSlice

}

func CloneMapP[K comparable, V any](value map[K]*V) map[K]*V {
	if value == nil {
		return nil
	}
	clonedMap := make(map[K]*V, len(value))
	for k, v := range value {
		clonedMap[k] = Clone(v)

	}
	return clonedMap
}

func CloneMap[K comparable, V any](value map[K]V) map[K]V {
	if value == nil {
		return nil
	}
	clonedMap := make(map[K]V, len(value))
	for k, v := range value {
		clonedMap[k] = v
	}
	return clonedMap
}
