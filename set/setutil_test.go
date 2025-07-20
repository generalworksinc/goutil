package gw_set

import (
	"sort"
	"testing"
)

// カスタム構造体のテスト用
type testStruct struct {
	ID   int
	Name string
}

// 元のテスト（後方互換性のため）
func TestNewSet(t *testing.T) {
	s := NewSet[string]()
	if len(s) != 0 {
		t.Errorf("Expected empty set, got %v", s)
	}
}

func TestAdd(t *testing.T) {
	s := NewSet[string]()
	s.Add("test")
	if len(s) != 1 {
		t.Errorf("Expected set with 1 element, got %v", s)
	}
	if _, exists := s["test"]; !exists {
		t.Errorf("Expected set to contain 'test'")
	}
}

func TestContains(t *testing.T) {
	s := NewSet[string]()
	s.Add("test")
	if !s.Contains("test") {
		t.Error("Expected set to contain 'test'")
	}
	if s.Contains("nonexistent") {
		t.Error("Expected set to not contain 'nonexistent'")
	}
}

func TestRemove(t *testing.T) {
	s := NewSet[string]()
	s.Add("test")
	s.Remove("test")
	if s.Contains("test") {
		t.Error("Expected 'test' to be removed")
	}

	// 存在しない要素の削除
	s.Remove("nonexistent") // エラーが発生しないことを確認
}

func TestToSlice(t *testing.T) {
	s := NewSet[string]()
	s.Add("one")
	s.Add("two")
	s.Add("three")

	slice := s.ToSlice()
	if len(slice) != 3 {
		t.Errorf("Expected slice length 3, got %d", len(slice))
	}

	// スライスに全ての要素が含まれているか確認
	elements := make(map[string]bool)
	for _, e := range slice {
		elements[e] = true
	}

	expectedElements := []string{"one", "two", "three"}
	for _, e := range expectedElements {
		if !elements[e] {
			t.Errorf("Expected slice to contain %s", e)
		}
	}
}

// 空のセットに対するテスト
func TestEmptySet(t *testing.T) {
	s := NewSet[string]()
	slice := s.ToSlice()
	if len(slice) != 0 {
		t.Errorf("Expected empty slice, got %v", slice)
	}
}

// 大量のデータに対するテスト
func TestLargeDataSet(t *testing.T) {
	s := NewSet[int]()
	for i := 0; i < 1000; i++ {
		s.Add(i)
	}

	if len(s) != 1000 {
		t.Errorf("Expected set with 1000 elements, got %d", len(s))
	}

	for i := 0; i < 1000; i++ {
		if !s.Contains(i) {
			t.Errorf("Expected set to contain %d", i)
		}
	}
}

// 異なる型でのテスト
func TestDifferentTypes(t *testing.T) {
	// 整数型のセット
	intSet := NewSet[int]()
	intSet.Add(1)
	intSet.Add(2)
	if !intSet.Contains(1) || !intSet.Contains(2) {
		t.Error("Expected int set to contain 1 and 2")
	}

	// 文字列型のセット
	stringSet := NewSet[string]()
	stringSet.Add("hello")
	stringSet.Add("world")
	if !stringSet.Contains("hello") || !stringSet.Contains("world") {
		t.Error("Expected string set to contain 'hello' and 'world'")
	}

	// 浮動小数点型のセット
	floatSet := NewSet[float64]()
	floatSet.Add(3.14)
	floatSet.Add(2.71)
	if !floatSet.Contains(3.14) || !floatSet.Contains(2.71) {
		t.Error("Expected float set to contain 3.14 and 2.71")
	}
}

// エッジケースのテスト
func TestEdgeCases(t *testing.T) {
	s := NewSet[string]()

	// 空文字列の追加
	s.Add("")
	if !s.Contains("") {
		t.Error("Expected set to contain empty string")
	}

	// 同じ要素の複数回追加
	s.Add("duplicate")
	s.Add("duplicate")
	slice := s.ToSlice()
	count := 0
	for _, e := range slice {
		if e == "duplicate" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Expected 'duplicate' to appear once, got %d", count)
	}

	// 削除後の再追加
	s.Remove("test")
	s.Add("test")
	if !s.Contains("test") {
		t.Error("Expected set to contain 'test' after re-adding")
	}
}

