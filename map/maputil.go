package gw_map

func GetKeysFromMap[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m)) // スライスを初期化
	for k := range m {
		keys = append(keys, k) // キーをスライスに追加
	}
	return keys
}

func GetValuesFromMap[K comparable, V any](m map[K]V) []V {
	values := make([]V, 0, len(m)) // スライスを初期化
	for _, v := range m {
		values = append(values, v) // 値をスライスに追加
	}
	return values
}
