/*
Package res provides RES service implementations for realtime API's through Resgate:

https://github.com/jirenius/resgate

The implementation provides low level methods to listen to and handle incoming
requests, and to send events.

Concurrency

Requests are handled concurrently for multiple resources, but the package
guarantees that only one goroutine is executing handlers for any unique
resource at any one time. This allows handlers to modify models without
additional synchronization such as mutexes.

Usage

Create a new service:

	serv := res.NewService("myservice")

Add handlers for a single model resource:

	mymodel := map[string]interface{}{"name": "foo", "value": 42}
	s.Handle("mymodel",
		res.Access(res.AccessGranted),
		res.GetModel(func(w res.GetModelResponse, r *res.Request) {
			w.Model(mymodel)
		}),
	)

Add handlers for a single collection resource:

	mycollection := []string{"first", "second", "third"}
	s.Handle("mycollection",
		res.Access(res.AccessGranted),
		res.GetCollection(func(w res.GetCollectionResponse, r *res.Request) {
			w.Collection(mycollection)
		}),
	)

Add handlers for parameterized resources:

	s.Handle("article.$id",
		res.Access(res.AccessGranted),
		res.GetModel(func(w res.GetModelResponse, r *res.Request) {
			article := getArticle(r.PathParams["id"]) // Returns nil if not found
			if article == nil {
				w.NotFound()
			} else {
				w.Model(article)
			}
		}),
	)

Add handlers for method calls:

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

Send change event on model update:

	s.Get("myservice.mymodel", func(r *res.Resource) {
		mymodel["name"] = "bar"
		r.ChangeEvent(map[string]interface{}{"name": "bar"})
	})

Send add event on collection update:

	s.Get("myservice.mycollection", func(r *res.Resource) {
		mycollection = append(mycollection, "fourth")
		r.AddEvent("fourth", len(mycollection)-1)
	})

Start service:

	s.ListenAndServe("nats://localhost:4222")
*/
package res
