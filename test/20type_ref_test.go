package test

import (
	"encoding/json"
	"fmt"
	"testing"

	res "github.com/jirenius/go-res"
	"github.com/jirenius/go-res/restest"
)

// Test Ref IsValid method
func TestRefIsValid(t *testing.T) {
	tbl := []struct {
		RID   res.Ref
		Valid bool
	}{
		// Valid RID
		{"test", true},
		{"test.model", true},
		{"test.model._hej_", true},
		{"test.model.<strange", true},
		{"test.model.23", true},
		{"test.model.23?", true},
		{"test.model.23?foo=bar", true},
		{"test.model.23?foo=test.bar", true},
		{"test.model.23?foo=*&?", true},
		// Invalid RID
		{"", false},
		{".test", false},
		{"test.", false},
		{".test.model", false},
		{"test..model", false},
		{"test.model.", false},
		{"test\tmodel", false},
		{"test\nmodel", false},
		{"test\rmodel", false},
		{"test model", false},
		{"test\ufffdmodel", false},
		{"täst.model", false},
		{"test.*.model", false},
		{"test.>.model", false},
		{"test.model.>", false},
		{"?foo=test.bar", false},
		{".test.model?foo=test.bar", false},
		{"test..model?foo=test.bar", false},
		{"test.model.?foo=test.bar", false},
		{"test\tmodel?foo=test.bar", false},
		{"test\nmodel?foo=test.bar", false},
		{"test\rmodel?foo=test.bar", false},
		{"test model?foo=test.bar", false},
		{"test\ufffdmodel?foo=test.bar", false},
		{"täst.model?foo=test.bar", false},
		{"test.*.model?foo=test.bar", false},
		{"test.>.model?foo=test.bar", false},
		{"test.model.>?foo=test.bar", false},
	}

	for _, l := range tbl {
		v := l.RID.IsValid()
		if v != l.Valid {
			if l.Valid {
				t.Errorf("expected Ref %#v to be valid, but it wasn't", l.RID)
			} else {
				t.Errorf("expected Ref %#v not to be valid, but it was", l.RID)
			}
		}
	}
}

// Test Ref MarshalJSON method
func TestRefMarshalJSON(t *testing.T) {
	tbl := []struct {
		RID      res.Ref
		Expected []byte
	}{
		// Valid RID
		{"test", []byte(`{"rid":"test"}`)},
		{"test.model", []byte(`{"rid":"test.model"}`)},
		{"test.model._hej_", []byte(`{"rid":"test.model._hej_"}`)},
		{"test.model.<strange", []byte(`{"rid":"test.model.<strange"}`)},
		{"test.model.23", []byte(`{"rid":"test.model.23"}`)},
		{"test.model.23?", []byte(`{"rid":"test.model.23?"}`)},
		{"test.model.23?foo=bar", []byte(`{"rid":"test.model.23?foo=bar"}`)},
		{"test.model.23?foo=test.bar", []byte(`{"rid":"test.model.23?foo=test.bar"}`)},
		{"test.model.23?foo=*&?", []byte(`{"rid":"test.model.23?foo=*&?"}`)},
	}

	for _, l := range tbl {
		out, err := l.RID.MarshalJSON()
		restest.AssertNoError(t, err)
		restest.AssertEqualJSON(t, "Ref.MarshalJSON()", json.RawMessage(out), json.RawMessage(l.Expected))
	}
}

// Test Ref UnmarshalJSON method
func TestRefUnmarshalJSON(t *testing.T) {
	tbl := []struct {
		JSON     []byte
		Expected res.Ref
		Error    bool
	}{
		// Valid RID
		{[]byte(`{"rid":"test"}`), "test", false},
		{[]byte(`{"rid":"test.model"}`), "test.model", false},
		{[]byte(`{"rid":"test.model._hej_"}`), "test.model._hej_", false},
		{[]byte(`{"rid":"test.model.<strange"}`), "test.model.<strange", false},
		{[]byte(`{"rid":"test.model.23"}`), "test.model.23", false},
		{[]byte(`{"rid":"test.model.23?"}`), "test.model.23?", false},
		{[]byte(`{"rid":"test.model.23?foo=bar"}`), "test.model.23?foo=bar", false},
		{[]byte(`{"rid":"test.model.23?foo=test.bar"}`), "test.model.23?foo=test.bar", false},
		{[]byte(`{"rid":"test.model.23?foo=*&?"}`), "test.model.23?foo=*&?", false},
		// Valid but resulting in empty
		{[]byte(`{"rid":""}`), "", false},
		{[]byte(`{"foo":"bar"}`), "", false},
		{[]byte(`{}`), "", false},
		{[]byte(`{"rid": null}`), "", false},
		{[]byte(`null`), "", false},
		// Invalid RID
		{[]byte(`{"rid": 42}`), "", true},
		{[]byte(`{"rid": true}`), "", true},
		{[]byte(`{"rid": ["test"]}`), "", true},
		{[]byte(`{"rid": {"foo":"bar"}}`), "", true},
		{[]byte(`["rid","test"]`), "", true},
		{[]byte(`"test"`), "", true},
		{[]byte(`42`), "", true},
		{[]byte(`true`), "", true},
		{[]byte(`{]`), "", true},
	}

	for i, l := range tbl {
		var ref res.Ref
		err := ref.UnmarshalJSON(l.JSON)
		if l.Error {
			restest.AssertError(t, err, fmt.Sprintf("test #%d", i+1))
		} else {
			restest.AssertNoError(t, err, fmt.Sprintf("test #%d", i+1))
		}
		if ref != l.Expected {
			t.Errorf("expected ref to be:\n%s\nbut got:\n%s", ref, l.Expected)
		}
	}
}
