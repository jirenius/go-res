package restest

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	res "github.com/jirenius/go-res"
)

// AssertEqualJSON expects that a and b json marshals into equal values, and
// returns true if they do, otherwise logs a fatal error and returns false.
func AssertEqualJSON(t *testing.T, name string, result, expected interface{}, ctx ...interface{}) bool {
	aa, aj := jsonMap(t, result)
	bb, bj := jsonMap(t, expected)

	if !reflect.DeepEqual(aa, bb) {
		t.Fatalf("expected %s to be:\n\t%s\nbut got:\n\t%s%s", name, bj, aj, ctxString(ctx))
		return false
	}

	return true
}

// AssertTrue expects that a condition is true.
func AssertTrue(t *testing.T, expectation string, isTrue bool, ctx ...interface{}) bool {
	if !isTrue {
		t.Fatalf("expected %s%s", expectation, ctxString(ctx))
		return false
	}

	return true
}

// AssertNoError expects that err is nil, otherwise logs an error
// with t.Fatalf
func AssertNoError(t *testing.T, err error, ctx ...interface{}) {
	if err != nil {
		t.Fatalf("expected no error but got:\n%s%s", err, ctxString(ctx))
	}
}

// AssertError expects that err is not nil, otherwise logs an error
// with t.Fatalf
func AssertError(t *testing.T, err error, ctx ...interface{}) {
	if err == nil {
		t.Fatalf("expected an error but got none%s", ctxString(ctx))
	}
}

// AssertResError expects that err is of type *res.Error and matches rerr.
func AssertResError(t *testing.T, err error, rerr *res.Error, ctx ...interface{}) {
	AssertError(t, err, ctx...)
	v, ok := err.(*res.Error)
	if !ok {
		t.Fatalf("expected error to be of type *res.Error%s", ctxString(ctx))
	}
	AssertEqualJSON(t, "error", v, rerr, ctx...)
}

// AssertErrorCode expects that err is of type *res.Error with given code.
func AssertErrorCode(t *testing.T, err error, code string, ctx ...interface{}) {
	AssertError(t, err, ctx...)
	v, ok := err.(*res.Error)
	if !ok {
		t.Fatalf("expected error to be of type *res.Error%s", ctxString(ctx))
	}
	AssertEqualJSON(t, "error code", v.Code, code, ctx...)
}

// AssertPanic expects the callback function to panic, otherwise
// logs an error with t.Errorf
func AssertPanic(t *testing.T, cb func(), ctx ...interface{}) {
	defer func() {
		v := recover()
		if v == nil {
			t.Errorf("expected callback to panic, but it didn't%s", ctxString(ctx))
		}
	}()
	cb()
}

// AssertPanicNoRecover expects the callback function to panic, otherwise
// logs an error with t.Errorf. Does not recover from the panic
func AssertPanicNoRecover(t *testing.T, cb func(), ctx ...interface{}) {
	panicking := true
	defer func() {
		if !panicking {
			t.Errorf(`expected callback to panic, but it didn't%s`, ctxString(ctx))
		}
	}()
	cb()
	panicking = false
}

// AssertNil expects that a value is nil, otherwise it
// logs an error with t.Fatalf.
func AssertNil(t *testing.T, v interface{}, ctx ...interface{}) {
	if v != nil && !reflect.ValueOf(v).IsNil() {
		t.Fatalf("expected non-nil but got nil%s", ctxString(ctx))
	}
}

// AssertNotNil expects that a value is non-nil, otherwise it
// logs an error with t.Fatalf.
func AssertNotNil(t *testing.T, v interface{}, ctx ...interface{}) {
	if v == nil || reflect.ValueOf(v).IsNil() {
		t.Fatalf("expected nil but got %+v%s", v, ctxString(ctx))
	}
}

func ctxString(ctx []interface{}) string {
	if len(ctx) == 0 {
		return ""
	}
	return "\nin " + fmt.Sprint(ctx...)
}

func jsonMap(t *testing.T, v interface{}) (interface{}, []byte) {
	var err error
	j, err := json.Marshal(v)
	if err != nil {
		panic("test: error marshaling value: " + err.Error())
	}

	var m interface{}
	err = json.Unmarshal(j, &m)
	if err != nil {
		panic("test: error unmarshaling value: " + err.Error())
	}

	return m, j
}
