# Edit Text Persisted Example

**Tags:** *Models*, *Call methods*, *Client subscriptions*, *Persistence*

## Description
This is the Edit Text example where all changes are persisted using the BadgerDB middleware. With a database middleware, both clients and database can be updated with a single event.

## Prerequisite

* [Download](https://golang.org/dl/) and install Go
* [Install](https://resgate.io/docs/get-started/installation/) *NATS Server* and *Resgate* (done with 3 docker commands).

## Install and run

```text
git clone https://github.com/jirenius/go-res
cd go-res/examples/03-edit-text-persisted
go run main.go
```

Open the client
```
http://localhost:8083
```

## Things to try out

### BadgerDB persistance
Run the client and make edits to the text. Restart the service and observe all changes are persisted.

## API

Request | Resource | Description
--- | --- | ---
*get* | `text.shared` | Simple model.
*call* | `text.shared.set` | Sets the model's *message* property.

## REST API

Resources can be retrieved using ordinary HTTP GET requests, and methods can be called using HTTP POST requests.

### Get model
```
GET http://localhost:8080/api/text/shared
```

### Update model
```
POST http://localhost:8080/api/text/shared/set
```
*Body*  
```
{ "message": "Updated through HTTP" }
```
