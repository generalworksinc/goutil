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
	visited map[uintptr]bool
}

// 新しい循環参照検出器を作成
func newCycleDetector() *cycleDetector {
	return &cycleDetector{
		visited: make(map[uintptr]bool),
	}
}

// 循環参照をチェック
// スライスの循環チェックに注意。
// arr := []int{1, 2, 3, 4} として、sub := arr[1:3] のように部分スライスを作ると、arr と sub は底層配列を共有するので val.Pointer() が同じになるので循環参照として検出されます。
func (cd *cycleDetector) check(addr uintptr) bool {
	if cd.visited[addr] {
		return true // 循環参照を検出
	}
	cd.visited[addr] = true
	return false
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

	// 1. ポインタの場合: 中身をたどって再帰的にコピー
	case reflect.Ptr:
		if val.IsNil() {
			return nil, nil
		}
		//cyclecheck
		addr := val.Pointer()
		if cd.check(addr) {
			return nil, gw_errors.New("circular reference detected")
		}
		// ポインタが指す値の型を元に新しい値を作り
		// 再帰的に値をコピーしてからアサインする
		newPtr := reflect.New(val.Elem().Type())
		clonedChild, err := copyRecursive(val.Elem(), cd)
		if err != nil {
			return nil, gw_errors.Wrap(err)
		}
		newPtr.Elem().Set(reflect.ValueOf(clonedChild))
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
				// 型の不整合を防ぐため、フィールドの型チェックを行う
				if clonedField == nil {
					// nilの場合はゼロ値をセット
					newStruct.Field(i).Set(reflect.Zero(newStruct.Field(i).Type()))
				} else {
					clonedVal := reflect.ValueOf(clonedField)
					if clonedVal.Type().AssignableTo(newStruct.Field(i).Type()) {
						newStruct.Field(i).Set(clonedVal)
					} else {
						// 型が一致しない場合はゼロ値をセット
						newStruct.Field(i).Set(reflect.Zero(newStruct.Field(i).Type()))
					}
				}
			} else {
				// エクスポートされていないフィールドなど
				// 直接セットできない場合は値をそのままにするかスキップ
				// ここでは何もしない例としてスキップ
			}
		}
		return newStruct.Interface(), nil

	// 3. スライスの場合: 全要素を再帰的にコピー
	case reflect.Slice:
		if val.IsNil() {
			return nil, nil
		}
		//cyclecheck
		addr := val.Pointer()
		if cd.check(addr) {
			return nil, gw_errors.New("circular reference detected")
		}
		newSlice := reflect.MakeSlice(val.Type(), val.Len(), val.Cap())
		for i := 0; i < val.Len(); i++ {
			itemVal := val.Index(i).Interface()
			clonedChild, err := copyRecursive(reflect.ValueOf(itemVal), cd)
			if err != nil {
				return nil, gw_errors.Wrap(err)
			}
			newSlice.Index(i).Set(reflect.ValueOf(clonedChild))
		}
		return newSlice.Interface(), nil

	// 4. マップの場合: 全エントリを再帰的にコピー
	case reflect.Map:
		if val.IsNil() {
			return nil, nil
		}
		//cyclecheck
		addr := val.Pointer()
		if cd.check(addr) {
			return nil, gw_errors.New("circular reference detected")
		}
		newMap := reflect.MakeMapWithSize(val.Type(), val.Len())
		for _, key := range val.MapKeys() {
			valKey := key.Interface()
			valVal := val.MapIndex(key).Interface()
			// mapのkeyは原則としてImmutableなためそのまま使用可能
			clonedKey, err := copyRecursive(reflect.ValueOf(valKey), cd)
			if err != nil {
				return nil, gw_errors.Wrap(err)
			}
			clonedVal, err := copyRecursive(reflect.ValueOf(valVal), cd)
			if err != nil {
				return nil, gw_errors.Wrap(err)
			}
			newMap.SetMapIndex(reflect.ValueOf(clonedKey), reflect.ValueOf(clonedVal))
		}
		return newMap.Interface(), nil

	// 5. 配列の場合: スライスと同様に要素をコピー
	case reflect.Array:
		newArray := reflect.New(val.Type()).Elem()
		for i := 0; i < val.Len(); i++ {
			itemVal := val.Index(i).Interface()
			clonedChild, err := copyRecursive(reflect.ValueOf(itemVal), cd)
			if err != nil {
				return nil, gw_errors.Wrap(err)
			}
			newArray.Index(i).Set(reflect.ValueOf(clonedChild))
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
