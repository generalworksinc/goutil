package gw_arrays

func ContainsUint(array []uint, value uint) bool {
	for _, v := range array {
		if value == v {
			return true
		}
	}
	return false
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
