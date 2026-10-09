package invariants

import (
	"reflect"
	"testing"
)

type object = map[string]any

func jsonValue(t *testing.T, value any) any {
	t.Helper()
	return jsonCompatible(reflect.ValueOf(value))
}

func jsonCompatible(value reflect.Value) any {
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Interface || value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		return jsonCompatible(value.Elem())
	}
	switch value.Kind() {
	case reflect.Map:
		if value.IsNil() {
			return nil
		}
		result := object{}
		for _, key := range value.MapKeys() {
			result[key.String()] = jsonCompatible(value.MapIndex(key))
		}
		return result
	case reflect.Slice, reflect.Array:
		if value.Kind() == reflect.Slice && value.IsNil() {
			return nil
		}
		result := make([]any, value.Len())
		for i := range result {
			result[i] = jsonCompatible(value.Index(i))
		}
		return result
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(value.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(value.Uint())
	case reflect.Float32, reflect.Float64:
		return value.Float()
	default:
		return value.Interface()
	}
}
