// Package valuecopy detaches acyclic JSON-style maps and slices without numeric coercion.
// Reflection is restricted to dynamic container values; scalars retain their exact types.
package valuecopy

import "reflect"

func Copy[T any](value T) T {
	result := reflect.ValueOf(&value).Elem()
	result.Set(copyValue(result))
	return value
}

func copyValue(value reflect.Value) reflect.Value {
	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		result := reflect.New(value.Type()).Elem()
		result.Set(copyValue(value.Elem()))
		return result
	case reflect.Map:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		result := reflect.MakeMapWithSize(value.Type(), value.Len())
		iter := value.MapRange()
		for iter.Next() {
			result.SetMapIndex(iter.Key(), copyValue(iter.Value()))
		}
		return result
	case reflect.Slice:
		if value.IsNil() {
			return reflect.Zero(value.Type())
		}
		result := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
		for i := 0; i < value.Len(); i++ {
			result.Index(i).Set(copyValue(value.Index(i)))
		}
		return result
	default:
		return value
	}
}
