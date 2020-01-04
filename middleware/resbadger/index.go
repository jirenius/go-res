package resbadger

import (
	"fmt"

	res "github.com/jirenius/go-res"
)

// Index defines an index used for a resource.
//
// When used on Model resource, an index entry will be added for each model entry.
// An index entry will have no value (nil), and the key will have the following structure:
//    <Name>:<Key>?<RID>
// Where:
// * <Name> is the name of the Index (so keep it rather short)
// * <Key> is the index value as returned from the Key callback
// * <RID> is the resource ID of the indexed model
type Index struct {
	// Index name
	Name string
	// Key callback is called with a resource item of the type defined by Type,
	// and should return the string to use as index value.
	// It does not have to be unique.
	//
	// Example index by Country and lower case Name on a user model:
	// 	func(v interface{}) {
	// 		user := v.(UserModel)
	// 		return []byte(user.Country + "_" + strings.ToLower(user.Name))
	// 	}
	Key func(interface{}) []byte
}

// Indexes represents a set of indexes for a model resource.
type Indexes struct {
	// List of indices
	Indexes []Index
	// Index listener callbacks to be called on changes in the index.
	listeners []func(r res.Resource, before, after interface{})
}

const ridSeparator = byte(0)

// Listen adds a callback listening to the changes that have affected one or more index entries.
//
// The model before value will be nil if the model was created, or if previously not indexed.
// The model after value will be nil if the model was deleted, or if no longer indexed.
func (i *Indexes) Listen(cb func(r res.Resource, before, after interface{})) {
	i.listeners = append(i.listeners, cb)
}

// triggerListeners calls the callback of each registered listener.
func (i *Indexes) triggerListeners(r res.Resource, before, after interface{}) {
	for _, cb := range i.listeners {
		cb(r, before, after)
	}
}

// IndexQuery represents a query towards an index.
type IndexQuery struct {
	// Index used
	Index Index
	// KeyPrefix to match against the index key
	KeyPrefix []byte
	// FilterKeys for keys in the query collection. May be nil.
	FilterKeys func(key []byte) bool
	// Offset from which item to start.
	Offset int
	// Limit how many items to read. 0 means unlimited.
	Limit int
	// Normalized query
	NormalizedQuery string
}

// GetIndex returns an index by name, or an error if not found.
func (i *Indexes) GetIndex(name string) (Index, error) {
	for _, idx := range i.Indexes {
		if idx.Name == name {
			return idx, nil
		}
	}
	return Index{}, fmt.Errorf("index %s not found", name)
}

func (idx Index) getKey(rname []byte, value []byte) []byte {
	b := make([]byte, len(idx.Name)+len(value)+len(rname)+2)
	copy(b, idx.Name)
	offset := len(idx.Name)
	b[offset] = ':'
	offset++
	copy(b[offset:], value)
	offset += len(value)
	b[offset] = ridSeparator
	copy(b[offset+1:], rname)
	return b
}

func (idx Index) getQuery(keyPrefix []byte) []byte {
	b := make([]byte, len(idx.Name)+len(keyPrefix)+1)
	copy(b, idx.Name)
	offset := len(idx.Name)
	b[offset] = ':'
	offset++
	copy(b[offset:], keyPrefix)
	return b
}
