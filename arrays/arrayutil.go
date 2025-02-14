package gw_arrays

func ContainsUint(array []uint, value uint) bool {
	for _, v := range array {
		if value == v {
			return true
		}
	}
	return false
}
func GetKeys(m map[string]interface{}) []string {
	ks := []string{}
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

func RemoveDuplicateStrList(array []string) []string {
	// 順番は保持されません
	retMap := map[string]interface{}{}
	for _, v := range array {
		retMap[v] = struct{}{}
	}
	return GetKeys(retMap)
}
func ContainsString(array []string, value string) bool {
	for _, v := range array {
		if value == v {
			return true
		}
	}
	return false
}

func IndexString(array []string, value string) int {
	for ind, v := range array {
		if value == v {
			return ind
		}
	}
	return -1
}
func UnionString(xs, ys []string) []string {
	zs := make([]string, len(xs))
	copy(zs, xs)
	for _, y := range ys {
		if !ContainsString(zs, y) {
			zs = append(zs, y)
		}
	}
	return zs
}
func IntersectString(xs, ys []string) []string {
	zs := make([]string, 0)
	for _, y := range ys {
		if ContainsString(xs, y) {
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
