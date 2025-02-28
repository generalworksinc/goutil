package gw_set

import (
	"testing"
	"time"
)

// カスタム構造体のテスト用
type testStruct struct {
	ID   int
	Name string
}

func TestNewSet(t *testing.T) {
	s := NewSet[int]()
	if len(s) != 0 {
		t.Errorf("Expected empty set, got size %d", len(s))
	}
}

func TestAdd(t *testing.T) {
	s := NewSet[string]()
	s.Add("item1")

	if !s.Contains("item1") {
		t.Error("Expected set to contain 'item1'")
	}

	// Add duplicate item
	s.Add("item1")
	if len(s) != 1 {
		t.Errorf("Expected set size 1, got %d", len(s))
	}
}

func TestContains(t *testing.T) {
	s := NewSet[int]()
	s.Add(1)
	s.Add(2)

	if !s.Contains(1) {
		t.Error("Expected set to contain 1")
	}

	if !s.Contains(2) {
		t.Error("Expected set to contain 2")
	}

	if s.Contains(3) {
		t.Error("Expected set to not contain 3")
	}
}

func TestRemove(t *testing.T) {
	s := NewSet[string]()
	s.Add("item1")
	s.Add("item2")

	s.Remove("item1")
	if s.Contains("item1") {
		t.Error("Expected 'item1' to be removed")
	}

	if !s.Contains("item2") {
		t.Error("Expected 'item2' to still be in set")
	}

	// Remove non-existent item should not cause error
	s.Remove("item3")
}

func TestToSlice(t *testing.T) {
	s := NewSet[int]()
	s.Add(1)
	s.Add(2)
	s.Add(3)

	slice := s.ToSlice()

	if len(slice) != 3 {
		t.Errorf("Expected slice length 3, got %d", len(slice))
	}

	// Check all elements are in the slice
	found := make(map[int]bool)
	for _, v := range slice {
		found[v] = true
	}

	if !found[1] || !found[2] || !found[3] {
		t.Error("Not all elements from set found in slice")
	}
}

// 空のセットに対するテスト
func TestEmptySet(t *testing.T) {
	// 空のセットからToSlice
	s := NewSet[int]()
	slice := s.ToSlice()
	if len(slice) != 0 {
		t.Errorf("Expected empty slice from empty set, got length %d", len(slice))
	}

	// 空のセットからの要素削除
	s.Remove(1) // 存在しない要素の削除
	if len(s) != 0 {
		t.Errorf("Expected empty set after removing from empty set, got size %d", len(s))
	}

	// 空のセットに対するContains
	if s.Contains(1) {
		t.Error("Expected empty set to not contain any elements")
	}
}

// 大量のデータに対するテスト
func TestLargeDataSet(t *testing.T) {
	s := NewSet[int]()

	// 1000個の要素を追加
	for i := 0; i < 1000; i++ {
		s.Add(i)
	}

	if len(s) != 1000 {
		t.Errorf("Expected set size 1000, got %d", len(s))
	}

	// すべての要素が含まれていることを確認
	for i := 0; i < 1000; i++ {
		if !s.Contains(i) {
			t.Errorf("Expected set to contain %d", i)
			break
		}
	}

	// ToSliceの結果を確認
	slice := s.ToSlice()
	if len(slice) != 1000 {
		t.Errorf("Expected slice length 1000, got %d", len(slice))
	}
}

// 異なる型でのテスト
func TestDifferentTypes(t *testing.T) {
	// float64型
	floatSet := NewSet[float64]()
	floatSet.Add(1.1)
	floatSet.Add(2.2)

	if !floatSet.Contains(1.1) {
		t.Error("Expected float set to contain 1.1")
	}

	// bool型
	boolSet := NewSet[bool]()
	boolSet.Add(true)

	if !boolSet.Contains(true) {
		t.Error("Expected bool set to contain true")
	}
	if boolSet.Contains(false) {
		t.Error("Expected bool set to not contain false")
	}

	// time.Time型
	timeSet := NewSet[time.Time]()
	now := time.Now()
	timeSet.Add(now)

	if !timeSet.Contains(now) {
		t.Error("Expected time set to contain now")
	}

	// カスタム構造体
	structSet := NewSet[testStruct]()
	item1 := testStruct{ID: 1, Name: "Test1"}
	item2 := testStruct{ID: 2, Name: "Test2"}

	structSet.Add(item1)

	if !structSet.Contains(item1) {
		t.Error("Expected struct set to contain item1")
	}
	if structSet.Contains(item2) {
		t.Error("Expected struct set to not contain item2")
	}
}

// エッジケースのテスト
func TestEdgeCases(t *testing.T) {
	// ゼロ値の追加と削除
	intSet := NewSet[int]()
	intSet.Add(0)

	if !intSet.Contains(0) {
		t.Error("Expected set to contain zero value (0)")
	}

	intSet.Remove(0)
	if intSet.Contains(0) {
		t.Error("Expected set to not contain zero value after removal")
	}

	// 文字列のゼロ値
	strSet := NewSet[string]()
	strSet.Add("")

	if !strSet.Contains("") {
		t.Error("Expected set to contain empty string")
	}

	// ポインタのゼロ値
	ptrSet := NewSet[*int]()
	var nilPtr *int
	ptrSet.Add(nilPtr)

	if !ptrSet.Contains(nilPtr) {
		t.Error("Expected set to contain nil pointer")
	}
}
