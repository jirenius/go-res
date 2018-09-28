# RES service package

[![GoDoc](https://godoc.org/github.com/jirenius/go-res?status.svg)](http://godoc.org/github.com/jirenius/go-res)

A [Go](http://golang.org) package implementing the RES-Service protocol for [Resgate - Real-time API Gateway](https://github.com/jirenius/resgate).  
When you want to create stateless REST API services but need to have your reactive web clients updated in real time.

## Installation

```bash
go get github.com/jirenius/go-res
```

## Examples

* [Hello World](examples/hello-world/) - Single model updated in real time
* [Book Collection](examples/book-collection/) - List of books, added, edited, and updated in real time

### As easy as

```go
package main

import (
	"net/http"
	"github.com/go-chi/chi"
)

func main() {
	s := res.Service("hello")
	r.Handle("world", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("welcome"))
	})
	http.ListenAndServe(":3000", r)
}
```

## Contributing

The go-res package is still under development, and commits may still contain breaking changes. It should only be used for educational purpose. Any feedback on the package API or its implementation is highly appreciated!

If you find any issues, feel free to [report them](https://github.com/jirenius/go-res/issues/new) as an Issue.

