package main

import (
	"net/url"
	"strings"

	"github.com/dgraph-io/badger"
	"github.com/jirenius/go-res/store/badgerstore"
	"github.com/rs/xid"
)

// BookStore contains the store and query stores for the books. BadgerDB is used
// for storage, but any other database can be used. What is needed is a wrapper
// that implements the Store and QueryStore interfaces found in package:
//
// 	github.com/jirenius/go-res/store
type BookStore struct {
	*badgerstore.Store
	BooksByTitle *badgerstore.QueryStore
}

// A badgerstore db index by book title (lower case).
var idxBookTitle = badgerstore.Index{
	Name: "idxBook_title",
	Key: func(v interface{}) []byte {
		book := v.(Book)
		return []byte(strings.ToLower(book.Title))
	},
}

// NewBookStore creates a new BookStore.
func NewBookStore(db *badger.DB) *BookStore {
	st := badgerstore.NewStore(db).
		SetType(Book{}).
		SetPrefix("book")
	return &BookStore{
		Store: st,
		BooksByTitle: badgerstore.NewQueryStore(st, booksByTitleIndexQuery).
			AddIndex(idxBookTitle),
	}
}

// booksByTitleIndexQuery handles query requests. This method is badgerstore
// specific, and allows for simple index based queries towards the badgerDB
// store.
//
// Other database implementations for store.QueryStore would do it differently.
// A sql implementation might have you generate a proper WHERE statement, where
// as a mongoDB implementation would need a bson query document.
func booksByTitleIndexQuery(qs *badgerstore.QueryStore, q url.Values) (*badgerstore.IndexQuery, error) {
	// All query parameters are ignored. Just query all books without limit.
	return &badgerstore.IndexQuery{
		Index: idxBookTitle,
		Limit: -1,
	}, nil
}

// Init seeds an empty store with some initial books. It panics on errors.
func (st *BookStore) Init() {
	if err := st.Store.Init(func(add func(id string, v interface{})) error {
		for _, book := range []Book{
			{Title: "Animal Farm", Author: "George Orwell"},
			{Title: "Brave New World", Author: "Aldous Huxley"},
			{Title: "Coraline", Author: "Neil Gaiman"},
		} {
			book.ID = xid.New().String()
			add(book.ID, book)
		}
		return nil
	}); err != nil {
		panic(err)
	}
	// Wait for the badgerDB index to be created
	st.BooksByTitle.Flush()
}
