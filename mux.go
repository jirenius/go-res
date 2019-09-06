package res

// Code inspired, and partly borrowed, from SubList in nats-server
// https://github.com/nats-io/nats-server/blob/master/server/sublist.go

// Common byte variables for wildcards and token separator.
const (
	pmark = '$'
	pwild = '*'
	fwild = '>'
	btsep = '.'
)

const invalidPattern = "res: invalid pattern"

// Mux stores patterns and efficiently retrieves pattern handlers.
type Mux struct {
	pattern string
	root    *node
	parent  *Mux
}

// A registered handler
type regHandler struct {
	Handler
	group group
}

// A node represents one part of the path, and has pointers
// to the next nodes, including wildcards.
// Only one instance of handlers may exist per node.
type node struct {
	hs      *regHandler // Handlers on this node
	params  []pathParam // path parameters for the handlers
	nodes   map[string]*node
	param   *node
	wild    *node // Wild card node
	mounted bool
}

// A pathParam represent a parameter part of the resource name.
type pathParam struct {
	name string // name of the parameter
	idx  int    // token index of the parameter
}

// Matchin handlers instance to a resource name
type nodeMatch struct {
	hs       *regHandler
	params   map[string]string
	mountIdx int
}

// NewMux returns a new root Mux starting with given pattern.
func NewMux(pattern string) *Mux {
	return &Mux{
		pattern: pattern,
		root:    &node{},
	}
}

// Pattern returns the pattern that prefix all resources,
// not including the pattern of any parent Mux.
func (m *Mux) Pattern() string {
	return m.pattern
}

// FullPattern returns the pattern that prefix all resources,
// including the pattern of any parent Mux.
func (m *Mux) FullPattern() string {
	if m.parent == nil {
		return m.pattern
	}
	return mergePattern(m.parent.pattern, m.pattern)
}

// Handle registers the handler functions for the given resource subpattern.
//
// A pattern may contain placeholders that acts as wildcards, and will be
// parsed and stored in the request.PathParams map.
// A placeholder is a resource name part starting with a dollar ($) character:
//  s.Handle("user.$id", handler) // Will match "user.10", "user.foo", etc.
// An anonymous placeholder is a resource name part using an asterisk (*) character:
//  s.Handle("user.*", handler)   // Will match "user.10", "user.foo", etc.
// A full wildcard can be used as last part using a greather than (>) character:
//  s.Handle("data.>", handler)   // Will match "data.foo", "data.foo.bar", etc.
//
// If the pattern is already registered, or if there are conflicts among
// the handlers, Handle panics.
func (m *Mux) Handle(subpattern string, hf ...Option) {
	var h Handler
	for _, f := range hf {
		f.SetOption(&h)
	}
	m.AddHandler(subpattern, h)
}

// AddHandler register a handler for the given resource subpattern.
// The pattern used is the same as described for Handle.
func (m *Mux) AddHandler(subpattern string, hs Handler) {
	h := regHandler{
		Handler: hs,
		group:   parseGroup(hs.Group, subpattern),
	}

	m.add(subpattern, &h)
}

// Mount attaches another Mux at a given pattern.
// When mounting, any pattern set on the sub Mux will be suffixed to the subpattern.
func (m *Mux) Mount(subpattern string, sub *Mux) {
	if sub.parent != nil {
		panic("res: already mounted")
	}
	spattern := mergePattern(subpattern, sub.pattern)
	if spattern == "" {
		panic("res: attempting to mount to root")
	}
	n, _ := m.fetch(spattern, sub.root)
	if n != sub.root {
		panic("res: attempting to mount to existing pattern: " + mergePattern(m.pattern, spattern))
	}
	sub.pattern = spattern
	sub.root.mounted = true
	sub.parent = m
}

// Route create a new Mux and mounts it to the given subpattern.
func (m *Mux) Route(subpattern string, fn func(m *Mux)) *Mux {
	sub := NewMux("")
	if fn != nil {
		fn(sub)
	}
	m.Mount(subpattern, sub)
	return sub
}

// add inserts new handlers for a given subpattern.
// An invalid pattern, or a pattern already registered will cause panic.
func (m *Mux) add(subpattern string, hs *regHandler) {
	n, params := m.fetch(subpattern, nil)

	if n.hs != nil {
		panic("res: registration already done for pattern " + mergePattern(m.pattern, subpattern))
	}
	n.params = params
	n.hs = hs
}

