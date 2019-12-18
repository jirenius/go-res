package res

import "testing"

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