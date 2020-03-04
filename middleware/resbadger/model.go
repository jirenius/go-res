package resbadger

import (
	"encoding/json"
	"errors"
	"reflect"

	"github.com/dgraph-io/badger"
	res "github.com/jirenius/go-res"
)

// Model represents a model that is stored in the badger DB by its resource ID.
type Model struct {
	// BadgerDB middleware
	BadgerDB BadgerDB
	// Default resource value if not found in database.
	// Will return res.ErrNotFound if not set.
	Default interface{}
	// Type used to marshal into when calling r.Value() or r.RequireValue().
	// Defaults to map[string]interface{} if not set.
	Type interface{}
	// IndexSet defines a set of indexes to be created for the model.
	IndexSet *IndexSet
	// Map defines a map callback to transform the model when
	// responding to get requests.
	Map func(interface{}) (interface{}, error)
}

// WithDefault returns a new BadgerDB value with the Default resource value set to i.
func (o Model) WithDefault(i interface{}) Model {
	o.Default = i
	return o
}

// WithType returns a new Model value with the Type value set to v.
func (o Model) WithType(v interface{}) Model {
	o.Type = v
	return o
}

// WithIndexSet returns a new Model value with the IndexSet set to idxs.
func (o Model) WithIndexSet(idxs *IndexSet) Model {
	o.IndexSet = idxs
	return o
}

// WithMap returns a new Model value with the Map set to m.
//
// The m callback takes the model value v, with the type being Type,
// and returns the value to send in response to the get request.
func (o Model) WithMap(m func(interface{}) (interface{}, error)) Model {
	o.Map = m
	return o
}

// RebuildIndexes drops existing indexes and creates new entries for the
// models with the given resource pattern.
//
// The resource pattern should be the full pattern, including
// any service name. It may contain $tags, or end with a full wildcard (>).
// 	test.model.$id
// 	test.resource.>
func (o Model) RebuildIndexes(pattern string) error {
	// Quick exit in case no index exists
	if o.IndexSet == nil || len(o.IndexSet.Indexes) == 0 {
		return nil
	}

	p := res.Pattern(pattern)
	if !p.IsValid() {
		return errors.New("invalid pattern")
	}

	// Drop existing index entries
	for _, idx := range o.IndexSet.Indexes {
		err := o.BadgerDB.DB.DropPrefix([]byte(idx.Name))
		if err != nil {
			return err
		}
	}

	t := reflect.TypeOf(o.Type)

	// Create a prefix to seek from
	ridPrefix := pattern
	i := p.IndexWildcard()
	if i >= 0 {
		ridPrefix = pattern[:i]
	}

	// Create new index entries in a single transaction
	return o.BadgerDB.DB.Update(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := []byte(ridPrefix)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			// Ensure the key matches the pattern
			if !p.Matches(string(it.Item().Key())) {
				continue
			}
			// Load item and unmarshal it
			item := it.Item()
			v := reflect.New(t)
			err := item.Value(func(dta []byte) error {
				return json.Unmarshal(dta, v.Interface())
			})
			if err != nil {
				return err
			}
			// Loop through indexes and generate a new entry per index
			for _, idx := range o.IndexSet.Indexes {
				rname := item.KeyCopy(nil)
				idxKey := idx.getKey(rname, idx.Key(v.Elem().Interface()))
				err = txn.SetEntry(&badger.Entry{Key: idxKey, Value: nil, UserMeta: typeIndex})
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
}

// SetOption sets the res handler options,
// and implements the res.Option interface.
func (o Model) SetOption(hs *res.Handler) {
	var err error

	if o.BadgerDB.DB == nil {
		panic("middleware: no badger DB set")
	}

	b := resourceHandler{
		def:      o.Default,
		idxs:     o.IndexSet,
		m:        o.Map,
		BadgerDB: o.BadgerDB,
	}

	if o.Type != nil {
		b.t = reflect.TypeOf(o.Type)
	} else {
		b.t = reflect.TypeOf(map[string]interface{}(nil))
	}

	if b.def != nil {
		if !b.t.AssignableTo(reflect.TypeOf(b.def)) {
			panic("resbadger: default value not assignable to Type")
		}
		b.rawDefault, err = json.Marshal(b.def)
		if err != nil {
			panic(err)
		}
	}

	res.Model.SetOption(hs)
	res.GetResource(b.getResource).SetOption(hs)
	res.ApplyChange(b.applyChange).SetOption(hs)
	res.ApplyCreate(b.applyCreate).SetOption(hs)
	res.ApplyDelete(b.applyDelete).SetOption(hs)
}
