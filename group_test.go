package res

import (
	"testing"
)

// Test parseGroup panics when expected
func TestParseGroup(t *testing.T) {
	tbl := []struct {
		Group   string
		Pattern string
		Panic   bool
	}{
		// Valid groups
		{"", "test", false},
		{"test", "test", false},
		{"test", "test.$foo", false},
		{"test.${foo}", "test.$foo", false},
		{"${foo}", "test.$foo", false},
		{"${foo}.test", "test.$foo", false},
		{"${foo}${bar}", "test.$foo.$bar", false},
		{"${bar}${foo}", "test.$foo.$bar", false},
		{"${foo}.${bar}", "test.$foo.$bar.>", false},
		{"${foo}${foo}", "test.$foo.$bar", false},

		// Invalid groups
		{"$", "test.$foo", true},
		{"${", "test.$foo", true},
		{"${foo", "test.$foo", true},
		{"${}", "test.$foo", true},
		{"${$foo}", "test.$foo", true},
		{"${bar}", "test.$foo", true},
	}

	for _, l := range tbl {
		func() {
			defer func() {
				if r := recover(); r != nil {
					if !l.Panic {
						t.Errorf("expected parseGroup not to panic, but it did:\n\tpanic   : %s\n\tgroup   : %s\n\tpattern : %s", r, l.Group, l.Pattern)
					}
				} else {
					if l.Panic {
						t.Errorf("expected parseGroup to panic, but it didn't\n\tgroup   : %s\n\tpattern : %s", l.Group, l.Pattern)
					}
				}
			}()

			parseGroup(l.Group, l.Pattern)
		}()
	}
}

// Test group toString
func TestGroupToString(t *testing.T) {
	tbl := []struct {
		Group        string
		Pattern      string
		ResourceName string
		Tokens       []string
		Expected     string
	}{
		{"", "test", "test", []string{"test"}, "test"},
		{"test", "test", "test", []string{"test"}, "test"},
		{"foo", "test", "test", []string{"test"}, "foo"},
		{"test", "test.$foo", "test.42", []string{"test", "42"}, "test"},
		{"test.${foo}", "test.$foo", "test.42", []string{"test", "42"}, "test.42"},
		{"bar.${foo}", "test.$foo", "test.42", []string{"test", "42"}, "bar.42"},
		{"${foo}", "test.$foo", "test.42", []string{"test", "42"}, "42"},
		{"${foo}.test", "test.$foo", "test.42", []string{"test", "42"}, "42.test"},
		{"${foo}${bar}", "test.$foo.$bar", "test.42.baz", []string{"test", "42", "baz"}, "42baz"},
		{"${bar}${foo}", "test.$foo.$bar", "test.42.baz", []string{"test", "42", "baz"}, "baz42"},
		{"${foo}.${bar}", "test.$foo.$bar.>", "test.42.baz.extra.all", []string{"test", "42", "baz", "extra", "all"}, "42.baz"},
		{"${foo}${foo}", "test.$foo.$bar", "test.42.baz", []string{"test", "42", "baz"}, "4242"},
		{"${foo}.test.this.${bar}", "test.$foo.$bar", "test.42.baz", []string{"test", "42", "baz"}, "42.test.this.baz"},
	}

	for _, l := range tbl {
		func() {
			gr := parseGroup(l.Group, l.Pattern)
			wid := gr.toString(l.ResourceName, l.Tokens)
			if wid != l.Expected {
				t.Errorf("expected parseGroup(%#v, %#v).toString(%#v, %#v) to return:\n\t%#v\nbut got:\n\t%#v", l.Group, l.Pattern, l.ResourceName, l.Tokens, l.Expected, wid)
			}
		}()
	}
}
