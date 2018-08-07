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

Clone go-res repository:
```bash
git clone https://github.com/jirenius/go-res
```

All examples below contains both a service and a stand-alone client.  
Run the client in multiple browser tabs to observe real-time updates.

### Hello world example

```bash
cd go-res/example/res-helloworld
go run main.go
```

Go to http://localhost:8081/

### Book collection example

```bash
cd go-res/example/res-collection
go run main.go
```

Go to http://localhost:8082/

## Contributing

The go-res package is still under development, and commits may still contain breaking changes. It should only be used for educational purpose. Any feedback on the package API or its implementation is highly appreciated!

If you find any issues, feel free to [report them](https://github.com/jirenius/go-res/issues/new) as an Issue.

