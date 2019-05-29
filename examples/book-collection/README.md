# Book Collection Example

This is an example RES service that shows a lists of books, where book titles can be added,
edited and deleted by multiple users simultaneously.

* It exposes a collection, `library.books`, containing book model references.
* It exposes book models, `library.book.<BOOK_ID>`, of each book.
* It allows setting the books' *title* and *author* property through the `set` method.
* It allows creating new books that are added to the collection with the `new` method.
* It allows deleting existing books from the collection with the `delete` method.
* It verifies that a *title* and *author* is always set.
* It resets the collection and models on server restart.

## Prerequisite

* Have [NATS Server](https://nats.io/download/nats-io/gnatsd/) and [Resgate](https://github.com/resgateio/resgate) running

## Install and run

Clone go-res repository and run example:
```bash
git clone https://github.com/jirenius/go-res
cd go-res/examples/book-collection
go run main.go
```

Open the client
```
http://localhost:8082
```

## Things to try out

**Realtime updates**  
Run the client in two separate tabs to observe realtime updates.

**System reset**  
Run the client and make some changes. Restart the node.js server to observe resetting of resources in clients.

**Resynchronization**  
Run the client on two separate devices. Disconnect one device, then make changes with the other. Reconnect the first device to observe resynchronization.


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