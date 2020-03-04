package res

import (
	"reflect"
	"testing"
)

func TestPatternIsValid_ValidPattern_ReturnsTrue(t *testing.T) {
	tbl := []struct {
		Pattern string
	}{
		{""},
		{"test"},
		{"test.model"},
		{"test.model.foo"},
		{"test$.model"},

		{">"},
		{"test.>"},
		{"test.model.>"},

		{"*"},
		{"test.*"},
		{"*.model"},
		{"test.*.foo"},
		{"test.model.*"},
		{"*.model.foo"},
		{"test.*.*"},

		{"$foo"},
		{"test.$foo"},
		{"$foo.model"},
		{"test.$foo.foo"},
		{"test.model.$foo"},
		{"test.$foo.$bar"},

		{"test.*.>"},
		{"test.$foo.>"},
		{"*.$foo.>"},
	}

	for _, r := range tbl {
		if !Pattern(r.Pattern).IsValid() {
			t.Errorf("Pattern(%#v).IsValid() did not return true", r.Pattern)
		}
	}
}

func TestPatternIsValid_InvalidPattern_ReturnsFalse(t *testing.T) {
	tbl := []struct {
		Pattern string
	}{
		{"."},
		{".test"},
		{"test."},
		{"test..foo"},

		{"*test"},
		{"test*"},
		{"test.*foo"},
		{"test.foo*"},

		{">test"},
		{"test>"},
		{"test.>foo"},
		{"test.foo>"},
		{"test.>.foo"},

		{"$"},
		{"$.test"},
		{"test.$.foo"},
		{"test.foo.$"},

		{"test.$foo>"},
		{"test.$foo*"},

		{"test.foo?"},
		{"test. .foo"},
		{"test.\x127.foo"},
		{"test.räv"},
	}

	for _, r := range tbl {
		if Pattern(r.Pattern).IsValid() {
			t.Errorf("Pattern(%#v).IsValid() did not return false", r.Pattern)
		}
	}
}

func TestPatternMatches_MatchingPattern_ReturnsTrue(t *testing.T) {
	tbl := []struct {
		Pattern      string
		ResourceName string
	}{
		{"", ""},
		{"test", "test"},
		{"test.model", "test.model"},
		{"test.model.foo", "test.model.foo"},
		{"test$.model", "test$.model"},

		{">", "test.model.foo"},
		{"test.>", "test.model.foo"},
		{"test.model.>", "test.model.foo"},

		{"*", "test"},
		{"test.*", "test.model"},
		{"*.model", "test.model"},
		{"test.*.foo", "test.model.foo"},
		{"test.model.*", "test.model.foo"},
		{"*.model.foo", "test.model.foo"},
		{"test.*.*", "test.model.foo"},

		{"$foo", "test"},
		{"test.$foo", "test.model"},
		{"$foo.model", "test.model"},
		{"test.$foo.foo", "test.model.foo"},
		{"test.model.$foo", "test.model.foo"},
		{"$foo.model.foo", "test.model.foo"},
		{"test.$foo.$bar", "test.model.foo"},

		{"test.*.>", "test.model.foo"},
		{"test.$foo.>", "test.model.foo.bar"},
		{"*.$foo.>", "test.model.foo.bar"},
	}

	for _, r := range tbl {
		if !Pattern(r.Pattern).Matches(r.ResourceName) {
			t.Errorf("Pattern(%#v).Matches(%#v) did not return true", r.Pattern, r.ResourceName)
		}
	}
}

