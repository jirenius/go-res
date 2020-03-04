# Edit Text Example

**Tags:** *Models*, *Call methods*, *Client subscriptions*

## Description
A simple text field that can be edited by multiple clients simultaneously.

## Prerequisite

* [Download](https://golang.org/dl/) and install Go
* [Install](https://resgate.io/docs/get-started/installation/) *NATS Server* and *Resgate* (done with 3 docker commands).

## Install and run

```text
git clone https://github.com/jirenius/go-res
cd go-res/examples/02-edit-text
go run .
```

Open the client
```
http://localhost:8082
```

## Things to try out

### Realtime updates
* Open the client in two separate tabs.
* Edit the message in one tab, and observe realtime updates in both.

### System reset
* Stop the service.
* Edit the default text (`"Hello, Go World!"`) in *main.go*.
* Restart the service to observe resetting of the message in all clients.

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