// fetch get the node for a given subpattern (not including Mux path).
// An invalid pattern will cause panic.
func (m *Mux) fetch(subpattern string, mount *node) (*node, []pathParam) {
	tokens := splitPattern(subpattern)

	var params []pathParam

	l := m.root
	var n *node
	var doMount bool
	var mountIdx int

	for i, t := range tokens {
		if mount != nil && i == len(tokens)-1 {
			doMount = true
		}
		if l.mounted {
			mountIdx = i
		}

		lt := len(t)
		if lt == 0 {
			panic(invalidPattern)
		}

		if t[0] == pmark || t[0] == pwild {
			if lt == 1 {
				panic(invalidPattern)
			}
			if t[0] == pmark {
				name := t[1:]
				// Validate subpattern is unique
				for _, p := range params {
					if p.name == name {
						panic("res: placeholder " + t + " found multiple times in pattern: " + mergePattern(m.pattern, subpattern))
					}
				}
				params = append(params, pathParam{name: name, idx: i - mountIdx})
			}
			if l.param == nil {
				if doMount {
					l.param = mount
				} else {
					l.param = &node{}
				}
			}
			n = l.param
		} else if t[0] == fwild {
			// Validate the full wildcard is last
			if lt > 1 || i < len(tokens)-1 {
				panic(invalidPattern)
			}
			if l.wild == nil {
				if doMount {
					panic("res: attempting to mount on full wildcard pattern: " + mergePattern(m.pattern, subpattern))
				}
				l.wild = &node{}
			}
			n = l.wild
		} else {
			if l.nodes == nil {
				l.nodes = make(map[string]*node)
				if doMount {
					n = mount
				} else {
					n = &node{}
				}
				l.nodes[t] = n
			} else {
				n = l.nodes[t]
				if n == nil {
					if doMount {
						n = mount
					} else {
						n = &node{}
					}
					l.nodes[t] = n
				}
			}
		}

		l = n
	}

	return l, params
}

// GetHandler parses the resource name and gets the registered handler,
// path params, and group ID.
// Returns the matching handler, or an error if not found.
func (m *Mux) GetHandler(rname string) (Handler, map[string]string, string, error) {
	var tokens []string
	subrname := rname
	pl := len(m.pattern)
	if pl > 0 {
		rl := len(rname)
		if pl == rl {
			if m.pattern != rname {
				return Handler{}, nil, "", errHandlerNotFound
			}
			subrname = ""
		} else {
			if pl > rl || (rname[0:pl] != m.pattern) || rname[pl] != '.' {
				return Handler{}, nil, "", errHandlerNotFound
			}
			subrname = rname[pl+1:]
		}
	}

	if len(subrname) == 0 {
		if m.root.hs == nil {
			return Handler{}, nil, "", errHandlerNotFound
		}

		return m.root.hs.Handler, nil, m.root.hs.group.toString(rname, nil), nil
	}

	tokens = make([]string, 0, 32)
	start := 0
	for i := 0; i < len(subrname); i++ {
		if subrname[i] == btsep {
			tokens = append(tokens, subrname[start:i])
			start = i + 1
		}
	}
	tokens = append(tokens, subrname[start:])

	var nm nodeMatch
	matchNode(m.root, tokens, 0, 0, &nm)
	if nm.hs == nil {
		return Handler{}, nil, "", errHandlerNotFound
	}

	return nm.hs.Handler, nm.params, nm.hs.group.toString(rname, tokens[nm.mountIdx:]), nil
}

func matchNode(l *node, toks []string, i int, mi int, nm *nodeMatch) bool {
	n := l.nodes[toks[i]]
	if l.mounted {
		mi = i
	}
	i++
	c := 2
	for c > 0 {
		// Does the node exist
		if n != nil {
			// Check if it is the last token
			if len(toks) == i {
				// Check if this node has handlers
				if n.hs != nil {
					nm.hs = n.hs
					nm.mountIdx = mi
					// Check if we have path parameters for the handlers
					if len(n.params) > 0 {
						// Create a map with path parameter values
						nm.params = make(map[string]string, len(n.params))
						for _, pp := range n.params {
							nm.params[pp.name] = toks[pp.idx+mi]
						}
					}
					return true
				}
			} else {
				// Match against next node
				if matchNode(n, toks, i, mi, nm) {
					return true
				}
			}
		}

		// To avoid repeating code above, set node to test to l.param
		// and run it all again.
		n = l.param
		c--
	}

	// Check full wild card
	if l.wild != nil {
		n = l.wild
		nm.hs = n.hs
		nm.mountIdx = mi
		if len(n.params) > 0 {
			// Create a map with path parameter values
			nm.params = make(map[string]string, len(n.params))
			for _, pp := range n.params {
				nm.params[pp.name] = toks[pp.idx+mi]
			}
		}
		return true
	}

	return false
}

func splitPattern(p string) []string {
	if len(p) == 0 {
		return nil
	}
	tokens := make([]string, 0, 32)
	start := 0
	for i := 0; i < len(p); i++ {
		if p[i] == btsep {
			tokens = append(tokens, p[start:i])
			start = i + 1
		}
	}
	tokens = append(tokens, p[start:])
	return tokens
}

func mergePattern(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	return a + "." + b
}

func (m *Mux) hasResources() bool {
	return hasPattern(m.root, func(n *node) bool {
		if n.hs == nil {
			return false
		}
		hs := n.hs
		return hs.Get != nil || len(hs.Call) > 0 || len(hs.Auth) > 0 || hs.New != nil
	})
}

func (m *Mux) hasAccess() bool {
	return hasPattern(m.root, func(n *node) bool {
		if n.hs == nil {
			return false
		}
		return n.hs.Access != nil
	})
}

func hasPattern(n *node, test func(n *node) bool) bool {
	if n.wild != nil && test(n.wild) {
		return true
	}

	if n.param != nil && (test(n.param) || hasPattern(n.param, test)) {
		return true
	}

	for _, nn := range n.nodes {
		if test(nn) || hasPattern(nn, test) {
			return true
		}
	}

	return false
}
