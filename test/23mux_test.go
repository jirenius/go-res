package test

import (
	"encoding/json"
	"fmt"
	"testing"

	res "github.com/jirenius/go-res"
	"github.com/jirenius/go-res/restest"
)

var getMatchingPathTestSets = []struct {
	Path         string
	Pattern      string
	ResourceName string
}{
	{"", "model", "model"},
	{"", "model.foo", "model.foo"},
	{"", "model.$id", "model.42"},
	{"", "model.$id.foo", "model.42.foo"},
	{"", "model.>", "model.foo"},
	{"", "model.>", "model.foo.bar"},
	{"", "model.$id.>", "model.42.foo"},
	{"", "model.$id.>", "model.42.foo.bar"},
	{"", "$id", "model"},
	{"", ">", "model.foo"},
	{"", "$id.>", "model.foo.bar"},
	{"", "$id.$foo", "model.foo"},
	{"test", "model", "test.model"},
	{"test", "model.foo", "test.model.foo"},
	{"test", "model.$id", "test.model.42"},
	{"test", "model.$id.foo", "test.model.42.foo"},
	{"test", "model.>", "test.model.foo"},
	{"test", "model.>", "test.model.foo.bar"},
	{"test", "model.$id.>", "test.model.42.foo"},
	{"test", "model.$id.>", "test.model.42.foo.bar"},
	{"test", "$id", "test.model"},
	{"test", ">", "test.model.foo"},
	{"test", "$id.>", "test.model.foo.bar"},
	{"test", "$id.$foo", "test.model.foo"},
}

var getMatchingMountedPathTestSets = []struct {
	Path         string
	Pattern      string
	ResourceName string
}{
	{"", "model", "sub.model"},
	{"", "model.foo", "sub.model.foo"},
	{"", "model.$id", "sub.model.42"},
	{"", "model.$id.foo", "sub.model.42.foo"},
	{"", "model.>", "sub.model.foo"},
	{"", "model.>", "sub.model.foo.bar"},
	{"", "model.$id.>", "sub.model.42.foo"},
	{"", "model.$id.>", "sub.model.42.foo.bar"},
	{"", "$id", "sub.model"},
	{"", ">", "sub.model.foo"},
	{"", "$id.>", "sub.model.foo.bar"},
	{"", "$id.$foo", "sub.model.foo"},
	{"test", "model", "test.sub.model"},
	{"test", "model.foo", "test.sub.model.foo"},
	{"test", "model.$id", "test.sub.model.42"},
	{"test", "model.$id.foo", "test.sub.model.42.foo"},
	{"test", "model.>", "test.sub.model.foo"},
	{"test", "model.>", "test.sub.model.foo.bar"},
	{"test", "model.$id.>", "test.sub.model.42.foo"},
	{"test", "model.$id.>", "test.sub.model.42.foo.bar"},
	{"test", "$id", "test.sub.model"},
	{"test", ">", "test.sub.model.foo"},
	{"test", "$id.>", "test.sub.model.foo.bar"},
	{"test", "$id.$foo", "test.sub.model.foo"},
}

// Test Route adds to the path of the the parent
func TestRoute(t *testing.T) {
	runTest(t, func(s *res.Service) {
		s.Route("foo", func(m *res.Mux) {
			m.Handle("model",
				res.GetModel(func(r res.ModelRequest) {
					r.Model(mock.Model)
				}),
			)
		})
	}, func(s *restest.Session) {
		// Test getting the model
		s.Get("test.foo.model").
			Response().
			AssertModel(mock.Model)
	})
}

// Test Mount Mux to service
func TestMount(t *testing.T) {
	runTest(t, func(s *res.Service) {
		m := res.NewMux("")
		m.Handle("model",
			res.GetModel(func(r res.ModelRequest) {
				r.Model(mock.Model)
			}),
		)
		s.Mount("foo", m)
	}, func(s *restest.Session) {
		// Test getting the model
		s.Get("test.foo.model").
			Response().
			AssertModel(mock.Model)
	})
}

// Test Mount Mux to service root
func TestMountToRoot(t *testing.T) {
	runTest(t, func(s *res.Service) {
		m := res.NewMux("foo")
		m.Handle("model",
			res.GetModel(func(r res.ModelRequest) {
				r.Model(mock.Model)
			}),
		)
		s.Mount("", m)
	}, func(s *restest.Session) {
		// Test getting the model
		s.Get("test.foo.model").
			Response().
			AssertModel(mock.Model)
	})
}

