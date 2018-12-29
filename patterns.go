package res

// Code inspired, and partly borrowed, from SubList in gnatsd
// https://github.com/nats-io/gnatsd/blob/master/server/sublist.go

// Common byte variables for wildcards and token separator.
const (
	pmark = '$'
	btsep = '.'
)

const invalidPattern = "res: invalid pattern"

// patterns stores patterns and efficiently retrieves pattern handlers.
type patterns struct {
	root *node
}

// A node represents one part of the path, and has pointers
// to the next nodes, including wildcards.
// Only one instance of handlers may exist per node.
type node struct {
	hs     *regHandler // Handlers on this node
	params []pathParam // path parameters for the handlers
	nodes  map[string]*node
	param  *node
}

// A pathParam represent a parameter part of the resource name.
type pathParam struct {
	name string // name of the parameter
	idx  int    // token index of the parameter
}

// Matchin handlers instance to a resource name
type nodeMatch struct {
	hs     *regHandler
	params map[string]string
}

// add inserts new handlers to the pattern store.
// An invalid pattern, or a pattern already registered will make add panic.
func (ls *patterns) add(pattern string, hs *regHandler) {
	var tokens []string
	if len(pattern) > 0 {
		tokens = make([]string, 0, 32)
		start := 0
		for i := 0; i < len(pattern); i++ {
			if pattern[i] == btsep {
				tokens = append(tokens, pattern[start:i])
				start = i + 1
			}
		}
		tokens = append(tokens, pattern[start:])
	}
	var params []pathParam

	l := ls.root
	var n *node

	for i, t := range tokens {
		lt := len(t)
		if lt == 0 {
			panic(invalidPattern)
		}

		if t[0] == pmark {
			if lt == 1 {
				panic(invalidPattern)
			}
			params = append(params, pathParam{name: t[1:], idx: i})
			if l.param == nil {
				l.param = &node{}
			}
			n = l.param
		} else {
			if l.nodes == nil {
				l.nodes = make(map[string]*node)
				n = &node{}
				l.nodes[t] = n
			} else {
				n = l.nodes[t]
				if n == nil {
					n = &node{}
					l.nodes[t] = n
				}
			}
		}

		l = n
	}

	if l.hs != nil {
		panic("res: registration already done for pattern " + pattern)
	}

	l.params = params
	l.hs = hs
}

// get parses the resource name and gets the registered handlers and
// any path params.
// Returns nil, nil if there is no match
func (ls *patterns) get(rname string) (*regHandler, map[string]string) {
	var tokens []string
	if len(rname) > 0 {
		tokens = make([]string, 0, 32)
		start := 0
		for i := 0; i < len(rname); i++ {
			if rname[i] == btsep {
				tokens = append(tokens, rname[start:i])
				start = i + 1
			}
		}
		tokens = append(tokens, rname[start:])
	}

	var m nodeMatch
	matchNode(ls.root, tokens, 0, &m)

	return m.hs, m.params
}

func matchNode(l *node, toks []string, i int, m *nodeMatch) bool {
	t := toks[i]
	i++
	c := 2
	n := l.nodes[t]
	for c > 0 {
		// Does the node exist
		if n != nil {
			// Check if it is the last token
			if len(toks) == i {
				// Check if this node has handlers
				if n.hs != nil {
					m.hs = n.hs
					// Check if we have path parameters for the handlers
					if len(n.params) > 0 {
						// Create a map with path parameter values
						m.params = make(map[string]string, len(n.params))
						for _, pp := range n.params {
							m.params[pp.name] = toks[pp.idx]
						}
					}
					return true
				}
			} else {
				// Match against next node
				if matchNode(n, toks, i, m) {
					return true
				}
			}
		}

		// To avoid repeating code above, set node to test to l.param
		// and run it all again.
		n = l.param
		c--
	}

	return false
}
