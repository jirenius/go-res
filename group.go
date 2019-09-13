package res

import (
	"fmt"
	"strings"
)

type group []gpart

type gpart struct {
	str string
	idx int
}

// parseGroup takes a group name and parses it for ${tag} sequences,
// verifying the tags exists as parameter tags in the pattern as well.
// Panics if an error is encountered.
func parseGroup(g, pattern string) group {
	if g == "" {
		return nil
	}

	tokens := splitPattern(pattern)

	var gr group
	var c byte
	l := len(g)
	i := 0
	start := 0

StateDefault:
	if i == l {
		if i > start {
			gr = append(gr, gpart{str: g[start:i]})
		}
		return gr
	}
	if g[i] == '$' {
		if i > start {
			gr = append(gr, gpart{str: g[start:i]})
		}
		i++
		if i == l {
			goto UnexpectedEnd
		}
		if g[i] != '{' {
			panic(fmt.Sprintf("expected character \"{\" at pos %d", i))
		}
		i++
		start = i
		goto StateTag
	}
	i++
	goto StateDefault

StateTag:
	if i == l {
		goto UnexpectedEnd
	}
	c = g[i]
	if c == '}' {
		if i == start {
			panic(fmt.Sprintf("empty group tag at pos %d", i))
		}
		tag := "$" + g[start:i]
		for j, t := range tokens {
			if t == tag {
				gr = append(gr, gpart{idx: j})
				goto TokenFound
			}
		}
		panic(fmt.Sprintf("group tag %s not found in pattern", tag))
	TokenFound:
		i++
		start = i
		goto StateDefault
	}
	if (c < 'A' || c > 'Z') && (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' && c != '-' {
		panic(fmt.Sprintf("non alpha-numeric (a-z0-9_-) character in group tag at pos %d", i))
	}
	i++
	goto StateTag

UnexpectedEnd:
	panic("unexpected end of group tag")
}

func (g group) toString(rname string, tokens []string) string {
	l := len(g)
	if l == 0 {
		return rname
	}
	if l == 1 && g[0].str != "" {
		return g[0].str
	}

	var b strings.Builder
	for _, gp := range g {
		if gp.str == "" {
			b.WriteString(tokens[gp.idx])
		} else {
			b.WriteString(gp.str)
		}
	}

	return b.String()
}
