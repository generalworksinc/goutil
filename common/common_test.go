package gw_common

import (
	"reflect"
	"strings"
	"testing"
)

func TestClone_BasicTypes(t *testing.T) {
	t.Run("int", func(t *testing.T) {
		original := 42
		cloned, err := Clone(original)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cloned != original {
			t.Errorf("basic int mismatch: got %v, want %v", cloned, original)
		}
	})

	t.Run("string", func(t *testing.T) {
		original := "hello"
		cloned, err := Clone(original)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cloned != original {
			t.Errorf("basic string mismatch: got %v, want %v", cloned, original)
		}
	})

	t.Run("bool", func(t *testing.T) {
		original := true
		cloned, err := Clone(original)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cloned != original {
			t.Errorf("basic bool mismatch: got %v, want %v", cloned, original)
		}
	})
}

func TestClone_Pointers(t *testing.T) {
	t.Run("nil pointer", func(t *testing.T) {
		var original *int
		cloned, err := CloneP(original)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cloned != nil {
			t.Errorf("expected nil pointer clone, got non-nil: %v", cloned)
		}
	})

	t.Run("non-nil pointer", func(t *testing.T) {
		originalValue := 100
		originalPtr := &originalValue

		clonedPtr, err := CloneP(originalPtr)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if clonedPtr == nil {
			t.Fatalf("got nil clone pointer, want non-nil")
		}
		if *clonedPtr != *originalPtr {
			t.Errorf("pointer value mismatch: got %v, want %v", *clonedPtr, *originalPtr)
		}
		// アドレスが異なるか（ディープコピーされているか）を一応確認
		if clonedPtr == originalPtr {
			t.Errorf("pointer addresses should differ, got same address %p", clonedPtr)
		}
	})
}

func TestClone_Slices(t *testing.T) {
	t.Run("slice of int", func(t *testing.T) {
		original := []int{1, 2, 3, 4}
		cloned, err := CloneSlice(original)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// 要素数・内容チェック
		if len(cloned) != len(original) {
			t.Fatalf("length mismatch: got %d, want %d", len(cloned), len(original))
		}
		for i := range original {
			if cloned[i] != original[i] {
				t.Errorf("slice element mismatch at index %d: got %v, want %v", i, cloned[i], original[i])
			}
		}

		// スライスのアドレスが異なるか（同じ底層配列を共有していないか）は一概にチェックが難しいですが、
		// ここでは簡単に "appendできるか" などで見分けることがあります。
	})

	t.Run("slice of pointers", func(t *testing.T) {
		v1, v2 := 10, 20
		original := []*int{&v1, &v2}
		cloned, err := CloneSliceP(original)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cloned) != len(original) {
			t.Fatalf("length mismatch: got %d, want %d", len(cloned), len(original))
		}
		// 値のチェック & アドレスが違うかチェック
		for i := range original {
			if cloned[i] == nil {
				t.Fatalf("cloned[%d] is nil, want non-nil", i)
			}
			if *cloned[i] != *original[i] {
				t.Errorf("value mismatch at index %d: got %d, want %d", i, *cloned[i], *original[i])
			}
			if cloned[i] == original[i] {
				t.Errorf("pointer address at index %d should differ, got the same %p", i, cloned[i])
			}
		}
	})
}

func TestClone_Maps(t *testing.T) {
	t.Run("map[K]V", func(t *testing.T) {
		original := map[string]int{"a": 1, "b": 2}
		cloned, err := CloneMap(original)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if len(cloned) != len(original) {
			t.Fatalf("map length mismatch: got %d, want %d", len(cloned), len(original))
		}
		for k, v := range original {
			if cloned[k] != v {
				t.Errorf("value mismatch for key %s: got %d, want %d", k, cloned[k], v)
			}
		}
	})

	t.Run("map[K]*V", func(t *testing.T) {
		v1, v2 := 10, 20
		original := map[string]*int{"x": &v1, "y": &v2}
		cloned, err := CloneMapP(original)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cloned) != len(original) {
			t.Fatalf("map length mismatch: got %d, want %d", len(cloned), len(original))
		}
		for k, origPtr := range original {
			clonedPtr := cloned[k]
			if clonedPtr == nil || origPtr == nil {
				t.Errorf("unexpected nil pointer for key %s (got cloned=%v, original=%v)", k, clonedPtr, origPtr)
				continue
			}
			if *clonedPtr != *origPtr {
				t.Errorf("pointer value mismatch for key %s: got %d, want %d", k, *clonedPtr, *origPtr)
			}
			if clonedPtr == origPtr {
				t.Errorf("pointer address for key %s should differ, got the same %p", k, clonedPtr)
			}
		}
	})
}