func TestPatternMatches_NonMatchingPattern_ReturnsFalse(t *testing.T) {
	tbl := []struct {
		Pattern      string
		ResourceName string
	}{
		{"", "test"},
		{"test", "test.model"},
		{"test.model", "test.mode"},
		{"test.model.foo", "test.model"},
		{"test.model.foo", "test.mode.foo"},

		{">", ""},
		{"test.>", "test"},
		{"test.model.>", "test.model"},

		{"*", "test.model"},
		{"test.*", "test.model.foo"},
		{"*.model", "test"},
		{"test.*.foo", "test.model"},
		{"test.model.*", "test.model"},
		{"*.model.foo", "test.model"},
		{"test.*.*", "test.model"},

		{"$foo", "test.model"},
		{"test.$foo", "test.model.foo"},
		{"$foo.model", "test"},
		{"test.$foo.foo", "test.model"},
		{"test.model.$foo", "test.model"},
		{"$foo.model.foo", "test.model"},
		{"test.$foo.$bar", "test.model"},

		{"test.*.>", "test.model"},
		{"test.$foo.>", "test.model"},
		{"*.$foo.>", "test.model"},
	}

	for _, r := range tbl {
		if Pattern(r.Pattern).Matches(r.ResourceName) {
			t.Errorf("Pattern(%#v).Matches(%#v) did not return false", r.Pattern, r.ResourceName)
		}
	}
}

func TestPatternIndexWildcard_ValidPattern_ReturnsIndex(t *testing.T) {
	tbl := []struct {
		Pattern string
		Index   int
	}{
		{"test", -1},
		{"test.model", -1},
		{"test.model.foo", -1},
		{"test$.model", -1},

		{">", 0},
		{"test.>", 5},
		{"test.model.>", 11},

		{"*", 0},
		{"test.*", 5},
		{"*.model", 0},
		{"test.*.foo", 5},
		{"test.model.*", 11},
		{"*.model.foo", 0},
		{"test.*.*", 5},

		{"$foo", 0},
		{"test.$foo", 5},
		{"$foo.model", 0},
		{"test.$foo.foo", 5},
		{"test.model.$foo", 11},
		{"test.$foo.$bar", 5},

		{"test.*.>", 5},
		{"test.$foo.>", 5},
		{"*.$foo.>", 0},
	}

	for _, r := range tbl {
		idx := Pattern(r.Pattern).IndexWildcard()
		if idx != r.Index {
			t.Errorf("Expected Pattern(%#v).IndexWildcard() to return %d, but it returned %d", r.Pattern, r.Index, idx)
		}
	}
}

func TestPatternReplaceTag_ValidPattern_ReturnsExpectedPattern(t *testing.T) {
	tbl := []struct {
		Pattern  Pattern
		Tag      string
		Value    string
		Expected Pattern
	}{
		// No replacements
		{"test", "id", "foo", "test"},
		{"test.model", "model", "bar", "test.model"},
		{"test.model.foo", "", "bar", "test.model.foo"},
		{"$test", "id", "bar", "$test"},
		{"test.$model", "id", "bar", "test.$model"},
		{"$test.model.foo", "id", "bar", "$test.model.foo"},
		{"test.$model.foo", "id", "bar", "test.$model.foo"},
		{"test.model.$foo", "id", "bar", "test.model.$foo"},
		{"$test.$model.$foo", "id", "bar", "$test.$model.$foo"},
		{"test.$foobar", "foo", "baz", "test.$foobar"},
		// Single replacement
		{"$test", "test", "bar", "bar"},
		{"test.$model", "model", "bar", "test.bar"},
		{"$test.model.foo", "test", "bar", "bar.model.foo"},
		{"test.$model.foo", "model", "bar", "test.bar.foo"},
		{"test.model.$foo", "foo", "bar", "test.model.bar"},
		{"$test.$model.$foo", "test", "bar", "bar.$model.$foo"},
		{"$test.$model.$foo", "model", "bar", "$test.bar.$foo"},
		{"$test.$model.$foo", "foo", "bar", "$test.$model.bar"},
		// Multiple replacements
		{"$foo.$foo.test", "foo", "bar", "bar.bar.test"},
		{"$foo.test.$foo", "foo", "bar", "bar.test.bar"},
		{"test.$foo.$foo", "foo", "bar", "test.bar.bar"},
		{"$foo.$foo.$foo", "foo", "bar", "bar.bar.bar"},
		// No replacement with wildcards
		{"*", "model", "bar", "*"},
		{">", "model", "bar", ">"},
		{"test.*", "model", "bar", "test.*"},
		{"test.*.foo", "model", "bar", "test.*.foo"},
		{"test.model.*", "", "bar", "test.model.*"},
		{"test.model.>", "model", "bar", "test.model.>"},
		// Replacement with wildcards
		{"test.$model.*", "model", "bar", "test.bar.*"},
		{"test.*.$foo", "foo", "bar", "test.*.bar"},
		{"test.$model.>", "model", "bar", "test.bar.>"},
	}

	for _, r := range tbl {
		p := r.Pattern.ReplaceTag(r.Tag, r.Value)
		if p != r.Expected {
			t.Errorf("Expected Pattern(%#v).ReplaceTag(%#v, %#v) to return %#v, but it returned %#v", r.Pattern, r.Tag, r.Value, r.Expected, p)
		}
	}
}