// 新しいテスト（拡張機能用）
type TestItem struct {
	ID   int
	Name string
}

func TestNewSetKV(t *testing.T) {
	set := NewSetKV[int, TestItem]()
	if len(set) != 0 {
		t.Errorf("Expected empty set, got %v", set)
	}

	items := []TestItem{
		{ID: 1, Name: "Item 1"},
		{ID: 2, Name: "Item 2"},
		{ID: 3, Name: "Item 3"},
	}

	for _, item := range items {
		set[item.ID] = item
	}

	for _, item := range items {
		if !set.Contains(item.ID) {
			t.Errorf("Expected set to contain key %d", item.ID)
		}

		value, exists := set[item.ID]
		if !exists {
			t.Errorf("Expected set to contain key %d", item.ID)
		}

		if value.Name != item.Name {
			t.Errorf("Expected value with name %s, got %s", item.Name, value.Name)
		}
	}
}

func TestFromSlice(t *testing.T) {
	items := []TestItem{
		{ID: 1, Name: "Item 1"},
		{ID: 2, Name: "Item 2"},
		{ID: 3, Name: "Item 3"},
	}

	set := FromSlice(items, func(item TestItem) int {
		return item.ID
	})

	if len(set) != 3 {
		t.Errorf("Expected set with 3 elements, got %d", len(set))
	}

	for _, item := range items {
		if !set.Contains(item.ID) {
			t.Errorf("Expected set to contain key %d", item.ID)
		}

		value, exists := set[item.ID]
		if !exists {
			t.Errorf("Expected set to contain key %d", item.ID)
		}

		if value.Name != item.Name {
			t.Errorf("Expected value with name %s, got %s", item.Name, value.Name)
		}
	}
}

func TestSetToSliceValues(t *testing.T) {
	s := NewSetKV[int, string]()
	s[1] = "One"
	s[2] = "Two"
	s[3] = "Three"

	// Get values from the map
	values := make([]string, 0, len(s))
	for _, v := range s {
		values = append(values, v)
	}

	if len(values) != 3 {
		t.Errorf("Expected slice length 3, got %d", len(values))
	}

	// スライスに全ての値が含まれているか確認
	valueMap := make(map[string]bool)
	for _, v := range values {
		valueMap[v] = true
	}

	expectedValues := []string{"One", "Two", "Three"}
	for _, v := range expectedValues {
		if !valueMap[v] {
			t.Errorf("Expected slice to contain %s", v)
		}
	}
}

func TestKeys(t *testing.T) {
	s := NewSetKV[string, int]()
	s["one"] = 1
	s["two"] = 2
	s["three"] = 3

	keys := s.Keys()
	if len(keys) != 3 {
		t.Errorf("Expected keys length 3, got %d", len(keys))
	}

	// キーが全て含まれているか確認
	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}

	expectedKeys := []string{"one", "two", "three"}
	for _, k := range expectedKeys {
		if !keyMap[k] {
			t.Errorf("Expected keys to contain %s", k)
		}
	}
}

func TestSetKVContains(t *testing.T) {
	s := NewSetKV[int, string]()
	s[1] = "One"
	s[2] = "Two"

	if !s.Contains(1) {
		t.Error("Expected set to contain key 1")
	}
	if !s.Contains(2) {
		t.Error("Expected set to contain key 2")
	}
	if s.Contains(3) {
		t.Error("Expected set to not contain key 3")
	}
}

func TestSetKVAddKV(t *testing.T) {
	s := NewSetKV[string, int]()
	s["one"] = 1

	if !s.Contains("one") {
		t.Error("Expected set to contain key 'one'")
	}

	value, exists := s["one"]
	if !exists {
		t.Error("Expected set to contain key 'one'")
	}
	if value != 1 {
		t.Errorf("Expected value 1, got %d", value)
	}

	// 既存のキーに対する上書き
	s["one"] = 100
	value, exists = s["one"]
	if !exists {
		t.Error("Expected set to contain key 'one'")
	}
	if value != 100 {
		t.Errorf("Expected value 100, got %d", value)
	}
}

