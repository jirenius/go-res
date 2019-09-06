package test

import (
	"encoding/json"
	"testing"

	res "github.com/jirenius/go-res"
)

// Test Route adds to the path of the the parent
func TestRoute(t *testing.T) {
	model := resource["test.model"]

	runTest(t, func(s *Session) {
		s.Route("foo", func(m *res.Mux) {
			m.Handle("model",
				res.GetModel(func(r res.ModelRequest) {
					r.Model(json.RawMessage(model))
				}),
			)
		})
	}, func(s *Session) {
		// Test getting the model
		inb := s.Request("get.test.foo.model", newRequest())
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":{"model":`+model+`}}`))
	})
}

// Test Mount Mux to service
func TestMount(t *testing.T) {
	model := resource["test.model"]

	runTest(t, func(s *Session) {
		m := res.NewMux("")
		m.Handle("model",
			res.GetModel(func(r res.ModelRequest) {
				r.Model(json.RawMessage(model))
			}),
		)
		s.Mount("foo", m)
	}, func(s *Session) {
		// Test getting the model
		inb := s.Request("get.test.foo.model", newRequest())
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":{"model":`+model+`}}`))
	})
}

// Test Mount Mux to service root
func TestMountToRoot(t *testing.T) {
	model := resource["test.model"]

	runTest(t, func(s *Session) {
		m := res.NewMux("foo")
		m.Handle("model",
			res.GetModel(func(r res.ModelRequest) {
				r.Model(json.RawMessage(model))
			}),
		)
		s.Mount("", m)
	}, func(s *Session) {
		// Test getting the model
		inb := s.Request("get.test.foo.model", newRequest())
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":{"model":`+model+`}}`))
	})
}

// Test Mount root Mux to service
func TestMountRootMux(t *testing.T) {
	model := resource["test.model"]

	runTest(t, func(s *Session) {
		m := res.NewMux("")
		m.Handle("model",
			res.GetModel(func(r res.ModelRequest) {
				r.Model(json.RawMessage(model))
			}),
		)
		s.Mount("foo", m)
	}, func(s *Session) {
		// Test getting the model
		inb := s.Request("get.test.foo.model", newRequest())
		s.GetMsg(t).Equals(t, inb, json.RawMessage(`{"result":{"model":`+model+`}}`))
	})
}

// Test Mount root Mux to service root panics
func TestMountRootMuxToRoot(t *testing.T) {
	model := resource["test.model"]

	runTest(t, func(s *Session) {
		AssertPanic(t, func() {
			m := res.NewMux("")
			m.Handle("model",
				res.GetModel(func(r res.ModelRequest) {
					r.Model(json.RawMessage(model))
				}),
			)
			s.Mount("", m)
		})
	}, nil, withoutReset)
}

// Test Mount Mux twice panics
func TestMountMuxTwice(t *testing.T) {
	model := resource["test.model"]

	runTest(t, func(s *Session) {
		AssertPanic(t, func() {
			m := res.NewMux("")
			m.Handle("model",
				res.GetModel(func(r res.ModelRequest) {
					r.Model(json.RawMessage(model))
				}),
			)
			s.Mount("foo", m)
			s.Mount("bar", m)
		})
	}, nil, withoutReset)
}

// Test adding handler with a valid route pattern and handler path doesn't panic.
func TestAddHandlerWithValidPath(t *testing.T) {
	tbl := []struct {
		Pattern string
		Path    string
	}{
		{"", "model"},
		{"", "model.foo"},
		{"", "model.$id"},
		{"", "model.$id.foo"},
		{"", "model.>"},
		{"", "model.$id.>"},
		{"test", "model"},
		{"test", "model.foo"},
		{"test", "model.$id"},
		{"test", "model.$id.foo"},
		{"test", "model.>"},
		{"test", "model.$id.>"},
	}

	for _, l := range tbl {
		m := res.NewMux(l.Pattern)
		m.Handle(l.Path, res.GetResource(func(r res.GetRequest) { r.NotFound() }))
	}
}

