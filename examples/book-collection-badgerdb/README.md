# Book Collection BadgerDB Example 

This is the Book Collection example where all changes are persisted using the BadgerDB middleware.  
By using the BadgerDB middleware, both clients and database can be updated with a single event.

* It exposes a collection, `library.books`, containing book model references.
* It exposes book models, `library.book.<BOOK_ID>`, of each book.
* The middleware adds a GetResource handler that loads the resources from the database.
* The middleware adds a ApplyChange handler that updates the books on change events.
* The middleware adds a ApplyAdd handler that updates the list on add events.
* The middleware adds a ApplyRemove handler that updates the list on remove events.
* The middleware adds a ApplyCreate handler that stores new books on create events.
* The middleware adds a ApplyDelete handler that deletes books on delete events.
* It persists all changes to a local BadgerDB database under `./db`.

## Prerequisite

* Have [NATS Server](https://nats.io/download/nats-io/gnatsd/) and [Resgate](https://github.com/resgateio/resgate) running

## Install and run

Clone go-res repository and run example:
```bash
git clone https://github.com/jirenius/go-res
cd go-res/examples/book-collection-badgerdb
go run main.go
```

Open the client
```
http://localhost:8083
```

## Things to try out

**BadgerDB persistance**  
Run the client and make changes to the list of books. Restart the service and observe that all changes are persisted.

## Web resources

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
POST http://localhost:8080/api/library/books/add
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