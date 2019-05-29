# Edit Text BadgerDB Example

This is the Edit Text example where all changes are persisted using the BadgerDB middleware.  
By using the BadgerDB middleware, both clients and database can be updated with a single event.

* It exposes a single resource: `example.shared`.
* It allows setting the resource's `message` property through the `set` method.
* The middleware adds a GetResource handler that loads the resource from the database.
* The middleware adds a ApplyChange handler that updates the database on change events.
* It persists all changes to a local BadgerDB database under `./db`.
* It serves a web client at http://localhost:8082

## Prerequisite

* Have [NATS Server](https://nats.io/download/nats-io/gnatsd/) and [Resgate](https://github.com/resgateio/resgate) running

## Install and run

Clone go-res repository and run example:
```bash
git clone https://github.com/jirenius/go-res
cd go-res/examples/edit-text-badgerdb
go run main.go
```

Open the client
```
http://localhost:8082
```

## Things to try out

**BadgerDB persistance**  
Run the client and make edits to the text. Restart the service and observe all changes are persisted.

## Web resources

Resources can be retrieved using ordinary HTTP GET requests, and methods can be called using HTTP POST requests.

### Get model
```
GET http://localhost:8080/api/example/shared
```

### Update model
```
POST http://localhost:8080/api/example/shared/set
```
*Body*  
```
{ "message": "Updated through HTTP" }
```