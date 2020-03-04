package store

import (
	"fmt"
	"reflect"

	res "github.com/jirenius/go-res"
)

// TransformFuncs implements the Transformer interface by calling the functions
// for transforming store requests.
type transformer struct {
	ridToID   func(rid string, pathParams map[string]string) string
	idToRID   func(id string, v interface{}, p res.Pattern) string
	transform func(id string, v interface{}) (interface{}, error)
}

var _ Transformer = transformer{}

// TransformFuncs returns a Transformer that uses the provided functions. Any
// nil function will pass the value untransformed.
func TransformFuncs(ridToID func(rid string, pathParams map[string]string) string, idToRID func(id string, v interface{}, p res.Pattern) string, transform func(id string, v interface{}) (interface{}, error)) Transformer {
	return transformer{
		ridToID:   ridToID,
		idToRID:   idToRID,
		transform: transform,
	}
}

func (t transformer) RIDToID(rid string, pathParams map[string]string) string {
	if t.ridToID == nil {
		return rid
	}
	return t.ridToID(rid, pathParams)
}

func (t transformer) IDToRID(id string, v interface{}, p res.Pattern) string {
	if t.idToRID == nil {
		return id
	}
	return t.idToRID(id, v, p)
}

func (t transformer) Transform(id string, v interface{}) (interface{}, error) {
	if t.transform == nil {
		return v, nil
	}
	return t.transform(id, v)
}

// IDTransformer returns a transformer where the resource ID contains a single
// tag that is the internal ID.
//
//  // Assuming pattern is "library.book.$bookid"
//  IDTransformer("bookId", nil) // transforms "library.book.42" <=> "42"
func IDTransformer(tagName string, transform func(id string, v interface{}) (interface{}, error)) Transformer {
	return TransformFuncs(
		func(_ string, pathParams map[string]string) string {
			return pathParams[string(tagName)]
		},
		func(id string, _ interface{}, p res.Pattern) string {
			return string(p.ReplaceTag(string(tagName), id))
		},
		transform,
	)
}

// IDToRIDCollectionTransformer is a QueryTransformer that handles the common
// case of transforming a slice of id strings:
//
//  []string{"1", "2"}
//
// into slice of resource references:
//
//  []res.Ref{"library.book.1", "library.book.2"}
//
// The function converts a single ID returned by a the store into an external
// resource ID.
type IDToRIDCollectionTransformer func(id string) string

// TransformResult transforms a slice of id strings into a slice of resource
// references.
func (t IDToRIDCollectionTransformer) TransformResult(v interface{}) (interface{}, error) {
	ids, ok := v.([]string)
	if !ok {
		return nil, fmt.Errorf("failed to transform results: expected value of type []string, but got %s", reflect.TypeOf(v))
	}
	refs := make([]res.Ref, len(ids))
	for i, id := range ids {
		refs[i] = res.Ref(t(id))
	}
	return refs, nil
}

// TransformEvents transforms events for a []string collection into events for a
// []res.Ref collection.
func (t IDToRIDCollectionTransformer) TransformEvents(evs []ResultEvent) ([]ResultEvent, error) {
	for i, ev := range evs {
		if ev.Name == "add" {
			id, ok := ev.Value.(string)
			if !ok {
				return nil, fmt.Errorf("failed to transform add event: expected value of type string, but got %s", reflect.TypeOf(ev.Value))
			}
			evs[i].Value = res.Ref(t(id))
		}
	}
	return evs, nil
}

// IDToRIDModelTransformer is a QueryTransformer that handles the common case of
// transforming a slice of unique id strings:
//
//  []string{"1", "2"}
//
// into a map of resource references with id as key:
//
//  map[string]res.Ref{"1": "library.book.1", "2": "library.book.2"}
//
// The function converts a single ID returned by a the store into an external
// resource ID.
//
// The behavior is undefined for slices containing duplicate id string.
type IDToRIDModelTransformer func(id string) string

// TransformResult transforms a slice of id strings into a map of resource
// references with id as key.
func (t IDToRIDModelTransformer) TransformResult(v interface{}) (interface{}, error) {
	ids, ok := v.([]string)
	if !ok {
		return nil, fmt.Errorf("failed to transform results: expected value of type []string, but got %s", reflect.TypeOf(v))
	}
	refs := make(map[string]res.Ref, len(ids))
	for _, id := range ids {
		refs[id] = res.Ref(t(id))
	}
	return refs, nil
}

// TransformEvents transforms events for a []string collection into events for a
// map[string]res.Ref model.
func (t IDToRIDModelTransformer) TransformEvents(evs []ResultEvent) ([]ResultEvent, error) {
	if len(evs) == 0 {
		return evs, nil
	}
	ch := make(map[string]interface{}, len(evs))
	for _, ev := range evs {
		switch ev.Name {
		case "add":
			id, ok := ev.Value.(string)
			if !ok {
				return nil, fmt.Errorf("failed to transform add event: expected value of type string, but got %s", reflect.TypeOf(ev.Value))
			}
			ch[id] = res.Ref(t(id))
		case "remove":
			id, ok := ev.Value.(string)
			if !ok {
				return nil, fmt.Errorf("failed to transform remove event: expected value of type string, but got %s", reflect.TypeOf(ev.Value))
			}
			ch[id] = res.DeleteAction
		}
	}
	return evs, nil
}
