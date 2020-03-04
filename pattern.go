package res

// Pattern is a resource pattern that may contain wildcards and tags.
// 	Pattern("example.resource.>") // Full wild card (>) matches anything that follows
// 	Pattern("example.item.*")     // Wild card (*) matches a single part
// 	Pattern("example.model.$id")  // Tag (starting with $) matches a single part
type Pattern string

// IsValid returns true if the pattern is valid, otherwise false.
func (p Pattern) IsValid() bool {
	if len(p) == 0 {
		return true
	}
	start := true
	alone := false
	emptytag := false
	for i, c := range p {
		if c == '.' {
			if start || emptytag {
				return false
			}
			alone = false
			start = true
		} else {
			if alone || c < 33 || c > 126 || c == '?' {
				return false
			}
			switch c {
			case '>':
				if !start || i < len(p)-1 {
					return false
				}
			case '*':
				if !start {
					return false
				}
				alone = true
			case '$':
				if start {
					emptytag = true
				}
			default:
				emptytag = false
			}
			start = false
		}
	}

	return !(start || emptytag)
}

// Matches tests if the resource name, s, matches the pattern.
//
// The resource name might in itself contain wild cards and tags.
//
// Behavior is undefined for an invalid pattern or an invalid resource name.
func (p Pattern) Matches(s string) bool {
	pi := 0
	si := 0
	pl := len(p)
	sl := len(s)
	for pi < pl {
		if si == sl {
			return false
		}
		c := p[pi]
		pi++
		switch c {
		case '$':
			fallthrough
		case '*':
			for pi < pl && p[pi] != '.' {
				pi++
			}
			if s[si] == '>' {
				return false
			}
			for si < sl && s[si] != '.' {
				si++
			}
		case '>':
			return pi == pl
		default:
			if c != s[si] {
				return false
			}
			si++
		}
	}
	return si == sl
}

// ReplaceTags searches for tags and replaces them with
// the map value for the key matching the tag (without $).
//
// Behavior is undefined for an invalid pattern.
func (p Pattern) ReplaceTags(m map[string]string) Pattern {
	// Quick exit on empty map
	if len(m) == 0 {
		return p
	}
	return p.replace(func(t string) (string, bool) {
		v, ok := m[t]
		return v, ok
	})
}

// ReplaceTag searches for a given tag (without $) and replaces
// it with the value.
//
// Behavior is undefined for an invalid pattern.
func (p Pattern) ReplaceTag(tag string, value string) Pattern {
	return p.replace(func(t string) (string, bool) {
		if tag == t {
			return value, true
		}
		return "", false
	})
}

// replace replaces tags with a value.
func (p Pattern) replace(replacer func(tag string) (string, bool)) Pattern {
	type rep struct {
		o int    // tag offset (including $)
		e int    // tag end
		v string // replace value
	}
	var rs []rep
	pi := 0
	pl := len(p)
	start := true
	var o int
	for pi < pl {
		c := p[pi]
		pi++
		switch c {
		case '$':
			if start {
				// Temporarily store tag start offset
				o = pi
				// Find end of tag
				for pi < pl && p[pi] != '.' {
					pi++
				}
				// Get the replacement value from the replacer callback.
				if v, ok := replacer(string(p[o:pi])); ok {
					rs = append(rs, rep{o: o - 1, e: pi, v: v})
				}
			}
		case '.':
			start = true
		default:
			start = false
		}
	}
	// Quick exit on no replacements
	if len(rs) == 0 {
		return p
	}
	// Calculate length, nl, of resulting string
	nl := pl
	for _, r := range rs {
		nl += len(r.v) - r.e + r.o
	}
	// Create our result bytes
	result := make([]byte, nl)
	o = 0  // Reuse as result offset
	pi = 0 // Reuse as pattern index position
	for _, r := range rs {
		if r.o > 0 {
			seg := p[pi:r.o]
			copy(result[o:], seg)
			o += len(seg)
		}
		copy(result[o:], r.v)
		o += len(r.v)
		pi = r.e
	}
	if pi < pl {
		copy(result[o:], p[pi:])
	}
	return Pattern(result)
}

// Values extracts the tag values from a resource name, s, matching the pattern.
//
// The returned bool flag is true if s matched the pattern, otherwise false with a nil map.
//
// Behavior is undefined for an invalid pattern or an invalid resource name.
func (p Pattern) Values(s string) (map[string]string, bool) {
	pi := 0
	si := 0
	pl := len(p)
	sl := len(s)
	var m map[string]string
	for pi < pl {
		if si == sl {
			return nil, false
		}
		c := p[pi]
		pi++
		switch c {
		case '$':
			po := pi
			for pi < pl && p[pi] != '.' {
				pi++
			}
			so := si
			for si < sl && s[si] != '.' {
				si++
			}
			if m == nil {
				m = make(map[string]string)
			}
			m[string(p[po:pi])] = s[so:si]
		case '*':
			for pi < pl && p[pi] != '.' {
				pi++
			}
			for si < sl && s[si] != '.' {
				si++
			}
		case '>':
			if pi == pl {
				return m, true
			}
			return nil, false
		default:
			for {
				if c != s[si] {
					return nil, false
				}
				si++
				if c == '.' || pi == pl {
					break
				}
				c = p[pi]
				pi++
				if si == sl {
					return nil, false
				}
			}
		}
	}
	if si != sl {
		return nil, false
	}
	return m, true
}

// IndexWildcard returns the index of the first instance of a wild card (*, >, or $tag)
// in pattern, or -1 if no wildcard is present.
//
// Behavior is undefined for an invalid pattern.
func (p Pattern) IndexWildcard() int {
	start := true
	for i, c := range p {
		if c == '.' {
			start = true
		} else {
			if start && ((c == '>' && i == len(p)-1) ||
				c == '*' ||
				c == '$') {
				return i
			}
			start = false
		}
	}
	return -1
}
