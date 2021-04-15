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
	if set.dataMap == nil {
		set.dataMap = map[interface{}]string{}
	}
	set.dataMap[key] = ""
}
func (set *Set) Values() []interface{} {
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
	if set.dataMap == nil {
		set.dataMap = map[string]string{}
	}
	set.dataMap[key] = ""
}
func (set *StringSet) Values() []string {
	keys := []string{}
	for k, _ := range set.dataMap {
		keys = append(keys, k)
	}
	return keys
}
