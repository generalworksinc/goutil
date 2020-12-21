package gw_common

func CpInt(cond bool, a int, b int) int {
	if cond {
		return a
	} else {
		return b
	}
}
