/*
This is an example of how to create a RES service with collections.
* It exposes a collection: "bookService.books".
* It allows setting the books' Title and Author property through the "set" method.
* It allows creating new books that are added to the collection
* It allows deleting existing books from the collection
* It verifies that a title and author is always set
*/
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jirenius/go-res"
)

// Book model
type Book struct {
	ID     int64  `json:"id"`
	Title  string `json:"title"`
	Author string `json:"author"`
}

// Map of all book models
var bookModels = map[string]*Book{
	"bookService.book.1": {ID: 1, Title: "Animal Farm", Author: "George Orwell"},
	"bookService.book.2": {ID: 2, Title: "Brave New World", Author: "Aldous Huxley"},
	"bookService.book.3": {ID: 3, Title: "Coraline", Author: "Neil Gaiman"},
}

// ID counter for book models
var nextBookID int64 = 4

// Mutex to protect the bookModels map and nextBookID counter
var mu sync.RWMutex

// Collection of books
var books = []res.Ref{
	res.Ref("bookService.book.1"),
	res.Ref("bookService.book.2"),
	res.Ref("bookService.book.3"),
}

// getBook looks up a book based on the resource ID.
// Returns nil if no book was found.
func getBook(rid string) *Book {
	mu.RLock()
	defer mu.RUnlock()
	return bookModels[rid]
}

// deleteBook deletes a book from the bookModels map.
// Returns true when found and deleted, otherwise false.
func deleteBook(rid string) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := bookModels[rid]
	if ok {
		delete(bookModels, rid)
	}
	return ok
}

// createBook creates a new Book model, assigns it a unique ID,
// and adds it to the bookModels map.
// It returns the resource ID.
func newBook(title string, author string) string {
	mu.RLock()
	defer mu.RUnlock()
	rid := fmt.Sprintf("bookService.book.%d", nextBookID)
	book := &Book{ID: nextBookID, Title: title, Author: author}
	nextBookID++
	bookModels[rid] = book
	return rid
}

func main() {
	// Create a new RES Service
	serv := res.NewService("bookService")

	handleBookModels(serv)      // Add handlers for the book models
	handleBooksCollection(serv) // Add handlers for the books collection

	// Start service in separate goroutine
	stop := make(chan bool)
	go func() {
		defer close(stop)
		err := serv.ListenAndServe("nats://localhost:4222")
		if err != nil {
			fmt.Printf("%s\n", err.Error())
		}
	}()

	// Serve a client.
	path, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}
	go func() { log.Fatal(http.ListenAndServe(":8082", http.FileServer(http.Dir(path)))) }()
	fmt.Println("Client at: http://localhost:8082/")

	// Wait for interrupt signal
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	select {
	case <-c:
		// Graceful stop
		serv.Stop()
	case <-stop:
	}
}

// handleBookModels adds handlers for "bookService.book.$id" models
func handleBookModels(s *res.Service) {
	s.Handle(
		"book.$id",
		// res.Use(func(next res.Handler) res.Handler {
		// 	return func(r *res.Request, w *res.Response) {
		// 		book := bookModels[r.ResourceName]
		// 		if book == nil {
		// 			w.NotFound()
		// 			return
		// 		}
		// 		r.Value = book
		// 		next(r.WithContext(context.WithValue(r.Context(), "book", book)), w)
		// 	}
		// }),
		res.Access(res.AccessGranted),
		res.GetModel(func(r *res.Request, w *res.GetModelResponse) {
			book := getBook(r.ResourceName)
			if book == nil {
				w.NotFound()
				return
			}
			w.Model(book)
		}),
		res.Set(func(r *res.Request, w *res.CallResponse) {
			book := getBook(r.ResourceName)
			if book == nil {
				w.NotFound()
				return
			}

			// Unmarshal parameters to a anonymous struct
			var p struct {
				Title  *string `json:"title,omitempty"`
				Author *string `json:"author,omitempty"`
			}
			r.UnmarshalParams(&p)

			changed := make(map[string]interface{}, 2)

			// Check if the title property was changed
			if p.Title != nil {
				// Verify it is not empty
				title := strings.TrimSpace(*p.Title)
				if title == "" {
					w.InvalidParams("Title must not be empty")
					return
				}

				if title != book.Title {
					// Update the model.
					book.Title = title
					changed["title"] = title
				}
			}

			// Check if the author property was changed
			if p.Author != nil {
				// Verify it is not empty
				author := strings.TrimSpace(*p.Author)
				if author == "" {
					w.InvalidParams("Author must not be empty")
					return
				}
				if author != book.Author {
					// Update the model.
					book.Author = author
					changed["author"] = author
				}
			}

			// Send a change event with updated fields
			r.ChangeEvent(changed)

			// Send success response
			w.OK(nil)
		}),
	)
}

// handleBooksCollection adds handlers for "bookService.books" collection
func handleBooksCollection(s *res.Service) {
	s.Handle(
		"books",
		res.Access(res.AccessGranted),
		res.GetCollection(func(r *res.Request, w *res.GetCollectionResponse) {
			w.Collection(books)
		}),
		res.New(func(r *res.Request, w *res.NewResponse) {
			var p struct {
				Title  string `json:"title"`
				Author string `json:"author"`
			}
			r.UnmarshalParams(&p)

			// Trim whitespace
			p.Title = strings.TrimSpace(p.Title)
			p.Author = strings.TrimSpace(p.Author)

			// Check if we received both title and author
			if p.Title == "" || p.Author == "" {
				w.InvalidParams("Must provide both title and author")
				return
			}
			// Create a new book model
			rid := newBook(p.Title, p.Author)
			// Convert resource ID to a resource reference
			ref := res.Ref(rid)
			// Send add event
			r.AddEvent(ref, len(books))
			// Appends the book reference to the collection
			books = append(books, ref)

			// Respond with a reference to the newly created book model
			w.OK(ref)
		}),
		res.Call("delete", func(r *res.Request, w *res.CallResponse) {
			var p struct {
				ID int64 `json:"id,omitempty"`
			}
			r.UnmarshalParams(&p)

			rname := fmt.Sprintf("bookService.book.%d", p.ID)
			if deleteBook(rname) {
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
			w.OK(nil)
		}),
	)
}
