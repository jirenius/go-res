# RES service package

[![GoDoc](https://godoc.org/github.com/jirenius/go-res?status.svg)](http://godoc.org/github.com/jirenius/go-res)

A [Go](http://golang.org) package implementing the RES-Service protocol for [Resgate - Real-time API Gateway](https://github.com/jirenius/resgate).  
When you want to create stateless REST API services but need to have all your resources updated in real time on your reactive web clients.

All resources and methods served by RES services are made accessible through [Resgate](https://github.com/jirenius/resgate) in two ways:
* Ordinary HTTP requests
* Over WebSocket using [ResClient](https://www.npmjs.com/package/resclient)

With ResClient, all resources will be updated in real time, without having to write a single line of client code to handle specific events. It just works.

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

import res "github.com/jirenius/go-res"

func main() {
    s := res.NewService("hello")
    s.Handle("world",
        res.Access(res.AccessGranted),
        res.GetModel(func(w res.GetModelResponse, r *res.Request) {
            w.Model(map[string]string{"greeting": "welcome"})
        }),
    )
    s.ListenAndServe("nats://localhost:4222")
}
```

### Usage

While a RES service communicates over a message broker (NATS Server), instead of listening to HTTP request, the pattern of requests and responses are similar.

#### Create a new service

    serv := res.NewService("myservice")

#### Add model handlers

```go
mymodel := map[string]interface{}{"name": "foo", "value": 42}
s.Handle("mymodel",
    res.Access(res.AccessGranted),
    res.GetModel(func(w res.GetModelResponse, r *res.Request) {
        w.Model(mymodel)
    }),
)
```

#### Add collection handlers

```go
mycollection := []string{"first", "second", "third"}
s.Handle("mycollection",
    res.Access(res.AccessGranted),
    res.GetCollection(func(w res.GetCollectionResponse, r *res.Request) {
        w.Collection(mycollection)
    }),
)
```

#### Add handlers for parameterized resources

```go
s.Handle("article.$id",
    res.Access(res.AccessGranted),
    res.GetModel(func(w res.GetModelResponse, r *res.Request) {
        article := getArticle(r.PathParams["id"])
        if article == nil {
            w.NotFound()
        } else {
            w.Model(article)
        }
    }),
)
```

#### Add handlers for method calls

```go
s.Handle("math",
    res.Access(res.AccessGranted),
    res.Call("double", func(w res.CallResponse, r *res.Request) {
        var p struct {
            Value int `json:"value"`
        }
        r.UnmarshalParams(&p)
        w.OK(p.Value * 2)
    }),
)
```

#### Send change event on model update
A change event will update the model on all subscribing clients.
```go
s.Get("myservice.mymodel", func(r *res.Resource) {
    mymodel["name"] = "bar"
    r.ChangeEvent(map[string]interface{}{"name": "bar"})
})
```

#### Send add event on collection update:
An add event will update the collection on all subscribing clients.

```go
s.Get("myservice.mycollection", func(r *res.Resource) {
    mycollection = append(mycollection, "fourth")
    r.AddEvent("fourth", len(mycollection)-1)
})
```

#### Start service

```go
s.ListenAndServe("nats://localhost:4222")
```

## Credits

Inspiration on the API has been taken from [github.com/go-chi/chi](https://github.com/go-chi/chi), and will continue to do so when it is time to implement Middleware, sub-handlers, and mounting.

## Contributing

The go-res package is still under development, and commits may still contain breaking changes. It should only be used for educational purpose. Any feedback on the package API or its implementation is highly appreciated!

If you find any issues, feel free to [report them](https://github.com/jirenius/go-res/issues/new) as an Issue.

