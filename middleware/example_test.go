package middleware_test

import (
	"github.com/dgraph-io/badger"
	res "github.com/jirenius/go-res"
	"github.com/jirenius/go-res/middleware"
)

func ExampleBadgerDB() {
	db := &badger.DB{} // Dummy. Use badger.Open

	s := res.NewService("directory")
	s.Handle("user.$id",
		res.Model,
		middleware.BadgerDB{DB: db},
		/* ... */
	)
}

func ExampleBadgerDB_WithType() {
	db := &badger.DB{} // Dummy. Use badger.Open

	type User struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	s := res.NewService("directory")
	badgerDB := middleware.BadgerDB{DB: db}
	s.Handle("user.$id",
		res.Model,
		badgerDB.WithType(User{}),
		res.Set(func(r res.CallRequest) {
			_ = r.RequireValue().(User)
			/* ... */
			r.OK(nil)
		}),
	)
}

func ExampleBadgerDB_WithDefault() {
	db := &badger.DB{} // Dummy. Use badger.Open

	s := res.NewService("directory")
	badgerDB := middleware.BadgerDB{DB: db}
	s.Handle("users",
		res.Collection,
		// Default to an empty slice of references
		badgerDB.WithType([]res.Ref{}).WithDefault([]res.Ref{}),
		/* ... */
	)
}
