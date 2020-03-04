package badgerstore

import (
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"

	"github.com/dgraph-io/badger"
	res "github.com/jirenius/go-res"
	"github.com/jirenius/go-res/store"
	"github.com/jirenius/keylock"
)

// Store is a database CRUD store implementation for BadgerDB.
//
// It implements the store.Store interface.
//
// A Store must not be copied after first call to Read or Write.
type Store struct {
	DB           *badger.DB
	typ          interface{}
	t            reflect.Type
	kl           keylock.KeyLock
	prefix       string
	useMarshal   bool
	beforeChange []func(id string, before, after interface{}) error
	onChange     []func(id string, before, after interface{})
}

var _ store.Store = &Store{}

type readTxn struct {
	st     *Store
	v      interface{}
	id     string
	rname  []byte
	closed bool
}

type writeTxn struct {
	readTxn
}

var interfaceMapType = reflect.TypeOf(map[string]interface{}(nil))

// NewStore creates a new Store and initializes it.
//
// The type of typ will be used as value. If the type supports both the
// encoding.BinaryMarshaler and the encoding.BinaryUnmarshaler, those method
// will be used for marshaling the values. Otherwise, encoding/json will be
// used for marshaling.
func NewStore(db *badger.DB) *Store {
	return &Store{
		DB: db,
	}
}

// SetPrefix sets the prefix that will be prepended to all resource ID's, using
// a dot (.) as separator between the prefix and the rest of the ID.
func (st *Store) SetPrefix(prefix string) *Store {
	if prefix == "" {
		st.prefix = ""
	} else {
		st.prefix = prefix + "."
	}
	return st
}

// SetType sets the type, typ, that will be used to unmarshal stored values
// into. If the type supports both the encoding.BinaryMarshaler and the
// encoding.BinaryUnmarshaler, those method will be used for marshaling the
// values. Otherwise, encoding/json will be used for marshaling.
func (st *Store) SetType(typ interface{}) *Store {
	t := reflect.TypeOf(typ)
	typ = reflect.New(t).Elem().Interface()
	_, bmi := typ.(encoding.BinaryMarshaler)
	_, bui := typ.(encoding.BinaryUnmarshaler)
	st.typ = typ
	st.t = reflect.TypeOf(typ)
	st.useMarshal = bmi && bui
	return st
}

// Type returns a zero-value of the type used by the store for unmarshaling
// values.
func (st *Store) Type() interface{} {
	if st.typ == nil {
		return map[string]interface{}(nil)
	}
	return st.typ
}

// Read makes a read-lock for the resource that lasts until Close is called.
func (st *Store) Read(id string) store.ReadTxn {
	st.kl.RLock(id)
	return readTxn{st: st, id: id, rname: []byte(st.prefix + id)}
}

// Write makes a write-lock for the resource that lasts until Close is called.
func (st *Store) Write(id string) store.WriteTxn {
	st.kl.Lock(id)
	return writeTxn{readTxn{st: st, id: id, rname: []byte(st.prefix + id)}}
}

// Close closes the read transaction.
func (rt readTxn) Close() error {
	if rt.closed {
		return errors.New("already closed")
	}
	rt.closed = true
	rt.st.kl.RUnlock(rt.id)
	return nil
}

// Close closes the write transaction.
func (wt writeTxn) Close() error {
	if wt.closed {
		return errors.New("already closed")
	}
	wt.closed = true
	wt.st.kl.Unlock(wt.id)
	return nil
}

// Exists returns true if the value exists in the store, or false in case or
// read error or value does not exist.
func (rt readTxn) Exists() bool {
	return rt.st.DB.View(func(txn *badger.Txn) error {
		_, err := rt.st.getValue(txn, rt.rname)
		return err
	}) == nil
}

