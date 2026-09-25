package gw_arrays

import "slices"

func ContainsUint(array []uint, value uint) bool {
	return slices.Contains(array, value)
}
func GetKeys(m map[string]interface{}) []string {
	ks := []string{}
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

// 順番を保持したまま重複を削除する
func RemoveDuplicateStrList(array []string) []string {
	return RemoveDuplicate(array)
}

// 順番を保持したまま重複を削除する（Generics版）
func RemoveDuplicate[T comparable](array []T) []T {
	seen := make(map[T]struct{}, len(array))
	result := make([]T, 0, len(array))
	for _, v := range array {
		if _, exists := seen[v]; !exists {
			seen[v] = struct{}{}
			result = append(result, v)
		}
	}
	return result
}
func ContainsString(array []string, value string) bool {
	return slices.Contains(array, value)
}

func IndexString(array []string, value string) int {
	return slices.Index(array, value)
}
func UnionString(xs, ys []string) []string {
	zs := make([]string, len(xs))
	copy(zs, xs)
	seen := make(map[string]struct{}, len(xs)+len(ys))
	for _, x := range xs {
		seen[x] = struct{}{}
	}
	for _, y := range ys {
		if _, exists := seen[y]; !exists {
			seen[y] = struct{}{}
			zs = append(zs, y)
		}
	}
	return zs
}
func IntersectString(xs, ys []string) []string {
	zs := make([]string, 0)
	seen := make(map[string]struct{}, len(xs))
	for _, x := range xs {
		seen[x] = struct{}{}
	}
	for _, y := range ys {
		if _, exists := seen[y]; exists {
			zs = append(zs, y)
		}
	}
	return zs
}
func RemoveString(strarray []string, str string) []string {
	result := []string{}
	for _, s := range strarray {
		if s != str {
			result = append(result, s)
		}
	}
	return result
}
func RemoveStrInd(s []string, ind int) []string {
	s = append(s[:ind], s[ind+1:]...)
	return s
}

// Filter関数は、スライスとフィルタリング条件を受け取り、条件を満たす要素からなる新しいスライスを返します。
func Filter[T any](slice []T, predicate func(T) bool) []T {
	var result []T
	for _, elem := range slice {
		if predicate(elem) {
			result = append(result, elem)
		}
	}
	return result
}
