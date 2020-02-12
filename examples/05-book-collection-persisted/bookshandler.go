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
	pattern   res.Pattern
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
		// Get the pattern (eg. "library.book.$id") for this resource. This will
		// be used in the IDToRID transform function, to tell what resource is
		// affected when a book is changed in the store.
		res.OnRegister(func(_ *res.Service, pattern string, _ res.Handler) {
			h.pattern = res.Pattern(pattern)
		}),
	)
}

// newBook handles new call requests on the book collection.
func (h *BooksHandler) newBook(r res.CallRequest) {
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
	book := Book{ID: xid.New().String(), Title: title, Author: author}

	// Create a store write transaction
	txn := h.BookStore.Write(book.ID)
	defer txn.Close()

	// Add the book to the store
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

	// Delete the book from the store
	if err := txn.Delete(); err != nil {
		r.Error(err)
		return
	}

	// Send success response.
	r.OK(nil)
}