// Test Mount root Mux to service
func TestMountRootMux(t *testing.T) {
	runTest(t, func(s *res.Service) {
		m := res.NewMux("")
		m.Handle("model",
			res.GetModel(func(r res.ModelRequest) {
				r.Model(mock.Model)
			}),
		)
		s.Mount("foo", m)
	}, func(s *restest.Session) {
		// Test getting the model
		s.Get("test.foo.model").
			Response().
			AssertModel(mock.Model)
	})
}

// Test Mount root Mux to service root panics
func TestMountRootMuxToRoot(t *testing.T) {
	runTest(t, func(s *res.Service) {
		restest.AssertPanic(t, func() {
			m := res.NewMux("")
			m.Handle("model",
				res.GetModel(func(r res.ModelRequest) {
					r.Model(mock.Model)
				}),
			)
			s.Mount("", m)
		})
	}, nil, restest.WithoutReset)
}

// Test Mount Mux twice panics
func TestMountMuxTwice(t *testing.T) {
	runTest(t, func(s *res.Service) {
		restest.AssertPanic(t, func() {
			m := res.NewMux("")
			m.Handle("model",
				res.GetModel(func(r res.ModelRequest) {
					r.Model(mock.Model)
				}),
			)
			s.Mount("foo", m)
			s.Mount("bar", m)
		})
	}, nil, restest.WithoutReset)
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
		Path    string
		Pattern string
	}{
		{"test", "model.$id.type.$id"},
		{"test", "model..foo"},
		{"test", "model.$"},
		{"test", "model.$.foo"},
		{"test", "model.>.foo"},
		{"test", "model.foo.>bar"},
		{"$id", "model"},
		{">", "model"},
		{"test.>", "model"},
		{"test.", "model"},
		{"test.$id", "model"},
		{"test..foo", "model"},
	}

	for i, l := range tbl {
		restest.AssertPanic(t, func() {
			m := res.NewMux(l.Path)
			m.Handle(l.Pattern, res.GetResource(func(r res.GetRequest) { r.NotFound() }))
		}, "test ", i)
	}
}