func TestPatternReplaceTags_ValidPattern_ReturnsExpectedPattern(t *testing.T) {
	tbl := []struct {
		Pattern  Pattern
		Tags     map[string]string
		Expected Pattern
	}{
		// No replacements
		{"test", map[string]string{"id": "foo"}, "test"},
		{"test.model", map[string]string{"model": "bar"}, "test.model"},
		{"test.model.foo", nil, "test.model.foo"},
		{"test.model.foo", map[string]string{"": "bar"}, "test.model.foo"},
		{"$test", map[string]string{"id": "bar"}, "$test"},
		{"test.$model", map[string]string{"id": "bar"}, "test.$model"},
		{"$test.model.foo", nil, "$test.model.foo"},
		{"$test.model.foo", map[string]string{"id": "bar"}, "$test.model.foo"},
		{"test.$model.foo", map[string]string{"id": "bar"}, "test.$model.foo"},
		{"test.model.$foo", map[string]string{"id": "bar"}, "test.model.$foo"},
		{"$test.$model.$foo", map[string]string{"id": "bar"}, "$test.$model.$foo"},
		{"test.$foobar", map[string]string{"foo": "baz"}, "test.$foobar"},
		// Single replacement
		{"$test", map[string]string{"test": "bar"}, "bar"},
		{"test.$model", map[string]string{"model": "bar"}, "test.bar"},
		{"$test.model.foo", map[string]string{"test": "bar"}, "bar.model.foo"},
		{"test.$model.foo", map[string]string{"model": "bar"}, "test.bar.foo"},
		{"test.model.$foo", map[string]string{"foo": "bar"}, "test.model.bar"},
		{"$test.$model.$foo", map[string]string{"test": "bar"}, "bar.$model.$foo"},
		{"$test.$model.$foo", map[string]string{"model": "bar"}, "$test.bar.$foo"},
		{"$test.$model.$foo", map[string]string{"foo": "bar"}, "$test.$model.bar"},
		// Multiple replacements
		{"$test.$model.$foo", map[string]string{"test": "bar", "model": "baz"}, "bar.baz.$foo"},
		{"$test.$model.$foo", map[string]string{"test": "bar", "foo": "baz"}, "bar.$model.baz"},
		{"$test.$model.$foo", map[string]string{"model": "bar", "foo": "baz"}, "$test.bar.baz"},
		{"$test.$model.$foo", map[string]string{"test": "zoo", "model": "bar", "foo": "baz", "unused": "no"}, "zoo.bar.baz"},
		// No replacement with wildcards
		{"*", map[string]string{"model": "bar"}, "*"},
		{">", map[string]string{"model": "bar"}, ">"},
		{"test.*", map[string]string{"model": "bar"}, "test.*"},
		{"test.*.foo", map[string]string{"model": "bar"}, "test.*.foo"},
		{"test.model.*", map[string]string{"foo": "bar"}, "test.model.*"},
		{"test.model.>", map[string]string{"model": "bar"}, "test.model.>"},
		// Replacement with wildcards
		{"test.$model.*", map[string]string{"model": "bar"}, "test.bar.*"},
		{"test.*.$foo", map[string]string{"foo": "bar"}, "test.*.bar"},
		{"test.$model.>", map[string]string{"model": "bar"}, "test.bar.>"},
	}

	for _, r := range tbl {
		p := r.Pattern.ReplaceTags(r.Tags)
		if p != r.Expected {
			t.Errorf("Expected Pattern(%#v).ReplaceTags(%#v) to return %#v, but it returned %#v", r.Pattern, r.Tags, r.Expected, p)
		}
	}
}

