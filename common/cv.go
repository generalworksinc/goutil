package gw_common

import (
	"reflect"

	gw_errors "github.com/generalworksinc/goutil/errors"
)

func Ip(i int) *int {
	return &i
}

func Pointer[T any](value T) *T {
	return &value
}

// 循環参照を検出するための構造体
type cycleDetector struct {
	visited map[cloneVisit]bool
}

type cloneVisit struct {
	typ  reflect.Type
	addr uintptr
	len  int
	cap  int
}

func visitFor(val reflect.Value) cloneVisit {
	visit := cloneVisit{typ: val.Type(), addr: val.Pointer()}
	if val.Kind() == reflect.Slice {
		visit.len, visit.cap = val.Len(), val.Cap()
	}
	return visit
}

// 新しい循環参照検出器を作成
func newCycleDetector() *cycleDetector {
	return &cycleDetector{
		visited: make(map[cloneVisit]bool),
	}
}

// 循環参照は現在たどっている経路上の値だけを対象にする。
func (cd *cycleDetector) check(val reflect.Value) bool {
	visit := visitFor(val)
	if cd.visited[visit] {
		return true // 循環参照を検出
	}
	cd.visited[visit] = true
	return false
}

func (cd *cycleDetector) leave(val reflect.Value) {
	delete(cd.visited, visitFor(val))
}

func clonedValue(value interface{}, typ reflect.Type) (reflect.Value, error) {
	if value == nil {
		return reflect.Zero(typ), nil
	}
	result := reflect.ValueOf(value)
	if !result.Type().AssignableTo(typ) {
		return reflect.Value{}, gw_errors.Errorf("cloned value type %s is not assignable to %s", result.Type(), typ)
	}
	return result, nil
}

// deepCopyWithCycleCheck は循環参照チェック付きのdeepCopy
func deepCopyWithCycleCheck(src interface{}, cd *cycleDetector) (interface{}, error) {
	if src == nil {
		return nil, nil
	}
	//初回、循環参照検出器がない場合の処理を追加
	if cd == nil {
		cd = newCycleDetector()
	}

	return copyRecursive(reflect.ValueOf(src), cd)
}

func copyRecursive(val reflect.Value, cd *cycleDetector) (interface{}, error) {

	switch val.Kind() {
	case reflect.Interface:
		if val.IsNil() {
			return nil, nil
		}
		return copyRecursive(val.Elem(), cd)

	// 1. ポインタの場合: 中身をたどって再帰的にコピー
	case reflect.Pointer:
		if val.IsNil() {
			return reflect.Zero(val.Type()).Interface(), nil
		}
		//cyclecheck
		if cd.check(val) {
			return nil, gw_errors.New("circular reference detected")
		}
		defer cd.leave(val)
		// ポインタが指す値の型を元に新しい値を作り
		// 再帰的に値をコピーしてからアサインする
		newPtr := reflect.New(val.Elem().Type())
		clonedChild, err := copyRecursive(val.Elem(), cd)
		if err != nil {
			return nil, gw_errors.Wrap(err)
		}
		childValue, err := clonedValue(clonedChild, newPtr.Elem().Type())
		if err != nil {
			return nil, gw_errors.Wrap(err)
		}
		newPtr.Elem().Set(childValue)
		return newPtr.Interface(), nil

	// 2. 構造体の場合: フィールドを順にたどって再帰的にセット
	case reflect.Struct:

		newStruct := reflect.New(val.Type()).Elem()
		// フィールド数だけループ
		for i := 0; i < val.NumField(); i++ {
			fieldVal := val.Field(i)
			if fieldVal.CanInterface() {
				clonedField, err := copyRecursive(fieldVal, cd)
				if err != nil {
					return nil, gw_errors.Wrap(err)
				}
				clonedVal, err := clonedValue(clonedField, newStruct.Field(i).Type())
				if err != nil {
					return nil, gw_errors.Wrap(err)
				}
				newStruct.Field(i).Set(clonedVal)
			}
		}
		return newStruct.Interface(), nil

	// 3. スライスの場合: 全要素を再帰的にコピー
	case reflect.Slice:
		if val.IsNil() {
			return reflect.Zero(val.Type()).Interface(), nil
		}
		//cyclecheck
		if cd.check(val) {
			return nil, gw_errors.New("circular reference detected")
		}
		defer cd.leave(val)
		newSlice := reflect.MakeSlice(val.Type(), val.Len(), val.Cap())
		for i := 0; i < val.Len(); i++ {
			clonedChild, err := copyRecursive(val.Index(i), cd)
			if err != nil {
				return nil, gw_errors.Wrap(err)
			}
			childValue, err := clonedValue(clonedChild, newSlice.Index(i).Type())
			if err != nil {
				return nil, gw_errors.Wrap(err)
			}
			newSlice.Index(i).Set(childValue)
		}
		return newSlice.Interface(), nil

	// 4. マップの場合: 全エントリを再帰的にコピー
	case reflect.Map:
		if val.IsNil() {
			return reflect.Zero(val.Type()).Interface(), nil
		}
		//cyclecheck
		if cd.check(val) {
			return nil, gw_errors.New("circular reference detected")
		}
		defer cd.leave(val)
		newMap := reflect.MakeMapWithSize(val.Type(), val.Len())
		for _, key := range val.MapKeys() {
			// mapのkeyは原則としてImmutableなためそのまま使用可能
			clonedKey, err := copyRecursive(key, cd)
			if err != nil {
				return nil, gw_errors.Wrap(err)
			}
			clonedVal, err := copyRecursive(val.MapIndex(key), cd)
			if err != nil {
				return nil, gw_errors.Wrap(err)
			}
			keyValue, err := clonedValue(clonedKey, val.Type().Key())
			if err != nil {
				return nil, gw_errors.Wrap(err)
			}
			mapValue, err := clonedValue(clonedVal, val.Type().Elem())
			if err != nil {
				return nil, gw_errors.Wrap(err)
			}
			newMap.SetMapIndex(keyValue, mapValue)
		}
		return newMap.Interface(), nil

	// 5. 配列の場合: スライスと同様に要素をコピー
	case reflect.Array:
		newArray := reflect.New(val.Type()).Elem()
		for i := 0; i < val.Len(); i++ {
			clonedChild, err := copyRecursive(val.Index(i), cd)
			if err != nil {
				return nil, gw_errors.Wrap(err)
			}
			childValue, err := clonedValue(clonedChild, newArray.Index(i).Type())
			if err != nil {
				return nil, gw_errors.Wrap(err)
			}
			newArray.Index(i).Set(childValue)
		}
		return newArray.Interface(), nil

	// 6. その他（数値型、文字列など）はそのまま返す
	default:
		// 基本型 (bool, int, float, string, chan, func など) は
		// 「値のまま」コピーすればOK
		return val.Interface(), nil
	}
}