// Test adding handler with an invalid route handler path causes panic.
func TestAddHandlerWithInvalidPathCausesPanic(t *testing.T) {
	tbl := []struct {
		Path string
	}{
		{"model.$id.type.$id"},
		{"model..foo"},
		{"model.$"},
		{"model.$.foo"},
		{"model.>.foo"},
		{"model.foo.>bar"},
	}

	for _, l := range tbl {
		AssertPanic(t, func() {
			m := res.NewMux("test")
			m.Handle(l.Path, res.GetResource(func(r res.GetRequest) { r.NotFound() }))
		})
	}
}

// Test adding duplicate handler path causes panic.
func TestAddDuplicateHandlerPathCausesPanic(t *testing.T) {
	m := res.NewMux("")
	m.Handle("test.model")
	AssertPanic(t, func() {
		m.Handle("test.model")
	})
}

// Test adding a handler with a valid group does not cause panic.
func TestAddHandlerWithValidGroup(t *testing.T) {
	tbl := []struct {
		Group   string
		Pattern string
	}{
		{"", "test"},
		{"test", "test"},
		{"test", "test.$foo"},
		{"test.${foo}", "test.$foo"},
		{"${foo}", "test.$foo"},
		{"${foo}.test", "test.$foo"},
		{"${foo}${bar}", "test.$foo.$bar"},
		{"${bar}${foo}", "test.$foo.$bar"},
		{"${foo}.${bar}", "test.$foo.$bar.>"},
		{"${foo}${foo}", "test.$foo.$bar"},
	}

	for _, l := range tbl {
		m := res.NewMux("")
		m.Handle(l.Pattern, res.Group(l.Group))
	}
}

// Test adding a handler with invalid group causes panic.
func TestAddHandlerWithInvalidGroupWillPanic(t *testing.T) {
	tbl := []struct {
		Group   string
		Pattern string
	}{
		{"$", "test.$foo"},
		{"${", "test.$foo"},
		{"${foo", "test.$foo"},
		{"${}", "test.$foo"},
		{"${$foo}", "test.$foo"},
		{"${bar}", "test.$foo"},
	}

	for _, l := range tbl {
		m := res.NewMux("")
		AssertPanic(t, func() {
			m.Handle(l.Pattern, res.Group(l.Group))
		})
	}
}

// Test Path with a valid pattern returns the Mux path.
func TestPathWithValidPath(t *testing.T) {
	tbl := []struct {
		Path string
	}{
		{"test"},
		{"test.foo"},
		{"test.foo.bar"},
	}

	for i, l := range tbl {
		m := res.NewMux(l.Path)
		AssertEqual(t, "Mux.Path", m.Path(), l.Path, "test ", i)
	}
}

// Test Path with a valid path returns the Mux path.
func TestPathWithMountedChild(t *testing.T) {
	tbl := []struct {
		Path             string
		SubPath          string
		MountPath        string
		ExpectedPath     string
		ExpectedFullPath string
	}{
		{"", "", "sub", "", "sub"},
		{"", "sub", "", "sub", "sub"},
		{"", "sub", "foo", "sub", "foo.sub"},
		{"", "sub", "$id", "sub", "$id.sub"},
		{"test", "", "sub", "", "test.sub"},
		{"test", "sub", "", "sub", "test.sub"},
		{"test", "sub", "foo", "sub", "test.foo.sub"},
		{"test", "sub", "$id", "sub", "test.$id.sub"},
		{"test.foo", "", "sub", "", "test.foo.sub"},
		{"test.foo", "sub", "", "sub", "test.foo.sub"},
		{"test.foo", "sub", "bar", "sub", "test.foo.bar.sub"},
		{"test.foo", "sub", "$id", "sub", "test.foo.$id.sub"},
	}

	for i, l := range tbl {
		m := res.NewMux(l.Path)
		sub := res.NewMux(l.SubPath)
		m.Mount(l.MountPath, sub)
		AssertEqual(t, "Mux.Path", sub.Path(), l.ExpectedPath, "test ", i)
		AssertEqual(t, "Mux.Path", sub.FullPath(), l.ExpectedFullPath, "test ", i)
	}
}

