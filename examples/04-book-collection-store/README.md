# Book Collection Store Example

**Tags:** *Models*, *Collections*, *Linked resources*, *Call methods*, *Resource parameters*, *Store*

## Description
This is the Book Collection example where all changes are persisted using a badgerDB store.

## Prerequisite

* [Download](https://golang.org/dl/) and install Go
* [Install](https://resgate.io/docs/get-started/installation/) *NATS Server* and *Resgate* (done with 3 docker commands).

## Install and run

```text
git clone https://github.com/jirenius/go-res
cd go-res/examples/04-book-collection-store
go run .
```

Open the client
```text
http://localhost:8084
```

## Things to try out

### BadgerDB persistence
Run the client and make changes to the list of books. Restart the service and observe that all changes are persisted.

### Title sorting
The store sorts books by title. Change the title of one of the books. Observe how the list remains sorted by title.

## API

Request | Resource | Description
--- | --- | ---
*get* | `library.books` | Collection of book model references.
*call* | `library.books.new` | Creates a new book.
*get* | `library.book.<BOOK_ID>` | Models representing books.
*call* | `library.book.<BOOK_ID>.set` | Sets the books' *title* and *author* properties.
*call* | `library.book.<BOOK_ID>.delete` | Deletes a book.

## REST API

Resources can be retrieved using ordinary HTTP GET requests, and methods can be called using HTTP POST requests.

### Get book collection
```
GET http://localhost:8080/api/library/books
```

### Get book
```
GET http://localhost:8080/api/library/book/<BOOK_ID>
```

### Update book properties
```
POST http://localhost:8080/api/library/book/<BOOK_ID>/set
```
*Body*  
```
{ "title": "Animal Farming" }
```

### Add new book
```
POST http://localhost:8080/api/library/books/new
```
*Body*  
```
{ "title": "Dracula", "author": "Bram Stoker" }
```

### Delete book
```
POST http://localhost:8080/api/library/books/delete
```
*Body*  
```
{ "id": <BOOK_ID> }
```