func TestSetKVRemoveKey(t *testing.T) {
	s := NewSetKV[string, int]()
	s["one"] = 1
	s["two"] = 2

	delete(s, "one")
	if s.Contains("one") {
		t.Error("Expected 'one' to be removed")
	}
	if !s.Contains("two") {
		t.Error("Expected 'two' to still be in set")
	}

	// 存在しない要素の削除
	delete(s, "three") // エラーが発生しないことを確認
}

func TestDifferenceKeys(t *testing.T) {
	s1 := NewSetKV[string, int]()
	s1["one"] = 1
	s1["two"] = 2
	s1["three"] = 3

	s2 := NewSetKV[string, string]()
	s2["two"] = "Two"
	s2["four"] = "Four"

	diff := DifferenceKeys(s1, s2)

	// 結果をソートして比較
	sort.Strings(diff)
	expected := []string{"one", "three"}
	sort.Strings(expected)

	if len(diff) != 2 {
		t.Errorf("Expected difference keys length 2, got %d", len(diff))
	}

	for i, k := range expected {
		if diff[i] != k {
			t.Errorf("Expected key %s at position %d, got %s", k, i, diff[i])
		}
	}
}

func TestIntersectionKeys(t *testing.T) {
	s1 := NewSetKV[string, int]()
	s1["one"] = 1
	s1["two"] = 2
	s1["three"] = 3

	s2 := NewSetKV[string, string]()
	s2["two"] = "Two"
	s2["three"] = "Three"
	s2["four"] = "Four"

	intersection := IntersectionKeys(s1, s2)

	// 結果をソートして比較
	sort.Strings(intersection)
	expected := []string{"two", "three"}
	sort.Strings(expected)

	if len(intersection) != 2 {
		t.Errorf("Expected intersection keys length 2, got %d", len(intersection))
	}

	for i, k := range expected {
		if intersection[i] != k {
			t.Errorf("Expected key %s at position %d, got %s", k, i, intersection[i])
		}
	}
}

func TestDifference(t *testing.T) {
	s1 := NewSetKV[string, int]()
	s1["one"] = 1
	s1["two"] = 2
	s1["three"] = 3

	s2 := NewSetKV[string, int]()
	s2["two"] = 20
	s2["four"] = 4

	diff := s1.Difference(s2)

	if diff.Size() != 2 {
		t.Errorf("Expected difference size 2, got %d", diff.Size())
	}

	if !diff.Contains("one") || !diff.Contains("three") {
		t.Error("Expected difference to contain 'one' and 'three'")
	}

	if diff.Contains("two") || diff.Contains("four") {
		t.Error("Expected difference to not contain 'two' or 'four'")
	}

	// 値の確認
	if v, _ := diff["one"]; v != 1 {
		t.Errorf("Expected value 1 for key 'one', got %d", v)
	}
	if v, _ := diff["three"]; v != 3 {
		t.Errorf("Expected value 3 for key 'three', got %d", v)
	}
}

func TestIntersection(t *testing.T) {
	s1 := NewSetKV[string, int]()
	s1["one"] = 1
	s1["two"] = 2
	s1["three"] = 3

	s2 := NewSetKV[string, int]()
	s2["two"] = 20
	s2["three"] = 30
	s2["four"] = 4

	intersection := s1.Intersection(s2)

	if intersection.Size() != 2 {
		t.Errorf("Expected intersection size 2, got %d", intersection.Size())
	}

	if !intersection.Contains("two") || !intersection.Contains("three") {
		t.Error("Expected intersection to contain 'two' and 'three'")
	}

	if intersection.Contains("one") || intersection.Contains("four") {
		t.Error("Expected intersection to not contain 'one' or 'four'")
	}

	// 値の確認（s1の値が保持されるはず）
	if v, _ := intersection["two"]; v != 2 {
		t.Errorf("Expected value 2 for key 'two', got %d", v)
	}
	if v, _ := intersection["three"]; v != 3 {
		t.Errorf("Expected value 3 for key 'three', got %d", v)
	}
}

