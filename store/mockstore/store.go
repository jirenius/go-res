package mockstore

import (
	"errors"
	"sync"

	"github.com/jirenius/go-res/store"
)

// Store is an in-memory CRUD mock store implementation.
//
// It implements the store.Store interface.
//
// A Store must not be copied after first call to Read or Write.
type Store struct {
	// OnChangeCallbacks contains callbacks added with OnChange.
	OnChangeCallbacks []func(id string, before, after interface{})

	// RWMutex protects the Resources map.
	sync.RWMutex

	// Resources is a map of stored resources.
	Resources map[string]interface{}

	// NewID is a mock function returning an new ID when Create is called with
	// an empty ID. Default is that Create returns an error.
	NewID func() string

	// OnExists overrides the Exists call. Default behavior is to return true if
	// Resources contains the id.
	OnExists func(st *Store, id string) bool

	// OnValue overrides the Value call. Default behavior is to return the value
	// in Resources, or store.ErrNotFound if not found.
	OnValue func(st *Store, id string) (interface{}, error)

	// OnCreate overrides the Create call. Default behavior is to set Resources
	// with the value if the id does not exist, otherwise return a
	// store.ErrDuplicate error.
	OnCreate func(st *Store, id string, v interface{}) error

	// OnUpdate overrides the Update call. It should return the previous value,
	// or an error. Default behavior is to replace the Resources value if it
	// exists, or return store.ErrNotFound if not found.
	OnUpdate func(st *Store, id string, v interface{}) (interface{}, error)

	// OnDelete overrides the OnDelete call. It should return the deleted value,
	// or an error. Default behavior is to delete the Resources value if it
	// exists, or return store.ErrNotFound if not found.
	OnDelete func(st *Store, id string) (interface{}, error)
}

// Assert *Store implements the store.Store interface.
var _ store.Store = &Store{}

var errMissingID = errors.New("missing ID")

type readTxn struct {
	st     *Store
	id     string
	closed bool
}

type writeTxn struct {
	readTxn
}

// NewStore creates a new empty Store.
func NewStore() *Store {
	return &Store{}
}

// Add inserts a value into the Resources map.
func (st *Store) Add(id string, v interface{}) *Store {
	if st.Resources == nil {
		st.Resources = make(map[string]interface{}, 1)
	}
	st.Resources[id] = v
	return st
}

// Read makes a read-lock for the resource that lasts until Close is called.
func (st *Store) Read(id string) store.ReadTxn {
	st.RLock()
	return readTxn{st: st, id: id}
}

// Write makes a write-lock for the resource that lasts until Close is called.
func (st *Store) Write(id string) store.WriteTxn {
	st.Lock()
	return writeTxn{readTxn{st: st, id: id}}
}

// Close closes the read transaction.
func (rt readTxn) Close() error {
	if rt.closed {
		return errors.New("already closed")
	}
	rt.closed = true
	rt.st.RUnlock()
	return nil
}

// Close closes the write transaction.
func (wt writeTxn) Close() error {
	if wt.closed {
		return errors.New("already closed")
	}
	wt.closed = true
	wt.st.Unlock()
	return nil
}

// Exists returns true if the value exists in the store, or false in case or
// read error or value does not exist.
func (rt readTxn) Exists() bool {
	if rt.id == "" {
		return false
	}

	if rt.st.OnExists != nil {
		return rt.st.OnExists(rt.st, rt.id)
	}

	_, ok := rt.st.Resources[rt.id]
	return ok
}

// Value gets an existing value in the store.
//
// If the value does not exist, store.ErrNotFound is returned.
func (rt readTxn) Value() (interface{}, error) {
	if rt.id == "" {
		return nil, store.ErrNotFound
	}

	if rt.st.OnValue != nil {
		return rt.st.OnValue(rt.st, rt.id)
	}

	v, ok := rt.st.Resources[rt.id]
	if !ok {
		return nil, store.ErrNotFound
	}

	return v, nil
}

// ID returns the ID of the resource.
func (rt readTxn) ID() string {
	return rt.id
}

// Create adds a new value to the store.
//
// If a value already exists for the resource ID, id, an error is returned.
func (wt writeTxn) Create(v interface{}) error {
	if wt.id == "" {
		if wt.st.NewID == nil {
			return errMissingID
		}
		wt.id = wt.st.NewID()
		if wt.id == "" {
			panic("callback NewID returned empty string")
		}
	}

	var err error
	if wt.st.OnCreate != nil {
		err = wt.st.OnCreate(wt.st, wt.id, v)
	} else {
		_, ok := wt.st.Resources[wt.id]
		if ok {
			err = store.ErrDuplicate
		} else {
			if wt.st.Resources == nil {
				wt.st.Resources = make(map[string]interface{})
			}
			wt.st.Resources[wt.id] = v
		}
	}

	if err != nil {
		return err
	}

	wt.st.callOnChange(wt.id, nil, v)

	return nil
}

// Update overwrites an existing value in the store with a new value, v.
//
// If the value does not exist, res.ErrNotFound is returned.
func (wt writeTxn) Update(v interface{}) error {
	if wt.id == "" {
		return store.ErrNotFound
	}

	var err error
	var before interface{}
	var ok bool

	if wt.st.OnUpdate != nil {
		before, err = wt.st.OnUpdate(wt.st, wt.id, v)
	} else {
		before, ok = wt.st.Resources[wt.id]
		if !ok {
			err = store.ErrNotFound
		} else {
			wt.st.Resources[wt.id] = v
		}
	}

	if err != nil {
		return err
	}

	wt.st.callOnChange(wt.id, before, v)
	return nil
}

// Delete removes an existing value from the store.
//
// If the value does not exist, res.ErrNotFound is returned.
func (wt writeTxn) Delete() error {
	if wt.id == "" {
		return store.ErrNotFound
	}

	var err error
	var before interface{}
	var ok bool

	if wt.st.OnDelete != nil {
		before, err = wt.st.OnDelete(wt.st, wt.id)
	} else {
		before, ok = wt.st.Resources[wt.id]
		if !ok {
			err = store.ErrNotFound
		} else {
			delete(wt.st.Resources, wt.id)
		}
	}

	if err != nil {
		return err
	}

	wt.st.callOnChange(wt.id, before, nil)
	return nil
}

// OnChange adds a listener callback that is called whenever a value is created,
// updated, or deleted from the store.
//
// If a value is created, before will be set to nil.
//
// If a value is deleted, after will be set to nil.
func (st *Store) OnChange(cb func(id string, before, after interface{})) {
	st.OnChangeCallbacks = append(st.OnChangeCallbacks, cb)
}

// callOnChange loops through OnChange listeners and calls them.
func (st *Store) callOnChange(id string, before, after interface{}) {
	for _, cb := range st.OnChangeCallbacks {
		cb(id, before, after)
	}
}