func TestPatternValues_MatchingPattern_ReturnsValues(t *testing.T) {
	tbl := []struct {
		Pattern      Pattern
		ResourceName string
		Values       map[string]string
	}{
		{"", "", nil},
		{"test", "test", nil},
		{"test.model", "test.model", nil},
		{"test.model.foo", "test.model.foo", nil},
		{"test$.model", "test$.model", nil},

		{">", "test.model.foo", nil},
		{"test.>", "test.model.foo", nil},
		{"test.model.>", "test.model.foo", nil},

		{"*", "test", nil},
		{"test.*", "test.model", nil},
		{"*.model", "test.model", nil},
		{"test.*.foo", "test.model.foo", nil},
		{"test.model.*", "test.model.foo", nil},
		{"*.model.foo", "test.model.foo", nil},
		{"test.*.*", "test.model.foo", nil},

		{"$foo", "test", map[string]string{"foo": "test"}},
		{"test.$foo", "test.model", map[string]string{"foo": "model"}},
		{"$foo.model", "test.model", map[string]string{"foo": "test"}},
		{"test.$foo.foo", "test.model.foo", map[string]string{"foo": "model"}},
		{"test.model.$foo", "test.model.foo", map[string]string{"foo": "foo"}},
		{"$foo.model.foo", "test.model.foo", map[string]string{"foo": "test"}},
		{"test.$foo.$bar", "test.model.foo", map[string]string{"foo": "model", "bar": "foo"}},

		{"test.*.>", "test.model.foo", nil},
		{"test.$foo.>", "test.model.foo.bar", map[string]string{"foo": "model"}},
		{"*.$foo.>", "test.model.foo.bar", map[string]string{"foo": "model"}},
	}

	for _, r := range tbl {
		v, ok := r.Pattern.Values(r.ResourceName)
		if !ok {
			t.Errorf("Pattern(%#v).Values(%#v) did not return true", r.Pattern, r.ResourceName)
		}
		if !reflect.DeepEqual(r.Values, v) {
			t.Errorf("Expected Pattern(%#v).Values(%#v) to return:\n\t%+v\nbut it returned:\n\t%+v", r.Pattern, r.ResourceName, r.Values, v)
		}
	}
}

func TestPatternValues_NonMatchingPattern_ReturnsFalse(t *testing.T) {
	tbl := []struct {
		Pattern      Pattern
		ResourceName string
	}{
		{"", "test"},
		{"test", "test.model"},
		{"test.model", "test.mode"},
		{"test.model.foo", "test.model"},
		{"test.model.foo", "test.mode.foo"},

		{">", ""},
		{"test.>", "test"},
		{"test.model.>", "test.model"},

		{"*", "test.model"},
		{"test.*", "test.model.foo"},
		{"*.model", "test"},
		{"test.*.foo", "test.model"},
		{"test.model.*", "test.model"},
		{"*.model.foo", "test.model"},
		{"test.*.*", "test.model"},

		{"$foo", "test.model"},
		{"test.$foo", "test.model.foo"},
		{"$foo.model", "test"},
		{"test.$foo.foo", "test.model"},
		{"test.model.$foo", "test.model"},
		{"$foo.model.foo", "test.model"},
		{"test.$foo.$bar", "test.model"},

		{"test.*.>", "test.model"},
		{"test.$foo.>", "test.model"},
		{"*.$foo.>", "test.model"},
	}

	for _, r := range tbl {
		if _, ok := r.Pattern.Values(r.ResourceName); ok {
			t.Errorf("Pattern(%#v).Values(%#v) did not return false", r.Pattern, r.ResourceName)
		}
	}
}
