# RES service package

A [Go](http://golang.org) package implementing the RES-Service protocol for [Resgate - Real-time API Gateway](https://github.com/jirenius/resgate).

[![GoDoc](https://godoc.org/github.com/jirenius/go-res?status.svg)](http://godoc.org/github.com/jirenius/go-res)

## Installation

```bash
go get github.com/jirenius/go-res
```

## Examples

Install and run [NATS server](https://nats.io/download/nats-io/gnatsd/) and [Resgate](https://github.com/jirenius/resgate):

```bash
go get github.com/nats-io/gnatsd
gnatsd
```
```bash
go get github.com/jirenius/resgate
resgate
```

All examples below contains both a service and a stand-alone client.  
Run the client in multiple browser tabs to observe real-time updates.

### Hello world example

```bash
go get github.com/jirenius/go-res/example/res-helloworld
res-helloworld
```

Go to http://localhost:8081/

### Book collection example

```bash
go get github.com/jirenius/go-res/example/res-collection
res-collection
```

Go to http://localhost:8082/
