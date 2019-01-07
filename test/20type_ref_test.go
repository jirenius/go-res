package test

import (
	"testing"

	res "github.com/jirenius/go-res"
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
		{".test.model", false},
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
		{".test.model?foo=test.bar", false},
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
