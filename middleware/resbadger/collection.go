package resbadger

import (
	"encoding/json"
	"reflect"

	res "github.com/jirenius/go-res"
)

// Collection represents a collection that is stored in the badger DB by its resource ID.
type Collection struct {
	// BadgerDB middleware
	BadgerDB BadgerDB
	// Default resource value if not found in database.
	// Will return res.ErrNotFound if not set.
	Default interface{}
	// Type used to marshal into when calling r.Value() or r.RequireValue().
	// Defaults to []interface{} if not set.
	Type interface{}
}

// WithDefault returns a new BadgerDB value with the Default resource value set to i.
func (o Collection) WithDefault(i interface{}) Collection {
	o.Default = i
	return o
}

// WithType returns a new Collection value with the Type value set to v.
func (o Collection) WithType(v interface{}) Collection {
	o.Type = v
	return o
}

// SetOption sets the res handler options,
// and implements the res.Option interface.
func (o Collection) SetOption(hs *res.Handler) {
	var err error

	if o.BadgerDB.DB == nil {
		panic("middleware: no badger DB set")
	}

	b := resourceHandler{
		def:      o.Default,
		BadgerDB: o.BadgerDB,
	}

	if o.Type != nil {
		b.t = reflect.TypeOf(o.Type)
	} else {
		b.t = reflect.TypeOf([]interface{}(nil))
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

	res.Collection.SetOption(hs)
	res.GetResource(b.getResource).SetOption(hs)
	res.ApplyAdd(b.applyAdd).SetOption(hs)
	res.ApplyRemove(b.applyRemove).SetOption(hs)
	res.ApplyCreate(b.applyCreate).SetOption(hs)
	res.ApplyDelete(b.applyDelete).SetOption(hs)
}