// Value gets an existing value in the database.
//
// If the value does not exist, res.ErrNotFound is returned.
func (rt readTxn) Value() (interface{}, error) {
	if rt.v != nil {
		return rt.v, nil
	}
	var v interface{}
	err := rt.st.DB.View(func(txn *badger.Txn) error {
		var err error
		v, err = rt.st.getValue(txn, rt.rname)
		return err
	})
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, res.ErrNotFound
		}
		return nil, err
	}
	rt.v = v
	return v, nil
}

// ID returns the ID of the resource.
func (rt readTxn) ID() string {
	return rt.id
}

// Create adds a new value to the database.
//
// If a value already exists for the resource ID, id, an error is returned.
func (wt writeTxn) Create(v interface{}) error {
	vv := reflect.ValueOf(v)
	t := wt.st.t
	if t == nil {
		t = interfaceMapType
	}
	if vv.Type() != t {
		return fmt.Errorf("create value is of type %s, expected type %s", vv.Type().String(), t.String())
	}

	err := wt.st.DB.Update(func(txn *badger.Txn) error {
		// Validate that the resource doesn't exist
		_, err := txn.Get(wt.rname)
		if err == nil {
			return fmt.Errorf("cannot create because value for %s already exists", wt.id)
		}
		if err != badger.ErrKeyNotFound {
			return err
		}

		// Call beforeChange listeners and cancel if error occurred.
		err = wt.st.callBeforeChange(wt.id, nil, v)
		if err != nil {
			return err
		}

		// Marshal the value and store it in the database
		return wt.st.setValue(txn, wt.rname, v)
	})
	if err != nil {
		return err
	}

	wt.st.callOnChange(wt.id, nil, v)
	return nil
}

// Update overwrites an existing value in the database with a new value, v.
//
// If the value does not exist, res.ErrNotFound is returned.
func (wt writeTxn) Update(v interface{}) error {
	vv := reflect.ValueOf(v)
	t := wt.st.t
	if t == nil {
		t = interfaceMapType
	}
	if vv.Type() != t {
		return fmt.Errorf("update value is of type %s, expected type %s", vv.Type().String(), t.String())
	}
	var before interface{}
	err := wt.st.DB.Update(func(txn *badger.Txn) error {
		var err error
		// Get before value
		if wt.v != nil {
			before = wt.v
		} else {
			if before, err = wt.st.getValue(txn, wt.rname); err != nil {
				return err
			}
			wt.v = before
		}

		// Call beforeChange listeners and cancel if error occurred.
		err = wt.st.callBeforeChange(wt.id, before, v)
		if err != nil {
			return err
		}

		// Marshal new value and update
		return wt.st.setValue(txn, wt.rname, v)
	})
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return res.ErrNotFound
		}
		return err
	}

	wt.st.callOnChange(wt.id, before, v)
	return nil
}

// Delete removes an existing value from the database.
//
// If the value does not exist, res.ErrNotFound is returned.
func (wt writeTxn) Delete() error {
	var before interface{}
	err := wt.st.DB.Update(func(txn *badger.Txn) error {
		var err error

		// Get before value
		if wt.v != nil {
			before = wt.v
		} else {
			if before, err = wt.st.getValue(txn, wt.rname); err != nil {
				return err
			}
			wt.v = before
		}

		// Call beforeChange listeners and cancel if error occurred.
		err = wt.st.callBeforeChange(wt.id, before, nil)
		if err != nil {
			return err
		}

		// Delete value
		err = txn.Delete(wt.rname)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return res.ErrNotFound
		}
		return err
	}

	wt.st.callOnChange(wt.id, before, nil)
	return nil
}

// OnChange adds a listener callback that is called whenever a value is created,
// updated, or deleted from the database.
//
// If a value is created, before will be set to nil.
//
// If a value is deleted, after will be set to nil.
func (st *Store) OnChange(cb func(id string, before, after interface{})) {
	st.onChange = append(st.onChange, cb)
}

