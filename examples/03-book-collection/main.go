/*
This is an example RES service that shows a lists of books, where book titles can be added,
edited and deleted by multiple users simultaneously.

* It exposes a collection, `library.books`, containing book model references.
* It exposes book models, `library.book.<BOOK_ID>`, of each book.
* It allows setting the books' *title* and *author* property through the `set` method.
* It allows creating new books that are added to the collection with the `new` method.
* It allows deleting existing books from the collection with the `delete` method.
* It verifies that a *title* and *author* is always set.
* It resets the collection and models on server restart.
* It serves a web client at http://localhost:8083
*/
package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/jirenius/go-res"
)

// Book represents a book model
type Book struct {
	ID     int64  `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
}

// Map of all book models
var bookModels = map[string]*Book{
	"library.book.1": {ID: 1, Title: "Animal Farm", Author: "George Orwell"},
	"library.book.2": {ID: 2, Title: "Brave New World", Author: "Aldous Huxley"},
	"library.book.3": {ID: 3, Title: "Coraline", Author: "Neil Gaiman"},
}

// Collection of books
var books = []res.Ref{
	res.Ref("library.book.1"),
	res.Ref("library.book.2"),
	res.Ref("library.book.3"),
}

// ID counter for new book models
var nextBookID int64 = 4

func main() {
	// Create a new RES Service
	s := res.NewService("library")

	// Add handlers for "library.book.$id" models
	s.Handle(
		"book.$id",
		res.Access(res.AccessGranted),
		res.GetModel(getBookHandler),
		res.Set(setBookHandler),
	)

	// Add handlers for "library.books" collection
	s.Handle(
		"books",
		res.Access(res.AccessGranted),
		res.GetCollection(getBooksHandler),
		res.Call("new", newBookHandler),
		res.Call("delete", deleteBookHandler),
	)

	// Run a simple webserver to serve the client.
	// This is only for the purpose of making the example easier to run.
	go func() { log.Fatal(http.ListenAndServe(":8083", http.FileServer(http.Dir("wwwroot/")))) }()
	fmt.Println("Client at: http://localhost:8083/")

	s.ListenAndServe("nats://localhost:4222")
}

func getBookHandler(r res.ModelRequest) {
	book := bookModels[r.ResourceName()]
	if book == nil {
		r.NotFound()
		return
	}
	r.Model(book)
}

func setBookHandler(r res.CallRequest) {
	book := bookModels[r.ResourceName()]
	if book == nil {
		r.NotFound()
		return
	}

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
		// Update the model.
		book.Title = *p.Title
		changed["title"] = book.Title
	}

	// Check if the author property was changed
	if p.Author != nil && *p.Author != book.Author {
		// Update the model.
		book.Author = *p.Author
		changed["author"] = book.Author
	}

	// Send a change event with updated fields
	r.ChangeEvent(changed)

	// Send success response
	r.OK(nil)
}

func getBooksHandler(r res.CollectionRequest) {
	r.Collection(books)
}

func newBookHandler(r res.CallRequest) {
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
	rid := fmt.Sprintf("library.book.%d", nextBookID)
	book := &Book{ID: nextBookID, Title: title, Author: author}
	nextBookID++
	bookModels[rid] = book

	// Convert resource ID to a resource reference
	ref := res.Ref(rid)
	// Send add event
	r.AddEvent(ref, len(books))
	// Appends the book reference to the collection
	books = append(books, ref)

	// Respond with a reference to the newly created book model
	r.Resource(rid)
}

func deleteBookHandler(r res.CallRequest) {
	// Unmarshal parameters to an anonymous struct
	var p struct {
		ID int64 `json:"id,omitempty"`
	}
	r.ParseParams(&p)

	rname := fmt.Sprintf("library.book.%d", p.ID)

	// Ddelete book if it exist
	if _, ok := bookModels[rname]; ok {
		delete(bookModels, rname)
		// Find the book in books collection, and remove it
		for i, rid := range books {
			if rid == res.Ref(rname) {
				// Remove it from slice
				books = append(books[:i], books[i+1:]...)
				// Send remove event
				r.RemoveEvent(i)

				break
			}
		}
	}

	// Send success response. It is up to the service to define if a delete
	// should be idempotent or not. In this case we send success regardless
	// if the book existed or not, making it idempotent.
	r.OK(nil)
}
