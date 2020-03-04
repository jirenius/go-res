package resbadger

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"

	"github.com/dgraph-io/badger"
	res "github.com/jirenius/go-res"
)

type resourceHandler struct {
	def        interface{}
	rawDefault json.RawMessage
	t          reflect.Type
	idxs       *IndexSet
	m          func(interface{}) (interface{}, error)
	BadgerDB
}

var (
	errUnknownType           = res.InternalError(errors.New("unknown type"))
	errIndexOutOfRange       = res.InternalError(errors.New("index out of range"))
	errResourceAlreadyExists = res.InternalError(errors.New("resource already exists"))
)

// typeIndex is the meta data value given to index entries
var typeIndex byte = 127

func (b *resourceHandler) getResource(r res.GetRequest) {
	var dta []byte

	err := b.DB.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte([]byte(r.ResourceName())))
		if err != nil {
			return err
		}
		dta, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		// Handle key not found
		if err == badger.ErrKeyNotFound {
			if b.def != nil {
				switch r.ResourceType() {
				case res.TypeModel:
					if r.ForValue() {
						r.Model(b.def)
					} else {
						r.Model(b.rawDefault)
					}
				case res.TypeCollection:
					if r.ForValue() {
						r.Collection(b.def)
					} else {
						r.Collection(b.rawDefault)
					}
				default:
					r.Error(errUnknownType)
				}
				return
			}

			r.NotFound()
			return
		}
		r.Error(res.ToError(err))
		return
	}

	var resource interface{}
	if r.ForValue() || b.m != nil {
		v := reflect.New(b.t)
		err = json.Unmarshal(dta, v.Interface())
		if err != nil {
			r.Error(res.ToError(err))
		}
		resource = v.Elem().Interface()
	}

	if !r.ForValue() {
		if b.m != nil {
			resource, err = b.m(resource)
			if err != nil {
				r.Error(res.ToError(err))
				return
			}
		} else {
			resource = json.RawMessage(dta)
		}
	}

	switch r.ResourceType() {
	case res.TypeModel:
		r.Model(resource)
	case res.TypeCollection:
		r.Collection(resource)
	default:
		r.Error(errUnknownType)
	}
}