func TestClone_Structs(t *testing.T) {
	type SubStruct struct {
		SubVal int
	}
	type TestStruct struct {
		A int
		B string
		C *SubStruct
	}

	t.Run("simple struct", func(t *testing.T) {
		original := TestStruct{
			A: 999,
			B: "test",
			C: &SubStruct{SubVal: 1234},
		}
		cloned, err := Clone(original)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// 値が正しくコピーされているか
		if cloned.A != original.A {
			t.Errorf("field A mismatch: got %d, want %d", cloned.A, original.A)
		}
		if cloned.B != original.B {
			t.Errorf("field B mismatch: got %q, want %q", cloned.B, original.B)
		}
		if cloned.C == nil {
			t.Errorf("field C is nil, want non-nil")
		} else {
			if cloned.C.SubVal != original.C.SubVal {
				t.Errorf("sub-struct SubVal mismatch: got %d, want %d", cloned.C.SubVal, original.C.SubVal)
			}
			// ポインタが別物になっているか
			if cloned.C == original.C {
				t.Error("sub-struct pointer should differ from the original")
			}
		}
	})
}

func TestClone_CycleDetection(t *testing.T) {
	t.Run("self-referential pointer", func(t *testing.T) {
		type Node struct {
			Value int
			Next  *Node
		}
		n1 := &Node{Value: 1}
		n2 := &Node{Value: 2}
		n1.Next = n2
		n2.Next = n1 // 循環参照

		_, err := CloneP(n1)
		if err == nil {
			t.Error("expected circular reference error, got nil")
		} else {
			if !strings.Contains(err.Error(), "circular reference detected") {
				t.Errorf("expected circular reference error message, got %v", err)
			}
		}
	})

	t.Run("self-referential slice", func(t *testing.T) {
		// slice内で自分自身を参照するケース(実際にはやや特殊)
		type S struct {
			Name  string
			Slice []*S
		}
		s1 := &S{Name: "root"}
		s1.Slice = []*S{s1} // 自分自身を要素に持つ -> 循環参照

		_, err := CloneP(s1)
		if err == nil {
			t.Error("expected circular reference error, got nil")
		} else {
			if !strings.Contains(err.Error(), "circular reference detected") {
				t.Errorf("expected circular reference error message, got %v", err)
			}
		}
	})
}

func TestClone_EdgeCases(t *testing.T) {
	t.Run("nil interface", func(t *testing.T) {
		var original interface{}
		cloned, err := Clone(original)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cloned != nil {
			t.Errorf("expected nil, got %v", cloned)
		}
	})

	t.Run("empty slice", func(t *testing.T) {
		original := []int{}
		cloned, err := CloneSlice(original)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cloned) != 0 {
			t.Errorf("expected empty slice, got len=%d", len(cloned))
		}
	})

	t.Run("empty map", func(t *testing.T) {
		original := map[string]int{}
		cloned, err := CloneMap(original)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cloned) != 0 {
			t.Errorf("expected empty map, got len=%d", len(cloned))
		}
	})
}

func TestClone_TypeAssertionFailure(t *testing.T) {
	// このテストは、もし内部ロジックで想定外の型になるケースを検証したい場合に書く例です。
	// 現状のロジックでは通常起こりませんが、想定外の挙動を仕掛けるならこういったチェックも可能です。

	type MyStruct struct {
		X int
	}
	original := MyStruct{X: 123}

	// ここで何らかの方法で "cloned" が MyStruct 以外の型を返すと失敗するはず。
	cloned, err := Clone(original)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 正常系: 型アサーション成功
	if reflect.TypeOf(cloned) != reflect.TypeOf(original) {
		t.Errorf("type mismatch: got %T, want %T", cloned, original)
	}
}
