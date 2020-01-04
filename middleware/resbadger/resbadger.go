package resbadger

import (
	"encoding/json"
	"reflect"

	"github.com/dgraph-io/badger"
)

// BadgerDB provides persistence to BadgerDB for the res Handlers.
//
// It will set the GetResource and Apply* handlers to load, store, and update the resources
// in the database, using the resource ID as key value.
type BadgerDB struct {
	// BadgerDB database
	DB *badger.DB
}

type badgerDB struct {
	rawDefault json.RawMessage
	t          reflect.Type
	BadgerDB
}

// Model returns a middleware builder of type Model.
func (o BadgerDB) Model() Model {
	return Model{BadgerDB: o}
}

// Collection returns a middleware builder of type Collection.
func (o BadgerDB) Collection() Collection {
	return Collection{BadgerDB: o}
}

// QueryCollection returns a middleware builder of type QueryCollection.
func (o BadgerDB) QueryCollection() QueryCollection {
	return QueryCollection{BadgerDB: o}
}

// WithDB returns a new BadgerDB value with the DB set to db.
func (o BadgerDB) WithDB(db *badger.DB) BadgerDB {
	o.DB = db
	return o
}

// // WithIndexQueryCollection returns a new BadgerDB value with the IndexQueryCollection set to cb.
// func (o BadgerDB) WithIndexQueryCollection(cb func(idxs *Indexes, rname string, params map[string]string, q url.Values) (*IndexQuery, error)) BadgerDB {
// 	o.IndexQueryCollection = cb
// 	return o
// }
