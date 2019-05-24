package middleware

import (
	"encoding/json"
	"errors"
	"reflect"

	"github.com/dgraph-io/badger"

	res "github.com/jirenius/go-res"
)

type badgerDB struct {
	rawDefault json.RawMessage
	t          reflect.Type
	BadgerDBOption
}

type BadgerDBOption struct {
	DB      *badger.DB
	Default interface{}
	Type    interface{}
}

var (
	errUnknownType           = res.InternalError(errors.New("unknown type"))
	errMismatchingType       = res.InternalError(errors.New("mismatching resource type"))
	errIndexOutOfRange       = res.InternalError(errors.New("index out of range"))
	errResourceAlreadyExists = res.InternalError(errors.New("resource already exists"))
)

func BadgerDB(db *badger.DB) BadgerDBOption {
	return BadgerDBOption{DB: db}
}

func (o BadgerDBOption) SetDefault(i interface{}) BadgerDBOption {
	o.Default = i
	return o
}

func (o BadgerDBOption) SetType(v interface{}) BadgerDBOption {
	o.Type = v
	return o
}

func (o BadgerDBOption) SetDB(db *badger.DB) BadgerDBOption {
	o.DB = db
	return o
}

func (o BadgerDBOption) SetOption(hs *res.Handler) {
	var err error

	if hs.Type == res.TypeUnset {
		panic("middleware: no resource Type set for handler prior to setting BadgerDB middleware")
	}

	b := badgerDB{
		BadgerDBOption: o,
	}

	if o.Type != nil {
		b.t = reflect.TypeOf(o.Type)
	} else {
		// Set default type
		switch hs.Type {
		case res.TypeModel:
			b.t = reflect.TypeOf(map[string]interface{}(nil))
		case res.TypeCollection:
			b.t = reflect.TypeOf([]interface{}(nil))
		default:
			panic(errUnknownType)
		}
	}

	if o.Default != nil {
		if !b.t.AssignableTo(reflect.TypeOf(o.Default)) {
			panic("middleware: Default value not assignable to Type")
		}
		b.rawDefault, err = json.Marshal(o.Default)
		if err != nil {
			panic(err)
		}
	}

	res.GetResource(b.getResource).SetOption(hs)
	res.ApplyChange(b.applyChange).SetOption(hs)
	res.ApplyAdd(b.applyAdd).SetOption(hs)
	res.ApplyRemove(b.applyRemove).SetOption(hs)
	res.ApplyCreate(b.applyCreate).SetOption(hs)
	res.ApplyDelete(b.applyDelete).SetOption(hs)
}

func (b *badgerDB) getResource(r res.GetRequest) {
	var dta []byte
	var typ res.ResourceType

	err := b.DB.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte([]byte(r.ResourceName())))
		if err != nil {
			return err
		}
		dta, err = item.ValueCopy(nil)
		if err != nil {
			return err
		}
		typ = res.ResourceType(item.UserMeta())
		return nil
	})
	if err != nil {
		// Handle key not found
		if err == badger.ErrKeyNotFound {
			if b.Default != nil {
				switch r.ResourceType() {
				case res.TypeModel:
					if r.ForValue() {
						r.Model(b.Default)
					} else {
						r.Model(b.rawDefault)
					}
				case res.TypeCollection:
					if r.ForValue() {
						r.Collection(b.Default)
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

	if typ != r.ResourceType() {
		r.Error(errMismatchingType)
		return
	}

	var resource interface{}
	if r.ForValue() {
		v := reflect.New(b.t)
		err = json.Unmarshal(dta, v.Interface())
		if err != nil {
			r.Error(res.ToError(err))
		}
		resource = v.Elem().Interface()
	} else {
		resource = json.RawMessage(dta)
	}

	switch typ {
	case res.TypeModel:
		r.Model(resource)
	case res.TypeCollection:
		r.Collection(resource)
	default:
		r.Error(errUnknownType)
	}
}

func (b *badgerDB) applyChange(r res.Resource, changes map[string]interface{}) (map[string]interface{}, error) {
	typ := r.ResourceType()
	if typ != res.TypeModel {
		return nil, errors.New("change event on non-model resource")
	}
	var rev map[string]interface{}

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

		dta, err = json.Marshal(m)
		if err != nil {
			return err
		}

		err = txn.SetEntry(&badger.Entry{Key: rname, Value: dta, UserMeta: byte(res.TypeModel)})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return rev, nil
}

func (b *badgerDB) applyAdd(r res.Resource, value interface{}, idx int) error {
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

		if len(c) < idx {
			return errIndexOutOfRange
		}

		// Add value to collection
		dta, err = json.Marshal(value)
		c = append(c, nil)
		copy(c[idx+1:], c[idx:])
		c[idx] = json.RawMessage(dta)

		// Marshal new collection
		dta, err = json.Marshal(c)
		if err != nil {
			return err
		}

		err = txn.SetEntry(&badger.Entry{Key: rname, Value: dta, UserMeta: byte(res.TypeCollection)})
		if err != nil {
			return err
		}

		return nil
	})
}

func (b *badgerDB) applyRemove(r res.Resource, idx int) (interface{}, error) {
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

		err = txn.SetEntry(&badger.Entry{Key: rname, Value: dta, UserMeta: byte(res.TypeCollection)})
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

func (b *badgerDB) applyCreate(r res.Resource, value interface{}) error {
	typ := r.ResourceType()
	if typ == res.TypeUnset {
		return errors.New("create event on unset resource type")
	}

	return b.DB.Update(func(txn *badger.Txn) error {
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

		err = txn.SetEntry(&badger.Entry{Key: rname, Value: dta, UserMeta: byte(typ)})
		if err != nil {
			return err
		}

		return nil
	})
}

func (b *badgerDB) applyDelete(r res.Resource) (interface{}, error) {
	var dta []byte

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

		return nil
	})
	if err != nil {
		println("From 1", err)
		return nil, err
	}

	v := reflect.New(b.t)
	err = json.Unmarshal(dta, v.Interface())
	if err != nil {
		// Somehow we failed to unmarshal the data.
		// Instead of returning error, which will cancel the delete event,
		// we return the raw JSON data.
		// This might cause panic in any OnDelete handlers, when trying to type assert
		// the value. But then the delete event is at least propagated properly.

		println("From 2")
		return json.RawMessage(dta), nil
	}
	return v.Elem().Interface(), nil
}