func TestUnion(t *testing.T) {
	s1 := NewSetKV[string, int]()
	s1["one"] = 1
	s1["two"] = 2

	s2 := NewSetKV[string, int]()
	s2["two"] = 20
	s2["three"] = 3

	union := s1.Union(s2)

	if union.Size() != 3 {
		t.Errorf("Expected union size 3, got %d", union.Size())
	}

	if !union.Contains("one") || !union.Contains("two") || !union.Contains("three") {
		t.Error("Expected union to contain 'one', 'two', and 'three'")
	}

	// 値の確認（重複する場合はs2の値が優先されるはず）
	if v, _ := union["one"]; v != 1 {
		t.Errorf("Expected value 1 for key 'one', got %d", v)
	}
	if v, _ := union["two"]; v != 20 {
		t.Errorf("Expected value 20 for key 'two', got %d", v)
	}
	if v, _ := union["three"]; v != 3 {
		t.Errorf("Expected value 3 for key 'three', got %d", v)
	}
}

func TestSize(t *testing.T) {
	s := NewSetKV[int, string]()
	if s.Size() != 0 {
		t.Errorf("Expected size 0, got %d", s.Size())
	}

	s[1] = "One"
	s[2] = "Two"
	if s.Size() != 2 {
		t.Errorf("Expected size 2, got %d", s.Size())
	}

	delete(s, 1)
	if s.Size() != 1 {
		t.Errorf("Expected size 1, got %d", s.Size())
	}
}

func TestIsEmpty(t *testing.T) {
	s := NewSetKV[int, string]()
	if !s.IsEmpty() {
		t.Error("Expected set to be empty")
	}

	s[1] = "One"
	if s.IsEmpty() {
		t.Error("Expected set to not be empty")
	}

	delete(s, 1)
	if !s.IsEmpty() {
		t.Error("Expected set to be empty after removing all elements")
	}
}

func TestClear(t *testing.T) {
	s := NewSetKV[string, int]()
	s["one"] = 1
	s["two"] = 2
	s["three"] = 3

	s.Clear()
	if !s.IsEmpty() {
		t.Error("Expected set to be empty after Clear")
	}
	if s.Size() != 0 {
		t.Errorf("Expected size 0 after Clear, got %d", s.Size())
	}
}

func TestForEach(t *testing.T) {
	s := NewSetKV[string, int]()
	s["one"] = 1
	s["two"] = 2
	s["three"] = 3

	sum := 0
	keys := make([]string, 0)
	s.ForEach(func(k string, v int) {
		sum += v
		keys = append(keys, k)
	})

	if sum != 6 {
		t.Errorf("Expected sum 6, got %d", sum)
	}

	if len(keys) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(keys))
	}

	// キーが全て含まれているか確認
	keyMap := make(map[string]bool)
	for _, k := range keys {
		keyMap[k] = true
	}

	expectedKeys := []string{"one", "two", "three"}
	for _, k := range expectedKeys {
		if !keyMap[k] {
			t.Errorf("Expected keys to contain %s", k)
		}
	}
}

func TestFilter(t *testing.T) {
	s := NewSetKV[string, int]()
	s["one"] = 1
	s["two"] = 2
	s["three"] = 3
	s["four"] = 4

	// 偶数の値だけをフィルタリング
	filtered := s.Filter(func(k string, v int) bool {
		return v%2 == 0
	})

	if filtered.Size() != 2 {
		t.Errorf("Expected filtered size 2, got %d", filtered.Size())
	}

	if !filtered.Contains("two") || !filtered.Contains("four") {
		t.Error("Expected filtered to contain 'two' and 'four'")
	}

	if filtered.Contains("one") || filtered.Contains("three") {
		t.Error("Expected filtered to not contain 'one' or 'three'")
	}
}
