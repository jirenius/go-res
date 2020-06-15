<p align="center"><a href="https://resgate.io" target="_blank" rel="noopener noreferrer"><img width="100" src="https://resgate.io/img/resgate-logo.png" alt="Resgate logo"></a></p>
<h2 align="center"><b>Go utilities for communicating with RES Services</b><br/>Synchronize Your Clients</h2>
<p align="center">
<a href="../../LICENSE"><img src="https://img.shields.io/badge/License-MIT-blue.svg" alt="License"></a>
<a href="https://pkg.go.dev/github.com/jirenius/go-res/resprot"><img src="https://img.shields.io/static/v1?label=reference&message=go.dev&color=5673ae" alt="Reference"></a>
</p>

---

Package *resprot* provides low level structs and methods for communicating with
services using the RES Service Protocol over NATS server.

## Installation

```bash
go get github.com/jirenius/go-res/resprot
```

## Example usage

#### Make a request

```go
conn, _ := nats.Connect("nats://127.0.0.1:4222")
response := resprot.SendRequest(conn, "call.example.ping", nil, time.Second)
```

#### Get a model

```go
response := resprot.SendRequest(conn, "get.example.model", nil, time.Second)

var model struct {
	Message string `json:"message"`
}
_, err := response.ParseModel(&model)
```

#### Call a method

```go
response := resprot.SendRequest(conn, "call.math.add", resprot.Request{Params: struct {
	A float64 `json:"a"`
	B float64 `json:"b"`
}{5, 6}}, time.Second)

var result struct {
	Sum float64 `json:"sum"`
}
err := response.ParseResult(&result)
```
