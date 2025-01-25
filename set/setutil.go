package gw_set

type set[T comparable] map[T]struct{}

func NewSet[T comparable]() set[T] {
	return make(set[T])
}

func (s set[T]) Add(value T) {
	s[value] = struct{}{}
}

func (s set[T]) Contains(value T) bool {
	_, exists := s[value]
	return exists
}

func (s set[T]) Remove(value T) {
	delete(s, value)
}
func (s set[T]) ToSlice() []T {
	slice := make([]T, len(s))
	for k := range s {
		slice = append(slice, k)
	}
	return slice
}

// func sample() {
// 	set := NewSet[string]()

// 	set.Add("item1")
// 	set.Add("item2")
// 	fmt.Println(set.Contains("item1")) // true
// 	fmt.Println(set.Contains("item3")) // false

// 	set.Remove("item1")
// 	fmt.Println(set.Contains("item1")) // false
// }