func (b *resourceHandler) applyChange(r res.Resource, changes map[string]interface{}) (map[string]interface{}, error) {
	typ := r.ResourceType()
	if typ != res.TypeModel {
		return nil, errors.New("change event on non-model resource")
	}
	var rev map[string]interface{}
	var beforeValue, afterValue interface{}
	var updatedIdxs []string
	if b.idxs != nil {
		updatedIdxs = make([]string, 0, len(b.idxs.Indexes))
	}

	err := b.DB.Update(func(txn *badger.Txn) error {
		var m map[string]interface{}
		var dta []byte

		rname := []byte(r.ResourceName())

		item, err := txn.Get(rname)
		if err != nil {
			if err != badger.ErrKeyNotFound {
				return err
			}
			if b.rawDefault == nil {
				return res.ErrNotFound
			}
			dta = b.rawDefault
		} else {
			dta, err = item.ValueCopy(nil)
			if err != nil {
				return err
			}
		}
		err = json.Unmarshal(dta, &m)
		if err != nil {
			return err
		}

		rev = make(map[string]interface{}, len(changes))
		for k, v := range changes {
			ov, ok := m[k]
			if !ok {
				if v != res.DeleteAction {
					m[k] = v
					rev[k] = res.DeleteAction
				}
			} else if v == res.DeleteAction {
				delete(m, k)
				rev[k] = ov
			} else if !reflect.DeepEqual(v, ov) {
				m[k] = v
				rev[k] = ov
			}
		}

		// Exit in case of no actual changes
		if len(rev) == 0 {
			return nil
		}

		ndta, err := json.Marshal(m)
		if err != nil {
			return err
		}

		err = txn.Set(rname, ndta)
		if err != nil {
			return err
		}

		if b.idxs != nil {
			// Get the "before change value"
			v := reflect.New(b.t)
			err = json.Unmarshal(dta, v.Interface())
			if err != nil {
				return err
			}
			beforeValue = v.Elem().Interface()

			// Get the "after change value"
			v = reflect.New(b.t)
			err = json.Unmarshal(ndta, v.Interface())
			if err != nil {
				return err
			}
			afterValue = v.Elem().Interface()

			// Update index entries
			for _, idx := range b.idxs.Indexes {
				beforeKey := idx.Key(beforeValue)
				afterKey := idx.Key(afterValue)

				// Do nothing if key hasn't change; before and after is equal
				if bytes.Equal(beforeKey, afterKey) {
					continue
				}

				// Delete old index entry
				if len(beforeKey) > 0 {
					// [TODO] Log warning of failing to delete index entry
					_ = txn.Delete(idx.getKey(rname, beforeKey))
				}
				// Set new index entry
				if len(afterKey) > 0 {
					// [TODO] Log warning of failing to create index entry
					_ = txn.Set(idx.getKey(rname, afterKey), nil)
				}
				updatedIdxs = append(updatedIdxs, idx.Name)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(updatedIdxs) > 0 {
		for _, idxname := range updatedIdxs {
			b.idxs.triggerListeners(idxname, r, beforeValue, afterValue)
		}
		b.idxs.triggerListeners("", r, beforeValue, afterValue)
	}

	return rev, nil
}

func (b *resourceHandler) applyAdd(r res.Resource, value interface{}, idx int) error {
	typ := r.ResourceType()
	if typ != res.TypeCollection {
		return errors.New("add event on non-collection resource")
	}

	return b.DB.Update(func(txn *badger.Txn) error {
		var c []json.RawMessage
		var dta []byte
		rname := []byte(r.ResourceName())

		item, err := txn.Get(rname)
		if err != nil {
			if err != badger.ErrKeyNotFound {
				return err
			}
			if b.rawDefault == nil {
				// Default to an empty collection on add
				dta = []byte(`[]`)
			} else {
				dta = b.rawDefault
			}
		} else {
			dta, err = item.ValueCopy(nil)
			if err != nil {
				return err
			}
		}

		err = json.Unmarshal(dta, &c)
		if err != nil {
			return err
		}

		if len(c) < idx {
			return errIndexOutOfRange
		}

		// Add value to collection
		dta, err = json.Marshal(value)
		if err != nil {
			return err
		}
		c = append(c, nil)
		copy(c[idx+1:], c[idx:])
		c[idx] = json.RawMessage(dta)

		// Marshal new collection
		dta, err = json.Marshal(c)
		if err != nil {
			return err
		}

		err = txn.Set(rname, dta)
		if err != nil {
			return err
		}

		return nil
	})
}

func (b *resourceHandler) applyRemove(r res.Resource, idx int) (interface{}, error) {
	typ := r.ResourceType()
	if typ != res.TypeCollection {
		return nil, errors.New("remove event on non-collection resource")
	}

	err := b.DB.Update(func(txn *badger.Txn) error {
		var c []interface{}
		var dta []byte
		rname := []byte(r.ResourceName())

		item, err := txn.Get(rname)
		if err != nil {
			if err != badger.ErrKeyNotFound {
				return err
			}
			if b.rawDefault == nil {
				return res.ErrNotFound
			}
			dta = b.rawDefault
		} else {
			dta, err = item.ValueCopy(nil)
			if err != nil {
				return err
			}
		}
		err = json.Unmarshal(dta, &c)
		if err != nil {
			return err
		}

		if len(c) <= idx {
			return errIndexOutOfRange
		}

		copy(c[idx:], c[idx+1:])
		c[len(c)-1] = nil
		c = c[:len(c)-1]

		dta, err = json.Marshal(c)
		if err != nil {
			return err
		}

		err = txn.Set(rname, dta)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (b *resourceHandler) applyCreate(r res.Resource, value interface{}) error {
	typ := r.ResourceType()
	if typ == res.TypeUnset {
		return errors.New("create event on unset resource type")
	}
	var updatedIdxs []string
	if b.idxs != nil {
		updatedIdxs = make([]string, 0, len(b.idxs.Indexes))
	}

	err := b.DB.Update(func(txn *badger.Txn) error {
		rname := []byte(r.ResourceName())

		// Check that resource doesn't already exist
		_, err := txn.Get(rname)
		if err == nil {
			return errResourceAlreadyExists
		}
		if err != badger.ErrKeyNotFound {
			return err
		}

		if b.rawDefault != nil {
			return errResourceAlreadyExists
		}

		dta, err := json.Marshal(value)
		if err != nil {
			return err
		}

		err = txn.Set(rname, dta)
		if err != nil {
			return err
		}

		// Set index entry
		if b.idxs != nil {
			// [TODO] Check if value's type is the same as b.t. If not, use reflection to marshal into type b.t.
			for _, idx := range b.idxs.Indexes {
				iv := idx.Key(value)
				// Ignore nil keys
				if iv != nil {
					// [TODO] Log warning of failing to create index
					_ = txn.Set(idx.getKey(rname, iv), nil)
					updatedIdxs = append(updatedIdxs, idx.Name)
				}
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Call index listeners
	if len(updatedIdxs) > 0 {
		for _, idxname := range updatedIdxs {
			b.idxs.triggerListeners(idxname, r, nil, value)
		}
		b.idxs.triggerListeners("", r, nil, value)
	}

	return nil
}

func (b *resourceHandler) applyDelete(r res.Resource) (interface{}, error) {
	var dta []byte
	var value interface{}
	var updatedIdxs []string
	if b.idxs != nil {
		updatedIdxs = make([]string, 0, len(b.idxs.Indexes))
	}

	err := b.DB.Update(func(txn *badger.Txn) error {
		rname := []byte(r.ResourceName())

		// Check that the resource exists
		item, err := txn.Get(rname)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return nil
			}
			return err
		}
		dta, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}

		err = txn.Delete(rname)
		if err != nil {
			return err
		}

		// Delete index values
		if b.idxs != nil {
			// With indices, we need to get the value to the correct type first
			// so that our Index.Key callback can generate the key to delete
			v := reflect.New(b.t)
			err = json.Unmarshal(dta, v.Interface())
			if err != nil {
				return err
			}
			value = v.Elem().Interface()

			// Delete index entry
			for _, idx := range b.idxs.Indexes {
				iv := idx.Key(value)
				// Ignore nil keys
				if iv != nil {
					// [TODO] Log warning of failing to delete index
					_ = txn.Delete(idx.getKey(rname, iv))
					updatedIdxs = append(updatedIdxs, idx.Name)
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// If we have index, call index listeners
	if b.idxs != nil {
		for _, idxname := range updatedIdxs {
			b.idxs.triggerListeners(idxname, r, value, nil)
		}
		b.idxs.triggerListeners("", r, value, nil)
	} else {
		// If not, we need to unmarshal the data
		// and get a proper value
		v := reflect.New(b.t)
		err = json.Unmarshal(dta, v.Interface())
		if err != nil {
			return nil, err
		}
		value = v.Elem().Interface()
	}

	return value, nil
}