// Test GetHandler with matching path returns the registered handler.
func TestGetHandlerWithMatchingPathReturnsHandler(t *testing.T) {
	tbl := []struct {
		Pattern      string
		Path         string
		ResourceName string
	}{
		{"", "", ""},
		{"", "model", "model"},
		{"", "model.foo", "model.foo"},
		{"", "model.$id", "model.42"},
		{"", "model.$id.foo", "model.42.foo"},
		{"", "model.>", "model.foo"},
		{"", "model.>", "model.foo.bar"},
		{"", "model.$id.>", "model.42.foo"},
		{"", "model.$id.>", "model.42.foo.bar"},
		{"test", "", "test"},
		{"test", "model", "test.model"},
		{"test", "model.foo", "test.model.foo"},
		{"test", "model.$id", "test.model.42"},
		{"test", "model.$id.foo", "test.model.42.foo"},
		{"test", "model.>", "test.model.foo"},
		{"test", "model.>", "test.model.foo.bar"},
		{"test", "model.$id.>", "test.model.42.foo"},
		{"test", "model.$id.>", "test.model.42.foo.bar"},
	}

	for i, l := range tbl {
		called := 0
		m := res.NewMux(l.Pattern)
		m.Handle(l.Path, res.GetResource(func(r res.GetRequest) { called++ }))
		h, _, _, err := m.GetHandler(l.ResourceName)
		AssertNoError(t, err, "test ", i)
		h.Get(nil)
		AssertEqual(t, "called", called, 1, "test ", i)
	}
}

// Test GetHandler without matching path returns error.
func TestGetHandlerWithMismatchingPathReturnsError(t *testing.T) {
	tbl := []struct {
		Path         string
		Pattern      string
		ResourceName string
	}{
		{"", "", "model"},
		{"", "model", ""},
		{"", "model", "model.foo"},
		{"", "model.foo", "model"},
		{"", "model.$id", "model.42.foo"},
		{"", "model.$id.foo", "model.42"},
		{"", "model.>", "model"},
		{"", "model.$id.>", "model.42"},
		{"test", "", "model"},
		{"test", "model", "this.model"},
		{"test", "model", "test"},
		{"test", "model", "test.model.foo"},
		{"test", "model.foo", "test.model"},
		{"test", "model.$id", "test.model.42.foo"},
		{"test", "model.$id.foo", "test.model.42"},
		{"test", "model.>", "test.model"},
		{"test", "model.$id.>", "test.model.42"},
		{"test", "model", "test"},
	}

	for i, l := range tbl {
		m := res.NewMux(l.Path)
		m.Handle(l.Pattern)
		_, _, _, err := m.GetHandler(l.ResourceName)
		AssertError(t, err, "test ", i)
	}
}

