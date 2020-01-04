/*
This is the Book Collection example where all changes are persisted using the BadgerDB middleware. By using the BadgerDB middleware, both clients and database can be updated with a single event.
* It exposes a collection, `library.books`, containing book model references.
* It exposes book models, `library.book.<BOOK_ID>`, of each book.
* The middleware adds a GetResource handler that loads the resources from the database.
* The middleware adds a ApplyChange handler that updates the books on change events.
* The middleware adds a ApplyAdd handler that updates the list on add events.
* The middleware adds a ApplyRemove handler that updates the list on remove events.
* The middleware adds a ApplyCreate handler that stores new books on create events.
* The middleware adds a ApplyDelete handler that deletes books on delete events.
* It persists all changes to a local BadgerDB database under `./db`.
* It serves a web client at http://localhost:8085
*/
package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/dgraph-io/badger"
	"github.com/jirenius/go-res/middleware/resbadger"

	"github.com/jirenius/go-res"
	"github.com/rs/xid"
)

// Book represents a book model
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

	// Create badgerDB middleware for res.Service
	badgerDB := resbadger.BadgerDB{DB: db}

	// Create a new RES Service
	s := res.NewService("library")

	// Add handlers for "library.book.$id" models
	s.Handle(
		"book.$id",
		res.Access(res.AccessGranted),
		badgerDB.
			Model().
			WithType(Book{}),
		res.Set(setBookHandler),
	)

	// Add handlers for "library.books" collection
	s.Handle(
		"books",
		res.Access(res.AccessGranted),
		badgerDB.
			Collection().
			WithType([]res.Ref{}),
		res.Call("new", newBookHandler),
		res.Call("delete", deleteBookHandler),
	)

	// Set on serve handler to bootstrap the data, if needed
	s.SetOnServe(onServe)

	// Run a simple webserver to serve the client.
	// This is only for the purpose of making the example easier to run.
	go func() { log.Fatal(http.ListenAndServe(":8085", http.FileServer(http.Dir("wwwroot/")))) }()
	fmt.Println("Client at: http://localhost:8085/")

	s.ListenAndServe("nats://localhost:4222")
}

func setBookHandler(r res.CallRequest) {
	book := r.RequireValue().(Book)

	// Unmarshal parameters to an anonymous struct
	var p struct {
		Title  *string `json:"title,omitempty"`
		Author *string `json:"author,omitempty"`
	}
	r.ParseParams(&p)

	// Validate title param
	if p.Title != nil {
		*p.Title = strings.TrimSpace(*p.Title)
		if *p.Title == "" {
			r.InvalidParams("Title must not be empty")
			return
		}
	}

	// Validate author param
	if p.Author != nil {
		*p.Author = strings.TrimSpace(*p.Author)
		if *p.Author == "" {
			r.InvalidParams("Author must not be empty")
			return
		}
	}

	changed := make(map[string]interface{}, 2)
	// Check if the title property was changed
	if p.Title != nil && *p.Title != book.Title {
		changed["title"] = *p.Title
	}
	// Check if the author property was changed
	if p.Author != nil && *p.Author != book.Author {
		changed["author"] = *p.Author
	}

	// Send a change event with updated fields.
	// BadgerDB middleware will use the event to updated the stored model
	r.ChangeEvent(changed)

	// Send success response
	r.OK(nil)
}

func newBookHandler(r res.CallRequest) {
	books := r.RequireValue().([]res.Ref)

	var p struct {
		Title  string `json:"title"`
		Author string `json:"author"`
	}
	r.ParseParams(&p)

	// Trim whitespace
	title := strings.TrimSpace(p.Title)
	author := strings.TrimSpace(p.Author)

	// Check if we received both title and author
	if title == "" || author == "" {
		r.InvalidParams("Must provide both title and author")
		return
	}

	// Create a new book model
	book := &Book{ID: xid.New().String(), Title: title, Author: author}
	rid := "library.book." + book.ID

	// Send a create event with the new book resource
	if err := r.Service().With(rid, func(r res.Resource) {
		// BadgerDB middleware will use the event to store the new book
		r.CreateEvent(book)
	}); err != nil {
		panic(err)
	}

	// Convert resource ID to a resource reference
	ref := res.Ref(rid)
	// Add book at the bottom of the list
	// BadgerDB middleware will use the event to updated the stored list
	r.AddEvent(ref, len(books))

	// Respond with a reference to the newly created book model
	r.Resource(rid)
}

func deleteBookHandler(r res.CallRequest) {
	books := r.RequireValue().([]res.Ref)

	// Unmarshal parameters to an anonymous struct
	var p struct {
		ID string `json:"id,omitempty"`
	}
	r.ParseParams(&p)

	rname := "library.book." + p.ID

	// Find the book in books collection, and remove it
	for i, rid := range books {
		if rid == res.Ref(rname) {
			// Send remove event
			r.RemoveEvent(i)

			// Run with book resource
			if err := r.Service().With(rname, func(r res.Resource) {
				// Send book delete event.
				// BadgerDB middleware will use the event to delete the stored book
				r.DeleteEvent()
			}); err != nil {
				panic(err)
			}

			break
		}
	}

	// Send success response. It is up to the service to define if a delete
	// should be idempotent or not. In this case we send success regardless
	// if the book existed or not, making it idempotent.
	r.OK(nil)
}

// onServe bootstraps an empty database with some initial books.
func onServe(s *res.Service) {
	s.With("library.books", func(r res.Resource) {
		// Exit if library books already exists
		_, err := r.Value()
		if err != res.ErrNotFound {
			return
		}

		// Book models to bootstrap with
		books := []*Book{
			{ID: xid.New().String(), Title: "Animal Farm", Author: "George Orwell"},
			{ID: xid.New().String(), Title: "Brave New World", Author: "Aldous Huxley"},
			{ID: xid.New().String(), Title: "Coraline", Author: "Neil Gaiman"},
		}

		// Loop through the books and send appropriate events,
		// which BadgerDB middleware will persist.
		for i, book := range books {
			rid := "library.book." + book.ID
			r.Service().With(rid, func(r res.Resource) {
				r.CreateEvent(book)
			})
			r.AddEvent(res.Ref(rid), i)
		}
	})
}