// DeepClone は任意の値 v を深くコピーした新しい値を返します。
// (DeepClone(*T) のように呼び出されると *T のコピーが、
//
//	DeepClone(T) であれば T のコピーがそれぞれ返ります)
//
// 非エクスポートフィールドはコピーされません
func Clone[T any](v T) (T, error) {
	// deepCopyはinterface{}を返すため型アサーションを使ってTに戻す
	cloned, err := deepCopyWithCycleCheck(v, nil)
	if err != nil {
		return v, gw_errors.Wrap(err)
	}
	if cloned == nil {
		// T がポインタやスライス等で nil になるケース
		var zero T
		return zero, nil
	}
	return cloned.(T), nil
}

// Clone: *T を深くコピーして新しい *T を返す
// 非エクスポートフィールドはコピーされません
func CloneP[T any](value *T) (*T, error) {
	if value == nil {
		return nil, nil
	}
	c, err := Clone(*value)
	if err != nil {
		return nil, gw_errors.Wrap(err)
	}
	return &c, nil
}

// CloneSliceP: []*T の要素を再帰的に深いコピーを行い、
// 新しい []*T スライスを返す
// 非エクスポートフィールドはコピーされません
func CloneSliceP[T any](value []*T) ([]*T, error) {
	if value == nil {
		return nil, nil
	}
	cloned := make([]*T, len(value))
	for i, v := range value {
		var err error
		cloned[i], err = CloneP(v) // *TをCloneすると内部まで深いコピー
		if err != nil {
			return nil, gw_errors.Wrap(err)
		}
	}
	return cloned, nil
}

// CloneSlice: []T を深いコピーして新しい []T を返す
// 非エクスポートフィールドはコピーされません
func CloneSlice[T any](value []T) ([]T, error) {
	if value == nil {
		return nil, nil
	}
	// スライス自体を作り直した上で、各要素をDeepClone
	cloned := make([]T, len(value))
	for i, v := range value {
		var err error
		cloned[i], err = Clone(v)
		if err != nil {
			return nil, gw_errors.Wrap(err)
		}
	}
	return cloned, nil
}

// CloneMapP: map[K]*V を深いコピーして新しい map[K]*V を返す
// 非エクスポートフィールドはコピーされません
func CloneMapP[K comparable, V any](value map[K]*V) (map[K]*V, error) {
	if value == nil {
		return nil, nil
	}
	clonedMap := make(map[K]*V, len(value))
	for k, v := range value {
		var err error
		clonedMap[k], err = CloneP(v) // *VをClone→(内部まで深いコピー)
		if err != nil {
			return nil, gw_errors.Wrap(err)
		}
	}
	return clonedMap, nil
}

// CloneMap: map[K]V を深いコピーして新しい map[K]V を返す
// 非エクスポートフィールドはコピーされません
func CloneMap[K comparable, V any](value map[K]V) (map[K]V, error) {
	if value == nil {
		return nil, nil
	}
	clonedMap := make(map[K]V, len(value))
	for k, v := range value {
		// mapのvalueをCloneしてセット
		var err error
		clonedMap[k], err = Clone(v)
		if err != nil {
			return nil, gw_errors.Wrap(err)
		}
	}
	return clonedMap, nil
}