// Test GetHandler with matching path and group returns handler.
func TestGetHandlerWithMatchingPathAndGroupReturnsHandler(t *testing.T) {
	tbl := []struct {
		Pattern       string
		Path          string
		ResourceName  string
		Group         string
		ExpectedGroup string
	}{
		{"", "model", "model", "foo", "foo"},
		{"", "model.foo", "model.foo", "bar", "bar"},
		{"", "model.$id", "model.42", "foo.bar", "foo.bar"},
		{"", "model.$id", "model.42", "${id}", "42"},
		{"", "model.$id", "model.42", "${id}foo", "42foo"},
		{"", "model.$id", "model.42", "foo${id}", "foo42"},
		{"", "model.$id", "model.42", "foo${id}bar", "foo42bar"},
		{"", "model.$id.$type", "model.42.foo", "foo.bar", "foo.bar"},
		{"", "model.$id.$type", "model.42.foo", "${id}", "42"},
		{"", "model.$id.$type", "model.42.foo", "${type}", "foo"},
		{"", "model.$id.$type", "model.42.foo", "${id}${type}", "42foo"},
		{"", "model.$id.$type", "model.42.foo", "${id}.${type}", "42.foo"},
		{"", "model.$id.$type", "model.42.foo", "${type}${id}", "foo42"},
		{"", "model.$id.$type", "model.42.foo", "bar.${type}.${id}.baz", "bar.foo.42.baz"},
		{"test", "model", "test.model", "foo", "foo"},
		{"test", "model.foo", "test.model.foo", "bar", "bar"},
		{"test", "model.$id", "test.model.42", "foo.bar", "foo.bar"},
		{"test", "model.$id", "test.model.42", "${id}", "42"},
		{"test", "model.$id", "test.model.42", "${id}foo", "42foo"},
		{"test", "model.$id", "test.model.42", "foo${id}", "foo42"},
		{"test", "model.$id", "test.model.42", "foo${id}bar", "foo42bar"},
		{"test", "model.$id.$type", "test.model.42.foo", "foo.bar", "foo.bar"},
		{"test", "model.$id.$type", "test.model.42.foo", "${id}", "42"},
		{"test", "model.$id.$type", "test.model.42.foo", "${type}", "foo"},
		{"test", "model.$id.$type", "test.model.42.foo", "${id}${type}", "42foo"},
		{"test", "model.$id.$type", "test.model.42.foo", "${id}.${type}", "42.foo"},
		{"test", "model.$id.$type", "test.model.42.foo", "${type}${id}", "foo42"},
		{"test", "model.$id.$type", "test.model.42.foo", "bar.${type}.${id}.baz", "bar.foo.42.baz"},
	}

	for i, l := range tbl {
		called := 0
		m := res.NewMux(l.Pattern)
		m.Handle(l.Path, res.Group(l.Group), res.GetResource(func(r res.GetRequest) { called++ }))
		h, _, g, err := m.GetHandler(l.ResourceName)
		AssertNoError(t, err, "test ", i)
		AssertEqual(t, "group", g, l.ExpectedGroup)
		h.Get(nil)
		AssertEqual(t, "called", called, 1, "test ", i)
	}
}

// Test GetHandler with matching path and group on mounted Mux returns handler.
func TestGetHandlerWithMatchingPathAndGroupOnMountedMuxReturnsHandler(t *testing.T) {
	tbl := []struct {
		Pattern       string
		Path          string
		ResourceName  string
		Group         string
		ExpectedGroup string
	}{
		{"", "model", "sub.model", "foo", "foo"},
		{"", "model.foo", "sub.model.foo", "bar", "bar"},
		{"", "model.$id", "sub.model.42", "foo.bar", "foo.bar"},
		{"", "model.$id", "sub.model.42", "${id}", "42"},
		{"", "model.$id", "sub.model.42", "${id}foo", "42foo"},
		{"", "model.$id", "sub.model.42", "foo${id}", "foo42"},
		{"", "model.$id", "sub.model.42", "foo${id}bar", "foo42bar"},
		{"", "model.$id.$type", "sub.model.42.foo", "foo.bar", "foo.bar"},
		{"", "model.$id.$type", "sub.model.42.foo", "${id}", "42"},
		{"", "model.$id.$type", "sub.model.42.foo", "${type}", "foo"},
		{"", "model.$id.$type", "sub.model.42.foo", "${id}${type}", "42foo"},
		{"", "model.$id.$type", "sub.model.42.foo", "${id}.${type}", "42.foo"},
		{"", "model.$id.$type", "sub.model.42.foo", "${type}${id}", "foo42"},
		{"", "model.$id.$type", "sub.model.42.foo", "bar.${type}.${id}.baz", "bar.foo.42.baz"},
		{"test", "model", "test.sub.model", "foo", "foo"},
		{"test", "model.foo", "test.sub.model.foo", "bar", "bar"},
		{"test", "model.$id", "test.sub.model.42", "foo.bar", "foo.bar"},
		{"test", "model.$id", "test.sub.model.42", "${id}", "42"},
		{"test", "model.$id", "test.sub.model.42", "${id}foo", "42foo"},
		{"test", "model.$id", "test.sub.model.42", "foo${id}", "foo42"},
		{"test", "model.$id", "test.sub.model.42", "foo${id}bar", "foo42bar"},
		{"test", "model.$id.$type", "test.sub.model.42.foo", "foo.bar", "foo.bar"},
		{"test", "model.$id.$type", "test.sub.model.42.foo", "${id}", "42"},
		{"test", "model.$id.$type", "test.sub.model.42.foo", "${type}", "foo"},
		{"test", "model.$id.$type", "test.sub.model.42.foo", "${id}${type}", "42foo"},
		{"test", "model.$id.$type", "test.sub.model.42.foo", "${id}.${type}", "42.foo"},
		{"test", "model.$id.$type", "test.sub.model.42.foo", "${type}${id}", "foo42"},
		{"test", "model.$id.$type", "test.sub.model.42.foo", "bar.${type}.${id}.baz", "bar.foo.42.baz"},
	}

	for i, l := range tbl {
		called := 0
		m := res.NewMux(l.Pattern)
		sub := res.NewMux("")
		sub.Handle(l.Path, res.Group(l.Group), res.GetResource(func(r res.GetRequest) { called++ }))
		m.Mount("sub", sub)
		h, _, g, err := m.GetHandler(l.ResourceName)
		AssertNoError(t, err, "test ", i)
		AssertEqual(t, "group", g, l.ExpectedGroup)
		h.Get(nil)
		AssertEqual(t, "called", called, 1, "test ", i)
	}
}

