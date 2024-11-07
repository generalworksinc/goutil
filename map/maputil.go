package gw_map

func GetKeysFromMap[K comparable, V any](m map[K]V) []K {
	keys := make([]K, 0, len(m)) // スライスを初期化
	for k := range m {
		keys = append(keys, k) // キーをスライスに追加
	}
	return keys
}
