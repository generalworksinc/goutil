package gw_map

import "encoding/json"

type DoubleKey[T1 comparable, T2 comparable] struct {
	Key1 T1
	Key2 T2
}

type DoubleKeyMap[T1 comparable, T2 comparable, T3 any] map[DoubleKey[T1, T2]]T3

func (m DoubleKeyMap[T1, T2, T3]) Contains(key1 T1, key2 T2) bool {
	_, exists := m[DoubleKey[T1, T2]{Key1: key1, Key2: key2}]
	return exists
}
func (m DoubleKeyMap[T1, T2, T3]) Get(key1 T1, key2 T2) T3 {
	return m[DoubleKey[T1, T2]{Key1: key1, Key2: key2}]
}
func (m DoubleKeyMap[T1, T2, T3]) GetMapByFirstKey(key1 T1) map[T2]T3 {
	result := make(map[T2]T3)
	for k, v := range m {
		if k.Key1 == key1 {
			result[k.Key2] = v
		}
	}
	return result
}
func (m DoubleKeyMap[T1, T2, T3]) Set(key1 T1, key2 T2, value T3) {
	m[DoubleKey[T1, T2]{Key1: key1, Key2: key2}] = value
}

func (m DoubleKeyMap[T1, T2, T3]) Delete(key1 T1, key2 T2) {
	delete(m, DoubleKey[T1, T2]{Key1: key1, Key2: key2})
}

func (m DoubleKeyMap[T1, T2, T3]) Keys() []DoubleKey[T1, T2] {
	return GetKeysFromMap(m)
}

type jSONDoubleKeyMap[T1 comparable, T2 comparable, T3 any] map[T1]map[T2]T3

func (dkm *DoubleKeyMap[T1, T2, T3]) UnmarshalJSON(b []byte) error {
	var jsonMap jSONDoubleKeyMap[T1, T2, T3]
	if err := json.Unmarshal(b, &jsonMap); err != nil {
		return err
	}

	*dkm = make(DoubleKeyMap[T1, T2, T3])
	for k1, v1 := range jsonMap {
		for k2, v2 := range v1 {
			(*dkm)[MakeKey2(k1, k2)] = v2
		}
	}
	return nil
}

func (dkm DoubleKeyMap[T1, T2, T3]) MarshalJSON() ([]byte, error) {
	jsonMap := make(jSONDoubleKeyMap[T1, T2, T3])
	for k, v := range dkm {
		if _, ok := jsonMap[k.Key1]; !ok {
			jsonMap[k.Key1] = make(map[T2]T3)
		}
		jsonMap[k.Key1][k.Key2] = v
	}

	return json.Marshal(jsonMap)
}

func MakeKey2[T1 comparable, T2 comparable](key1 T1, key2 T2) DoubleKey[T1, T2] {
	return DoubleKey[T1, T2]{Key1: key1, Key2: key2}
}