// Test GetHandler on more specific path returns the more specific handler.
func TestGetHandlerMoreSpecificPath(t *testing.T) {
	tbl := []struct {
		Pattern      string
		SpecificPath string
		WildcardPath string
		ResourceName string
	}{
		{"", "model", "$type", "model"},
		{"", "model.foo", "model.$id", "model.foo"},
		{"", "model.foo", "$type.foo", "model.foo"},
		{"", "model.$id", "model.>", "model.42"},
		{"", "model.$id.foo", "model.$id.$type", "model.42.foo"},
		{"", "model.$id.foo", "model.$id.>", "model.42.foo"},
		{"", "model.$id.foo", "model.>", "model.42.foo"},
		{"", "model.>", ">", "model.foo"},
		{"", "model.>", "$type.>", "model.foo"},
		{"", "model.$id.>", "model.>", "model.42.foo"},
		{"", "model.$id.>", "$type.>", "model.42.foo"},
		{"", "model.$id.>", ">", "model.42.foo"},
		{"test", "model", "$type", "test.model"},
		{"test", "model.foo", "model.$id", "test.model.foo"},
		{"test", "model.foo", "$type.foo", "test.model.foo"},
		{"test", "model.$id", "model.>", "test.model.42"},
		{"test", "model.$id.foo", "model.$id.$type", "test.model.42.foo"},
		{"test", "model.$id.foo", "model.$id.>", "test.model.42.foo"},
		{"test", "model.$id.foo", "model.>", "test.model.42.foo"},
		{"test", "model.>", ">", "test.model.foo"},
		{"test", "model.>", "$type.>", "test.model.foo"},
		{"test", "model.$id.>", "model.>", "test.model.42.foo"},
		{"test", "model.$id.>", "$type.>", "test.model.42.foo"},
		{"test", "model.$id.>", ">", "test.model.42.foo"},
	}

	for i, l := range tbl {
		specificCalled := 0
		wildcardCalled := 0
		m := res.NewMux(l.Pattern)
		m.Handle(l.SpecificPath, res.GetResource(func(r res.GetRequest) { specificCalled++ }))
		m.Handle(l.WildcardPath, res.GetResource(func(r res.GetRequest) { wildcardCalled++ }))
		h, _, _, err := m.GetHandler(l.ResourceName)
		AssertNoError(t, err, "test ", i)
		h.Get(nil)
		AssertEqual(t, "specificCalled", specificCalled, 1, "test ", i)
		AssertEqual(t, "wildcardCalled", wildcardCalled, 0, "test ", i)
	}
}

