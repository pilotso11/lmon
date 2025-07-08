package common

import "testing"

func TestIsNil(t *testing.T) {
	// --- Test Setup ---
	var nilPtr *int
	var nonNilPtr = new(int)

	var nilMap map[string]string
	var nonNilMap = make(map[string]string)

	var nilSlice []int
	var nonNilSlice = make([]int, 0)

	var nilChan chan bool
	var nonNilChan = make(chan bool)

	var nilFunc func()
	var nonNilFunc = func() {}

	var nilArrayPtr *[5]int
	var nonNilArray [5]int

	// An interface can be non-nil itself but hold a nil value.
	// This is a classic Go "gotcha" that IsNil should handle.
	var typedNilInterface any = (*int)(nil)

	// --- Test Cases ---
	testCases := []struct {
		name     string
		input    any
		expected bool
	}{
		{name: "literal nil", input: nil, expected: true},
		{name: "typed nil interface", input: typedNilInterface, expected: true},
		{name: "zero int", input: 0, expected: false},
		{name: "empty string", input: "", expected: false},
		{name: "zero struct", input: struct{}{}, expected: false},
		{name: "nil pointer", input: nilPtr, expected: true},
		{name: "non-nil pointer", input: nonNilPtr, expected: false},
		{name: "nil map", input: nilMap, expected: true},
		{name: "non-nil map", input: nonNilMap, expected: false},
		{name: "nil slice", input: nilSlice, expected: true},
		{name: "non-nil slice", input: nonNilSlice, expected: false},
		{name: "nil channel", input: nilChan, expected: true},
		{name: "non-nil channel", input: nonNilChan, expected: false},
		{name: "nil function", input: nilFunc, expected: true},
		{name: "non-nil function", input: nonNilFunc, expected: false},
		{name: "nil pointer to array", input: nilArrayPtr, expected: true},
		{name: "non-nil pointer to array", input: &nonNilArray, expected: false},
		{name: "non-nil array value", input: nonNilArray, expected: false},
	}

	// --- Test Execution ---
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsNil(tc.input); got != tc.expected {
				// Use %#v to get a more detailed output of the input value
				t.Errorf("IsNil(%#v) = %t; want %t", tc.input, got, tc.expected)
			}
		})
	}
}
