package main

import (
	"strings"

	"github.com/jirenius/go-res"
	"github.com/jirenius/go-res/store"
	"github.com/rs/xid"
)

// BooksHandler is a handler for book collection requests.
type BooksHandler struct {
	BookStore *BookStore
}

// SetOption sets the res.Handler options.
func (h *BooksHandler) SetOption(rh *res.Handler) {
	rh.Option(
		// Handler handels a collection
		res.Collection,
		// QueryStore handler that handles get requests and change events.
		store.QueryHandler{
			QueryStore: h.BookStore.BooksByTitle,
			// The transformer transforms the QueryStore's resulting collection
			// of id strings, []string{"1","2"}, into a collection of resource
			// references, []res.Ref{"library.book.1","library.book.2"}.
			Transformer: store.IDToRIDCollectionTransformer(func(id string) string {
				return "library.book." + id
			}),
		},
		// New call method handler, for creating new books.
		res.Call("new", h.newBook),
		// Delete call method handler, for deleting books.
		res.Call("delete", h.deleteBook),
	)
}

// newBook handles new call requests on the book collection.
func (h *BooksHandler) newBook(r res.CallRequest) {
	// Parse request parameters into a book model
	var book Book
	r.ParseParams(&book)

	// Trim whitespace
	book.Title = strings.TrimSpace(book.Title)
	book.Author = strings.TrimSpace(book.Author)

	// Check if we received both title and author
	if book.Title == "" || book.Author == "" {
		r.InvalidParams("Must provide both title and author")
		return
	}

	// Create a new ID for the book
	book.ID = xid.New().String()

	// Create a store write transaction
	txn := h.BookStore.Write(book.ID)
	defer txn.Close()

	// Add the book to the store.
	// This will produce an add event for the books collection.
	if err := txn.Create(book); err != nil {
		r.Error(err)
		return
	}

	// Return a resource reference to a new book
	r.Resource("library.book." + book.ID)
}

// deleteBook handles delete call requests on the book collection.
func (h *BooksHandler) deleteBook(r res.CallRequest) {
	// Unmarshal parameters to an anonymous struct
	var p struct {
		ID string `json:"id,omitempty"`
	}
	r.ParseParams(&p)

	// Create a store write transaction
	txn := h.BookStore.Write(p.ID)
	defer txn.Close()

	// Delete the book from the store.
	// This will produce a remove event for the books collection.
	if err := txn.Delete(); err != nil {
		r.Error(err)
		return
	}

	// Send success response.
	r.OK(nil)
}
