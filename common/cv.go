package gw_common

import "reflect"

func Ip(i int) *int {
	return &i
}

func Pointer[T any](value T) *T {
	return &value
}

// deepCopy はあらゆる型を再帰的にコピーし、
// その「完全なクローン」を返すための内部関数です。
func deepCopy(src interface{}) interface{} {
	if src == nil {
		return nil
	}

	val := reflect.ValueOf(src)
	switch val.Kind() {

	// 1. ポインタの場合: 中身をたどって再帰的にコピー
	case reflect.Ptr:
		if val.IsNil() {
			return nil
		}
		// ポインタが指す値の型を元に新しい値を作り
		// 再帰的に値をコピーしてからアサインする
		pointedType := val.Elem().Type()
		newPtr := reflect.New(pointedType)
		newPtr.Elem().Set(reflect.ValueOf(deepCopy(val.Elem().Interface())))
		return newPtr.Interface()

	// 2. 構造体の場合: フィールドを順にたどって再帰的にセット
	case reflect.Struct:
		newStruct := reflect.New(val.Type()).Elem()
		// フィールド数だけループ
		for i := 0; i < val.NumField(); i++ {
			fieldVal := val.Field(i)
			if fieldVal.CanInterface() {
				clonedField := deepCopy(fieldVal.Interface())
				newStruct.Field(i).Set(reflect.ValueOf(clonedField))
			} else {
				// エクスポートされていないフィールドなど
				// 直接セットできない場合は値をそのままにするかスキップ
				// ここでは何もしない例としてスキップ
			}
		}
		return newStruct.Interface()

	// 3. スライスの場合: 全要素を再帰的にコピー
	case reflect.Slice:
		if val.IsNil() {
			return nil
		}
		newSlice := reflect.MakeSlice(val.Type(), val.Len(), val.Cap())
		for i := 0; i < val.Len(); i++ {
			itemVal := val.Index(i).Interface()
			newSlice.Index(i).Set(reflect.ValueOf(deepCopy(itemVal)))
		}
		return newSlice.Interface()

	// 4. マップの場合: 全エントリを再帰的にコピー
	case reflect.Map:
		if val.IsNil() {
			return nil
		}
		newMap := reflect.MakeMapWithSize(val.Type(), val.Len())
		for _, key := range val.MapKeys() {
			valKey := key.Interface()
			valVal := val.MapIndex(key).Interface()
			// mapのkeyは原則としてImmutableなためそのまま使用可能
			newMap.SetMapIndex(reflect.ValueOf(deepCopy(valKey)),
				reflect.ValueOf(deepCopy(valVal)))
		}
		return newMap.Interface()

	// 5. 配列の場合: スライスと同様に要素をコピー
	case reflect.Array:
		newArray := reflect.New(val.Type()).Elem()
		for i := 0; i < val.Len(); i++ {
			itemVal := val.Index(i).Interface()
			newArray.Index(i).Set(reflect.ValueOf(deepCopy(itemVal)))
		}
		return newArray.Interface()

	// 6. その他（数値型、文字列など）はそのまま返す
	default:
		// 基本型 (bool, int, float, string, chan, func など) は
		// 「値のまま」コピーすればOK
		return src
	}
}

// DeepClone は任意の値 v を深くコピーした新しい値を返します。
// (DeepClone(*T) のように呼び出されると *T のコピーが、
//
//	DeepClone(T) であれば T のコピーがそれぞれ返ります)
func Clone[T any](v T) T {
	// deepCopyはinterface{}を返すため型アサーションを使ってTに戻す
	cloned := deepCopy(v)
	if cloned == nil {
		// T がポインタやスライス等で nil になるケース
		var zero T
		return zero
	}
	return cloned.(T)
}

// Clone: *T を深くコピーして新しい *T を返す
func CloneP[T any](value *T) *T {
	if value == nil {
		return nil
	}
	c := Clone(*value)
	return &c
}

// CloneSliceP: []*T の要素を再帰的に深いコピーを行い、
// 新しい []*T スライスを返す
func CloneSliceP[T any](value []*T) []*T {
	if value == nil {
		return nil
	}
	cloned := make([]*T, len(value))
	for i, v := range value {
		cloned[i] = CloneP(v) // *TをCloneすると内部まで深いコピー
	}
	return cloned
}

// CloneSlice: []T を深いコピーして新しい []T を返す
func CloneSlice[T any](value []T) []T {
	if value == nil {
		return nil
	}
	// スライス自体を作り直した上で、各要素をDeepClone
	cloned := make([]T, len(value))
	for i, v := range value {
		cloned[i] = Clone(v)
	}
	return cloned
}

// CloneMapP: map[K]*V を深いコピーして新しい map[K]*V を返す
func CloneMapP[K comparable, V any](value map[K]*V) map[K]*V {
	if value == nil {
		return nil
	}
	clonedMap := make(map[K]*V, len(value))
	for k, v := range value {
		clonedMap[k] = CloneP(v) // *VをClone→(内部まで深いコピー)
	}
	return clonedMap
}

// CloneMap: map[K]V を深いコピーして新しい map[K]V を返す
func CloneMap[K comparable, V any](value map[K]V) map[K]V {
	if value == nil {
		return nil
	}
	clonedMap := make(map[K]V, len(value))
	for k, v := range value {
		// mapのvalueをCloneしてセット
		clonedMap[k] = Clone(v)
	}
	return clonedMap
}
