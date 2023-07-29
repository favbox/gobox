package assert

import (
	"reflect"
	"testing"
)

// Assert .
func Assert(t testing.TB, cond bool, val ...any) {
	t.Helper()
	if !cond {
		if len(val) > 0 {
			val = append([]any{"assertion failed:"}, val...)
		} else {
			t.Fatal("assertion failed")
		}
	}
}

// Assertf .
func Assertf(t testing.TB, cond bool, format string, val ...any) {
	t.Helper()
	if !cond {
		t.Fatalf(format, val...)
	}
}

// DeepEqual .
func DeepEqual(t testing.TB, excepted, actual any) {
	t.Helper()
	if !reflect.DeepEqual(actual, excepted) {
		t.Fatalf("断言失败，不期望：%v，期望：%v", actual, excepted)
	}
}

// Nil .
func Nil(t testing.TB, data any) {
	t.Helper()
	if data == nil {
		return
	}
	if !reflect.ValueOf(data).IsNil() {
		t.Fatalf("assertion failed, unexpected: %v, excepted: nil", data)
	}
}

// NotNil .
func NotNil(t testing.TB, data any) {
	t.Helper()
	if data == nil {
		return
	}
	if reflect.ValueOf(data).IsNil() {
		t.Fatalf("assertion failed, unexpected: %v, excepted: nil", data)
	}
}

// NotEqual .
func NotEqual(t testing.TB, expected, actual interface{}) {
	t.Helper()
	if expected == nil || actual == nil {
		if expected == actual {
			t.Fatalf("assertion failed, unexpected: %v, expected: %v", actual, expected)
		}
	}

	if reflect.DeepEqual(actual, expected) {
		t.Fatalf("assertion failed, unexpected: %v, expected: %v", actual, expected)
	}
}

// True .
func True(t testing.TB, obj any) {
	DeepEqual(t, true, obj)
}

// False .
func False(t testing.TB, obj any) {
	DeepEqual(t, false, obj)
}

// Panic .
func Panic(t testing.TB, fn func()) {
	t.Helper()
	defer func() {
		if err := recover(); err == nil {
			t.Fatal("assertion failed: did not panic")
		}
	}()
	fn()
}

// NotPanic .
func NotPanic(t testing.TB, fn func()) {
	t.Helper()
	defer func() {
		if err := recover(); err != nil {
			t.Fatal("assertion failed: did panic")
		}
	}()
	fn()
}
