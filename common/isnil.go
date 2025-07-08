package common

import "reflect"

// IsNil checks if a given interface is nil.
// It correctly handles "typed nils" for pointers, maps, slices, channels,
// functions, and interfaces, in addition to plain nil.
func IsNil(i any) bool {
	if i == nil {
		return true
	}

	// Use reflect to check for "typed" nils
	v := reflect.ValueOf(i)
	switch v.Kind() {
	case reflect.Ptr, reflect.Map, reflect.Slice, reflect.Chan, reflect.Func, reflect.Interface:
		return v.IsNil()
	default:
		return false
	}
}