// Init adds initial resources for the store. If the store has been previously
// initialized, no resources will be added. It uses a key, "$<prefix>.init"
// (where <prefix> is the set prefix), to mark the store as initialized.
func (st *Store) Init(cb func(add func(id string, v interface{})) error) error {
	created := make(map[string]interface{})
	return st.DB.Update(func(txn *badger.Txn) error {
		var err error
		initKey := []byte(`$` + st.prefix + `init`)
		// Check init flag key
		_, err = txn.Get(initKey)
		if err != badger.ErrKeyNotFound {
			return err
		}

		// Load entries
		entries := make(map[string]interface{})
		t := st.t
		if t == nil {
			t = interfaceMapType
		}
		var adderr error
		add := func(id string, v interface{}) {
			// Quick exit if we have encountered an error previously
			if adderr != nil {
				return
			}
			// Validation
			vv := reflect.ValueOf(v)
			if vv.Type() != t {
				adderr = fmt.Errorf("init value is of type %s, expected type %s", vv.Type().String(), t.String())
				return
			}
			if id == "" {
				adderr = errors.New("empty ID string")
				return
			}
			if _, ok := entries[id]; ok {
				adderr = fmt.Errorf("duplicate id: %s", id)
				return
			}
			entries[id] = v
		}
		// Call init callback with the add method
		if err := cb(add); err != nil {
			return err
		}
		if adderr != nil {
			return adderr
		}

		// Write resources
		for id, v := range entries {
			rname := []byte(st.prefix + id)
			// Skip values that already exists.
			_, err := txn.Get(rname)
			if err == nil {
				continue
			}
			if err != badger.ErrKeyNotFound {
				return err
			}
			if err := st.setValue(txn, rname, v); err != nil {
				return err
			}
			created[id] = v
		}

		// Call OnChange callback
		for id, v := range created {
			st.callOnChange(id, nil, v)
		}

		// Set init flag key
		return txn.Set(initKey, nil)
	})
}

// getValue gets a value from the database and unmarshals it.
func (st *Store) getValue(txn *badger.Txn, key []byte) (interface{}, error) {
	item, err := txn.Get(key)
	if err != nil {
		if err == badger.ErrKeyNotFound {
			return nil, res.ErrNotFound
		}
		return nil, err
	}
	var v interface{}
	if err = item.Value(func(dta []byte) error {
		t := st.t
		if t == nil {
			t = interfaceMapType
		}
		tv := reflect.New(t)
		if st.useMarshal {
			v = tv.Elem().Interface()
			err := v.(encoding.BinaryUnmarshaler).UnmarshalBinary(dta)
			if err != nil {
				return err
			}
		} else {
			err := json.Unmarshal(dta, tv.Interface())
			if err != nil {
				return err
			}
			v = tv.Elem().Interface()
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return v, nil
}

// BeforeChange adds a listener callback that is called before a value is
// created, updated, or deleted from the database.
//
// If the callback returns an error, the change will be canceled.
func (st *Store) BeforeChange(cb func(id string, before, after interface{}) error) {
	st.beforeChange = append(st.beforeChange, cb)
}

// setValue marshals a value and updates the database.
func (st *Store) setValue(txn *badger.Txn, key []byte, v interface{}) error {
	var err error
	var dta []byte
	if st.useMarshal {
		dta, err = v.(encoding.BinaryMarshaler).MarshalBinary()
	} else {
		dta, err = json.Marshal(v)
	}
	if err != nil {
		return err
	}
	return txn.Set(key, dta)
}

// callOnChange loops through OnChange listeners and calls them.
func (st *Store) callOnChange(id string, before, after interface{}) {
	for _, cb := range st.onChange {
		cb(id, before, after)
	}
}

// callBeforeChange loops through BeforeChange listeners and calls them.
func (st *Store) callBeforeChange(id string, before, after interface{}) error {
	for _, cb := range st.beforeChange {
		err := cb(id, before, after)
		if err != nil {
			return err
		}
	}
	return nil
}
