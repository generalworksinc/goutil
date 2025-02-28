package gw_set

// Set は拡張された集合を表すジェネリック型です
type Set[K comparable, V any] map[K]V

// 後方互換性のための型エイリアス
// type set[T comparable] Set[T, V]

// NewSet は新しい空の集合を作成します（後方互換性のため）
func NewSet[T comparable, V any]() Set[T, V] {
	return make(Set[T, V])
}

// Add は集合に要素を追加します（後方互換性のため）
func (s Set[T, V]) Add(value T) {
	var zero V
	s[value] = zero
}

// Remove は集合から要素を削除します（後方互換性のため）
func (s Set[T, V]) Remove(value T) {
	delete(s, value)
}

// ToSlice は集合をスライスに変換します（後方互換性のため）
func (s Set[T, V]) ToSlice() []T {
	slice := make([]T, 0, len(s))
	for k := range s {
		slice = append(slice, k)
	}
	return slice
}

// 以下は新しい拡張機能

// NewSetKV は新しい空の拡張集合を作成します
func NewSetKV[K comparable, V any]() Set[K, V] {
	return make(Set[K, V])
}

// FromSlice はスライスから集合を作成します
// keyFn は各要素からキーを抽出する関数です
func FromSlice[K comparable, V any](items []V, keyFn func(V) K) Set[K, V] {
	set := NewSetKV[K, V]()
	for _, item := range items {
		key := keyFn(item)
		set[key] = item
	}
	return set
}

// Keys は集合のキーをスライスとして返します
func (s Set[K, V]) Keys() []K {
	keys := make([]K, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	return keys
}

// Contains はキーが集合に含まれているかどうかを返します
func (s Set[K, V]) Contains(key K) bool {
	_, exists := s[key]
	return exists
}

// DifferenceKeys は差集合のキーを返します
// s に含まれていて other に含まれていないキーの集合です
func DifferenceKeys[K comparable, V1 any, V2 any](s Set[K, V1], other Set[K, V2]) []K {
	result := make([]K, 0)
	for k := range s {
		if _, exists := other[k]; !exists {
			result = append(result, k)
		}
	}
	return result
}

// IntersectionKeys は共通集合のキーを返します
// s と other の両方に含まれているキーの集合です
func IntersectionKeys[K comparable, V1 any, V2 any](s Set[K, V1], other Set[K, V2]) []K {
	result := make([]K, 0)
	for k := range s {
		if _, exists := other[k]; exists {
			result = append(result, k)
		}
	}
	return result
}

// Difference は差集合（s - other）を返します
// s に含まれていて other に含まれていない要素の集合です
func (s Set[K, V]) Difference(other Set[K, V]) Set[K, V] {
	result := NewSetKV[K, V]()
	for k, v := range s {
		if !other.Contains(k) {
			result[k] = v
		}
	}
	return result
}

// Intersection は共通集合（s ∩ other）を返します
// s と other の両方に含まれている要素の集合です
func (s Set[K, V]) Intersection(other Set[K, V]) Set[K, V] {
	result := NewSetKV[K, V]()
	for k, v := range s {
		if other.Contains(k) {
			result[k] = v
		}
	}
	return result
}

// Union は和集合（s ∪ other）を返します
// s または other のいずれかに含まれている要素の集合です
func (s Set[K, V]) Union(other Set[K, V]) Set[K, V] {
	result := NewSetKV[K, V]()

	// s の要素をすべて追加
	for k, v := range s {
		result[k] = v
	}

	// other の要素を追加（重複する場合は other の値で上書き）
	for k, v := range other {
		result[k] = v
	}

	return result
}

// Size は集合の要素数を返します
func (s Set[K, V]) Size() int {
	return len(s)
}

// IsEmpty は集合が空かどうかを返します
func (s Set[K, V]) IsEmpty() bool {
	return len(s) == 0
}

// Clear は集合のすべての要素を削除します
func (s Set[K, V]) Clear() {
	for k := range s {
		delete(s, k)
	}
}

// ForEach は集合の各要素に対して関数を適用します
func (s Set[K, V]) ForEach(fn func(K, V)) {
	for k, v := range s {
		fn(k, v)
	}
}

// Filter は条件を満たす要素だけを含む新しい集合を返します
func (s Set[K, V]) Filter(predicate func(K, V) bool) Set[K, V] {
	result := NewSetKV[K, V]()
	for k, v := range s {
		if predicate(k, v) {
			result[k] = v
		}
	}
	return result
}

// 以下は後方互換性のための実装（SimpleSetは削除）

// func sample() {
// 	set := NewSet[string]()

// 	set.Add("item1")
// 	set.Add("item2")
// 	fmt.Println(set.Contains("item1")) // true
// 	fmt.Println(set.Contains("item3")) // false

// 	set.Remove("item1")
// 	fmt.Println(set.Contains("item1")) // false
// }
