/*
This is the Book Collection example where all changes are persisted using a
badgerDB store.

* It exposes a collection, `library.books`, containing book model references.
* It exposes book models, `library.book.<BOOK_ID>`, of each book.
* The books are persisted in a badgerDB store under `./db`.
* The store.Handler handles get requests by loading resources from the store.
* The store.QueryHandler handles get requests by getting a list of stored books.
* Changed made to the store bubbles up as events.
* It serves a web client at http://localhost:8084
*/
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/dgraph-io/badger"

	"github.com/jirenius/go-res"
)

// Book represents a book model.
type Book struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
}

func main() {
	// Create badger DB
	db, err := badger.Open(badger.DefaultOptions("./db").WithTruncate(true))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create badgerDB store for books
	bookStore := NewBookStore(db)
	// Create some initial books, if not done before
	bookStore.Init()

	// Create a new RES Service
	s := res.NewService("library")

	// Add handler for "library.book.$id" models
	s.Handle("book.$id",
		&BookHandler{BookStore: bookStore},
		res.Access(res.AccessGranted))
	// Add handler for "library.books" collection
	s.Handle("books",
		&BooksHandler{BookStore: bookStore},
		res.Access(res.AccessGranted))

	// Run a simple webserver to serve the client.
	// This is only for the purpose of making the example easier to run.
	go func() { log.Fatal(http.ListenAndServe(":8084", http.FileServer(http.Dir("wwwroot/")))) }()
	fmt.Println("Client at: http://localhost:8084/")

	s.ListenAndServe("nats://localhost:4222")
}