// Test Mount to subpath mounts router.
func TestMountToSubpath(t *testing.T) {
	tbl := []struct {
		RootPattern    string
		SubPattern     string
		MountPattern   string
		HandlerPattern string
		ResourceName   string
	}{
		{"", "", "sub", "model", "sub.model"},
		{"", "sub", "", "model", "sub.model"},
		{"test", "", "sub", "model", "test.sub.model"},
		{"test", "sub", "", "model", "test.sub.model"},
		{"test", "", "sub", "$id", "test.sub.foo"},
		{"test", "sub", "", ">", "test.sub.foo.bar"},
	}

	for i, l := range tbl {
		called := 0
		m := res.NewMux(l.RootPattern)
		sub := res.NewMux(l.SubPattern)
		sub.Handle(l.HandlerPattern, res.GetResource(func(r res.GetRequest) { called++ }))
		m.Mount(l.MountPattern, sub)
		h, _, _, err := m.GetHandler(l.ResourceName)
		AssertNoError(t, err, "test ", i)
		h.Get(nil)
		AssertEqual(t, "called", called, 1, "test ", i)
	}
}

// Test Mount to root causes panic.
func TestMountToRootCausesPanic(t *testing.T) {
	m := res.NewMux("test")
	sub := res.NewMux("")
	AssertPanic(t, func() { m.Mount("", sub) })
}

// Test Mount an already mounted Mux causes panic.
func TestMountAMountedMuxCausesPanic(t *testing.T) {
	m1 := res.NewMux("test1")
	m2 := res.NewMux("test2")
	sub := res.NewMux("sub")
	m1.Mount("", sub)
	AssertPanic(t, func() { m2.Mount("", sub) })
}

// Test Mount to existing pattern causes panic.
func TestMountToExistingPatternCausesPanic(t *testing.T) {
	tbl := []struct {
		Pattern string
	}{
		{"model"},
		{"model.foo"},
		{"model.$id"},
		{"model.>"},
	}

	for i, l := range tbl {
		m := res.NewMux("test")
		m.Handle(l.Pattern)
		sub := res.NewMux("")
		AssertPanic(t, func() { m.Mount(l.Pattern, sub) }, "test ", i)
	}
}

var mountedSubrouterTestData = []struct {
	RootPattern    string
	HandlerPattern string
	ResourceName   string
	ExpectedParams string
}{
	{"", "model", "sub.model", `null`},
	{"", "model.foo", "sub.model.foo", `null`},
	{"", "model.$id", "sub.model.42", `{"id":"42"}`},
	{"", "model.$id.foo", "sub.model.42.foo", `{"id":"42"}`},
	{"", "model.>", "sub.model.foo", `null`},
	{"", "model.>", "sub.model.foo.bar", `null`},
	{"", "model.$id.>", "sub.model.42.foo", `{"id":"42"}`},
	{"", "model.$id.>", "sub.model.42.foo.bar", `{"id":"42"}`},
	{"test", "model", "test.sub.model", `null`},
	{"test", "model.foo", "test.sub.model.foo", `null`},
	{"test", "model.$id", "test.sub.model.42", `{"id":"42"}`},
	{"test", "model.$id.foo", "test.sub.model.42.foo", `{"id":"42"}`},
	{"test", "model.>", "test.sub.model.foo", `null`},
	{"test", "model.>", "test.sub.model.foo.bar", `null`},
	{"test", "model.$id.>", "test.sub.model.42.foo", `{"id":"42"}`},
	{"test", "model.$id.>", "test.sub.model.42.foo.bar", `{"id":"42"}`},
}

