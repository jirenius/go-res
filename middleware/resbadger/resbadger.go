package resbadger

import (
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
