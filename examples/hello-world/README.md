# Hello World example

This is an example of a simple Hello World RES service written in Go.
* It exposes a single resource: `example.mymodel`.
* It allows setting the resource's `message` property through the `set` method.
* It resets the model on server restart.
* It serves a web client at http://localhost:8081

## Prerequisite

* Have [NATS Server](https://nats.io/download/nats-io/gnatsd/) and [Resgate](https://github.com/jirenius/resgate) running

## Install and run

Clone go-res repository and run example:
```bash
git clone https://github.com/jirenius/go-res
cd go-res/examples/hello-world
go run main.go
```

Open the client
```
http://localhost:8081
```

## Things to try out

**Realtime updates**  
Run the client in two separate tabs, edit the message in one tab, and observe realtime updates in both.

## Web resources

Resources can be retrieved using ordinary HTTP GET requests, and methods can be called using HTTP POST requests.

### Get model
```
GET http://localhost:8080/api/exampleService/myModel
```

### Update model
```
POST http://localhost:8080/api/exampleService/myModel/set
```
*Body*  
```
{ "message": "Updated through HTTP" }
```