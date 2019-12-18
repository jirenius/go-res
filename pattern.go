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
