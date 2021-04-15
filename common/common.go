package gw_common

func CpInt(cond bool, a int, b int) int {
	if cond {
		return a
	} else {
		return b
	}
}

type Set struct {
	dataMap map[interface{}]string
}

func (set *Set) Put(key interface{}) {
	set.dataMap[key] = ""
}
func (set *Set) Values(key interface{}) []interface{} {
	keys := []interface{}{}
	for k, _ := range set.dataMap {
		keys = append(keys, k)
	}
	return keys
}

type StringSet struct {
	dataMap map[string]string
}

func (set *StringSet) Put(key string) {
	set.dataMap[key] = ""
}
func (set *StringSet) Values(key string) []string {
	keys := []string{}
	for k, _ := range set.dataMap {
		keys = append(keys, k)
	}
	return keys
}