// Test GetHandler from mounted child Mux gets the handler.
func TestGetHandlerFromMountedChildMux(t *testing.T) {
	for i, l := range mountedSubrouterTestData {
		called := 0
		m := res.NewMux(l.RootPattern)
		sub := res.NewMux("")
		sub.Handle(l.HandlerPattern, res.GetResource(func(r res.GetRequest) { called++ }))
		m.Mount("sub", sub)
		h, p, _, err := m.GetHandler(l.ResourceName)
		AssertNoError(t, err, "test ", i)
		h.Get(nil)
		AssertEqual(t, "called", called, 1, "test ", i)
		AssertEqual(t, "pathParams", p, json.RawMessage(l.ExpectedParams), "test ", i)
	}
}

// Test GetHandler on handler added to a mounted Mux after the mount.
func TestGetHandlerAddedAfterBeingMounted(t *testing.T) {
	for i, l := range mountedSubrouterTestData {
		called := 0
		m := res.NewMux(l.RootPattern)
		sub := res.NewMux("")
		m.Mount("sub", sub)
		m.Handle("sub."+l.HandlerPattern, res.GetResource(func(r res.GetRequest) { called++ }))
		h, p, _, err := m.GetHandler(l.ResourceName)
		AssertNoError(t, err, "test ", i)
		h.Get(nil)
		AssertEqual(t, "called", called, 1, "test ", i)
		AssertEqual(t, "pathParams", p, json.RawMessage(l.ExpectedParams), "test ", i)
	}
}

// Test Contains with single added handler.
func TestContainsWithSinglePath(t *testing.T) {
	tbl := []struct {
		Path    string
		Pattern string
	}{
		{"", "model"},
		{"", "model.foo"},
		{"", "model.$id"},
		{"", "model.$id.foo"},
		{"", "model.>"},
		{"", "model.$id.>"},
		{"test", "model"},
		{"test", "model.foo"},
		{"test", "model.$id"},
		{"test", "model.$id.foo"},
		{"test", "model.>"},
		{"test", "model.$id.>"},
	}

	for i, l := range tbl {
		m := res.NewMux(l.Path)
		m.Handle(l.Pattern)
		AssertTrue(t, "Contains to return true", m.Contains(func(h res.Handler) bool { return true }), "test ", i)
	}
}

// Test Contains with overlapping handler paths.
func TestContainsWithOverlappingPaths(t *testing.T) {
	tbl := []struct {
		Path            string
		SpecificPattern string
		WildcardPattern string
	}{
		{"", "model", "$type"},
		{"", "model.foo", "model.$id"},
		{"", "model.foo", "$type.foo"},
		{"", "model.$id", "model.>"},
		{"", "model.$id.foo", "model.$id.$type"},
		{"", "model.$id.foo", "model.$id.>"},
		{"", "model.$id.foo", "model.>"},
		{"", "model.>", ">"},
		{"", "model.>", "$type.>"},
		{"", "model.$id.>", "model.>"},
		{"", "model.$id.>", "$type.>"},
		{"", "model.$id.>", ">"},
		{"test", "model", "$type"},
		{"test", "model.foo", "model.$id"},
		{"test", "model.foo", "$type.foo"},
		{"test", "model.$id", "model.>"},
		{"test", "model.$id.foo", "model.$id.$type"},
		{"test", "model.$id.foo", "model.$id.>"},
		{"test", "model.$id.foo", "model.>"},
		{"test", "model.>", ">"},
		{"test", "model.>", "$type.>"},
		{"test", "model.$id.>", "model.>"},
		{"test", "model.$id.>", "$type.>"},
		{"test", "model.$id.>", ">"},
	}

	for i, l := range tbl {
		m := res.NewMux(l.Path)
		m.Handle(l.SpecificPattern, res.Model)
		m.Handle(l.WildcardPattern, res.Collection)
		AssertTrue(t, "Contains TypeModel to return true", m.Contains(func(h res.Handler) bool { return h.Type == res.TypeModel }), "test ", i)
		AssertTrue(t, "Contains TypeCollection to return true", m.Contains(func(h res.Handler) bool { return h.Type == res.TypeCollection }), "test ", i)
		AssertTrue(t, "Contains TypeUnset to return false", !m.Contains(func(h res.Handler) bool { return h.Type == res.TypeUnset }), "test ", i)
	}
}