// Test adding duplicate handler path causes panic.
func TestAddDuplicateHandlerPathCausesPanic(t *testing.T) {
	m := res.NewMux("")
	m.Handle("test.model")
	restest.AssertPanic(t, func() {
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
		restest.AssertPanic(t, func() {
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
		restest.AssertEqualJSON(t, "Mux.Path", m.Path(), l.Path, "test ", i)
		restest.AssertEqualJSON(t, "Mux.FullPath", m.FullPath(), l.Path, "test ", i)
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
		{"test", "", "sub", "", "test.sub"},
		{"test", "sub", "", "sub", "test.sub"},
		{"test", "sub", "foo", "sub", "test.foo.sub"},
		{"test.foo", "", "sub", "", "test.foo.sub"},
		{"test.foo", "sub", "", "sub", "test.foo.sub"},
		{"test.foo", "sub", "bar", "sub", "test.foo.bar.sub"},
	}

	for i, l := range tbl {
		m := res.NewMux(l.Path)
		sub := res.NewMux(l.SubPath)
		m.Mount(l.MountPath, sub)
		restest.AssertEqualJSON(t, "Mux.Path", sub.Path(), l.ExpectedPath, "test ", i)
		restest.AssertEqualJSON(t, "Mux.Path", sub.FullPath(), l.ExpectedFullPath, "test ", i)
	}
}

func TestMuxGetHandler_MatchingPath_ReturnsHandler(t *testing.T) {
	for i, l := range getMatchingPathTestSets {
		called := 0
		m := res.NewMux(l.Path)
		m.Handle(l.Pattern, res.GetResource(func(r res.GetRequest) { called++ }))
		mh := m.GetHandler(l.ResourceName)
		restest.AssertNotNil(t, mh, "test ", i)
		mh.Handler.Get(nil)
		restest.AssertEqualJSON(t, "called", called, 1, "test ", i)
	}
}

func TestMuxGetHandler_MatchingPathAddedBeforeMount_ReturnsHandler(t *testing.T) {
	for i, l := range getMatchingMountedPathTestSets {
		called := 0
		m := res.NewMux(l.Path)
		sub := res.NewMux("")
		sub.Handle(l.Pattern, res.GetResource(func(r res.GetRequest) { called++ }))
		m.Mount("sub", sub)
		mh := m.GetHandler(l.ResourceName)
		restest.AssertNotNil(t, mh, "test ", i)
		mh.Handler.Get(nil)
		restest.AssertEqualJSON(t, "called", called, 1, "test ", i)
	}
}

func TestMuxGetHandler_MatchingPathAddedAfterMount_ReturnsHandler(t *testing.T) {
	for i, l := range getMatchingMountedPathTestSets {
		called := 0
		m := res.NewMux(l.Path)
		sub := res.NewMux("")
		m.Mount("sub", sub)
		m.Handle("sub."+l.Pattern, res.GetResource(func(r res.GetRequest) { called++ }))
		mh := m.GetHandler(l.ResourceName)
		restest.AssertNotNil(t, mh, "test ", i)
		mh.Handler.Get(nil)
		restest.AssertEqualJSON(t, "called", called, 1, "test ", i)
	}
}

func TestMuxGetHandler_ListenersOnMatchingPath_ReturnsListeners(t *testing.T) {
	for i, l := range getMatchingPathTestSets {
		called1 := 0
		called2 := 0
		m := res.NewMux(l.Path)
		m.Handle(l.Pattern, res.Model)
		m.AddListener(l.Pattern, func(ev *res.Event) { called1++ })
		m.AddListener(l.Pattern, func(ev *res.Event) { called2++ })
		mh := m.GetHandler(l.ResourceName)
		restest.AssertNotNil(t, mh, "test ", i)
		restest.AssertEqualJSON(t, "len(mh.Listeners)", len(mh.Listeners), 2, "test ", i)
		restest.AssertTrue(t, "cb1 not to be called", called1 == 0, "test ", i)
		mh.Listeners[0](nil)
		restest.AssertTrue(t, "cb1 to be called once", called1 == 1, "test ", i)
		restest.AssertTrue(t, "cb2 not to be called", called2 == 0, "test ", i)
		mh.Listeners[1](nil)
		restest.AssertTrue(t, "cb2 to be called once", called2 == 1, "test ", i)
	}
}

func TestMuxGetHandler_ListenersMatchingPathAddedBeforeMount_ReturnsListeners(t *testing.T) {
	for i, l := range getMatchingMountedPathTestSets {
		called1 := 0
		called2 := 0
		m := res.NewMux(l.Path)
		sub := res.NewMux("")
		sub.AddListener(l.Pattern, func(ev *res.Event) { called1++ })
		sub.AddListener(l.Pattern, func(ev *res.Event) { called2++ })
		m.Mount("sub", sub)
		m.Handle("sub."+l.Pattern, res.Model)
		mh := m.GetHandler(l.ResourceName)
		restest.AssertNotNil(t, mh, "test ", i)
		restest.AssertEqualJSON(t, "len(mh.Listeners)", len(mh.Listeners), 2, "test ", i)
		restest.AssertTrue(t, "cb1 not to be called", called1 == 0, "test ", i)
		mh.Listeners[0](nil)
		restest.AssertTrue(t, "cb1 to be called once", called1 == 1, "test ", i)
		restest.AssertTrue(t, "cb2 not to be called", called2 == 0, "test ", i)
		mh.Listeners[1](nil)
		restest.AssertTrue(t, "cb2 to be called once", called2 == 1, "test ", i)
	}
}

func TestMuxGetHandler_ListenersMatchingPathAddedAfterMount_ReturnsListeners(t *testing.T) {
	for i, l := range getMatchingMountedPathTestSets {
		called1 := 0
		called2 := 0
		m := res.NewMux(l.Path)
		sub := res.NewMux("")
		sub.Handle(l.Pattern, res.Model)
		m.Mount("sub", sub)
		m.AddListener("sub."+l.Pattern, func(ev *res.Event) { called1++ })
		m.AddListener("sub."+l.Pattern, func(ev *res.Event) { called2++ })
		mh := m.GetHandler(l.ResourceName)
		restest.AssertNotNil(t, mh, "test ", i)
		restest.AssertEqualJSON(t, "len(mh.Listeners)", len(mh.Listeners), 2, "test ", i)
		restest.AssertTrue(t, "cb1 not to be called", called1 == 0, "test ", i)
		mh.Listeners[0](nil)
		restest.AssertTrue(t, "cb1 to be called once", called1 == 1, "test ", i)
		restest.AssertTrue(t, "cb2 not to be called", called2 == 0, "test ", i)
		mh.Listeners[1](nil)
		restest.AssertTrue(t, "cb2 to be called once", called2 == 1, "test ", i)
	}
}

// Test GetHandler without matching path returns error.
func TestGetHandlerWithMismatchingPathReturnsNil(t *testing.T) {
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
		mh := m.GetHandler(l.ResourceName)
		restest.AssertTrue(t, "*Match to equal nil", mh == nil, "test ", i)
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
		mh := m.GetHandler(l.ResourceName)
		restest.AssertNotNil(t, mh, "test ", i)
		restest.AssertEqualJSON(t, "group", mh.Group, l.ExpectedGroup)
		mh.Handler.Get(nil)
		restest.AssertEqualJSON(t, "called", called, 1, "test ", i)
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
		mh := m.GetHandler(l.ResourceName)
		restest.AssertNotNil(t, mh, "test ", i)
		restest.AssertEqualJSON(t, "group", mh.Group, l.ExpectedGroup)
		mh.Handler.Get(nil)
		restest.AssertEqualJSON(t, "called", called, 1, "test ", i)
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
		mh := m.GetHandler(l.ResourceName)
		restest.AssertNotNil(t, mh, "test ", i)
		mh.Handler.Get(nil)
		restest.AssertEqualJSON(t, "specificCalled", specificCalled, 1, "test ", i)
		restest.AssertEqualJSON(t, "wildcardCalled", wildcardCalled, 0, "test ", i)
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
		mh := m.GetHandler(l.ResourceName)
		restest.AssertNotNil(t, mh, "test ", i)
		mh.Handler.Get(nil)
		restest.AssertEqualJSON(t, "called", called, 1, "test ", i)
	}
}

// Test Mount to root causes panic.
func TestMountToRootCausesPanic(t *testing.T) {
	m := res.NewMux("test")
	sub := res.NewMux("")
	restest.AssertPanic(t, func() { m.Mount("", sub) })
}

// Test Mount an already mounted Mux causes panic.
func TestMountAMountedMuxCausesPanic(t *testing.T) {
	m1 := res.NewMux("test1")
	m2 := res.NewMux("test2")
	sub := res.NewMux("sub")
	m1.Mount("", sub)
	restest.AssertPanic(t, func() { m2.Mount("", sub) })
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
		restest.AssertPanic(t, func() { m.Mount(l.Pattern, sub) }, "test ", i)
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
		mh := m.GetHandler(l.ResourceName)
		restest.AssertNotNil(t, mh, "test ", i)
		mh.Handler.Get(nil)
		restest.AssertEqualJSON(t, "called", called, 1, "test ", i)
		restest.AssertEqualJSON(t, "pathParams", mh.Params, json.RawMessage(l.ExpectedParams), "test ", i)
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
		mh := m.GetHandler(l.ResourceName)
		restest.AssertNotNil(t, mh, "test ", i)
		mh.Handler.Get(nil)
		restest.AssertEqualJSON(t, "called", called, 1, "test ", i)
		restest.AssertEqualJSON(t, "pathParams", mh.Params, json.RawMessage(l.ExpectedParams), "test ", i)
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
		restest.AssertTrue(t, "Contains to return true", m.Contains(func(h res.Handler) bool { return true }), "test ", i)
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
		restest.AssertTrue(t, "Contains TypeModel to return true", m.Contains(func(h res.Handler) bool { return h.Type == res.TypeModel }), "test ", i)
		restest.AssertTrue(t, "Contains TypeCollection to return true", m.Contains(func(h res.Handler) bool { return h.Type == res.TypeCollection }), "test ", i)
		restest.AssertTrue(t, "Contains TypeUnset to return false", !m.Contains(func(h res.Handler) bool { return h.Type == res.TypeUnset }), "test ", i)
	}
}

// Test OnRegister on a Handler to be called when adding handler to Mux that is registered to a service.
func TestMuxOnRegister_WithService_CallsCallback(t *testing.T) {
	tbl := []struct {
		Path         string
		Pattern      string
		ExpectedPath string
	}{
		{"", "model", "model"},
		{"", "model.foo", "model.foo"},
		{"", "model.$id", "model.$id"},
		{"", "model.>", "model.>"},
		{"", "$id", "$id"},
		{"", ">", ">"},
		{"", "$id.>", "$id.>"},
		{"", "$id.$foo", "$id.$foo"},

		{"test", "model", "test.model"},
		{"test", "model.foo", "test.model.foo"},
		{"test", "model.$id", "test.model.$id"},
		{"test", "model.>", "test.model.>"},
		{"test", "$id", "test.$id"},
		{"test", ">", "test.>"},
		{"test", "$id.>", "test.$id.>"},
		{"test", "$id.$foo", "test.$id.$foo"},
	}

	for i, l := range tbl {
		s := res.NewService(l.Path)
		called := false
		s.Handle(l.Pattern, res.OnRegister(func(service *res.Service, pattern res.Pattern, h res.Handler) {
			called = true
			restest.AssertTrue(t, "service to be passed as argument", s == service)
			restest.AssertEqualJSON(t, "pattern", pattern, l.ExpectedPath)
			restest.AssertTrue(t, "handler.OnRegister to be set", h.OnRegister != nil)
		}))
		restest.AssertTrue(t, "callback to be called", called, fmt.Sprintf("test #%d", i+1))
	}
}

func TestMuxOnRegister_MultipleListenersWithService_CallsCallbacks(t *testing.T) {
	s := res.NewService("test")
	called1 := 0
	called2 := 0
	s.Handle("model",
		res.OnRegister(func(service *res.Service, pattern res.Pattern, h res.Handler) {
			called1++
			restest.AssertEqualJSON(t, "pattern", pattern, "test.model")
			restest.AssertTrue(t, "handler.OnRegister to be set", h.OnRegister != nil)
		}),
		res.OnRegister(func(service *res.Service, pattern res.Pattern, h res.Handler) {
			called2++
			restest.AssertEqualJSON(t, "pattern", pattern, "test.model")
			restest.AssertTrue(t, "handler.OnRegister to be set", h.OnRegister != nil)
		}),
	)
	restest.AssertTrue(t, "callback 1 to be called once", called1 == 1)
	restest.AssertTrue(t, "callback 2 to be called once", called2 == 1)
}

func TestMuxOnRegister_BeforeMountingToService_CallsCallback(t *testing.T) {
	tbl := []struct {
		Path         string
		Pattern      string
		ExpectedPath string
	}{
		{"", "model", "sub.model"},
		{"", "model.foo", "sub.model.foo"},
		{"", "model.$id", "sub.model.$id"},
		{"", "model.>", "sub.model.>"},
		{"", "$id", "sub.$id"},
		{"", ">", "sub.>"},
		{"", "$id.>", "sub.$id.>"},
		{"", "$id.$foo", "sub.$id.$foo"},

		{"test", "model", "test.sub.model"},
		{"test", "model.foo", "test.sub.model.foo"},
		{"test", "model.$id", "test.sub.model.$id"},
		{"test", "model.>", "test.sub.model.>"},
		{"test", "$id", "test.sub.$id"},
		{"test", ">", "test.sub.>"},
		{"test", "$id.>", "test.sub.$id.>"},
		{"test", "$id.$foo", "test.sub.$id.$foo"},
	}

	for i, l := range tbl {
		s := res.NewService(l.Path)
		m := res.NewMux("")
		called := false
		m.Handle(l.Pattern, res.OnRegister(func(service *res.Service, pattern res.Pattern, h res.Handler) {
			called = true
			restest.AssertTrue(t, "service to be passed as argument", s == service)
			restest.AssertEqualJSON(t, "pattern", pattern, l.ExpectedPath)
			restest.AssertTrue(t, "handler.OnRegister to be set", h.OnRegister != nil)
		}))
		restest.AssertTrue(t, "callback not to be called", !called, fmt.Sprintf("test #%d", i+1))
		s.Mount("sub", m)
		restest.AssertTrue(t, "callback to be called", called, fmt.Sprintf("test #%d", i+1))
	}
}

func TestMuxOnRegister_AfterMountingToService_CallsCallback(t *testing.T) {
	tbl := []struct {
		Path         string
		Pattern      string
		ExpectedPath string
	}{
		{"", "model", "sub.model"},
		{"", "model.foo", "sub.model.foo"},
		{"", "model.$id", "sub.model.$id"},
		{"", "model.>", "sub.model.>"},
		{"", "$id", "sub.$id"},
		{"", ">", "sub.>"},
		{"", "$id.>", "sub.$id.>"},
		{"", "$id.$foo", "sub.$id.$foo"},

		{"test", "model", "test.sub.model"},
		{"test", "model.foo", "test.sub.model.foo"},
		{"test", "model.$id", "test.sub.model.$id"},
		{"test", "model.>", "test.sub.model.>"},
		{"test", "$id", "test.sub.$id"},
		{"test", ">", "test.sub.>"},
		{"test", "$id.>", "test.sub.$id.>"},
		{"test", "$id.$foo", "test.sub.$id.$foo"},
	}

	for i, l := range tbl {
		s := res.NewService(l.Path)
		m := res.NewMux("")
		s.Mount("sub", m)
		called := false
		m.Handle(l.Pattern, res.OnRegister(func(service *res.Service, pattern res.Pattern, h res.Handler) {
			called = true
			restest.AssertTrue(t, "service to be passed as argument", s == service)
			restest.AssertEqualJSON(t, "pattern", pattern, l.ExpectedPath)
			restest.AssertTrue(t, "handler.OnRegister to be set", h.OnRegister != nil)
		}))
		restest.AssertTrue(t, "callback to be called", called, fmt.Sprintf("test #%d", i+1))
	}
}
