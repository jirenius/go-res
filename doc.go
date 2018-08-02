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

	serv := res.NewService("exampleService")

Add handlers for a single resource:

	s.Handle("myModel",
		res.Access(res.AccessGranted),
		res.Get(func(r *res.Request, w *res.GetResponse) {
			w.Model(myModel)
		}),
	)

Add handlers for parameterized resources:

	s.Handle("book.$id",
		res.Access(res.AccessGranted),
		res.Get(func(r *res.Request, w *res.GetResponse) {
			book := getBook(r.PathParams["id"]) // Returns nil if not found
			if book == nil {
				w.NotFound()
			} else {
				w.Model(book)
			}
		}),
	)

Add handlers for method calls:

	s.Handle("math",
		...
		res.Call("double", func(r *res.Request, w *res.CallResponse) {
			var p struct {
				Value `json:"value"`
			}
			r.UnmarshalParams(&p)

			w.OK(p.Value * 2)
		}),
	)

Send event:

	s.Handle("myModel",
		...
		res.Call("set", func(r *res.Request, w *res.CallResponse) {
			var p struct {
				Message *string `json:"message,omitempty"`
			}
			r.UnmarshalParams(&p)

			// Check if the message property was changed
			if p.Message != nil && *p.Message != myModel.Message {
				// Update the model
				myModel.Message = *p.Message
				// Send a change event with updated fields
				r.Event("change", p)
			}

			// Send success response
			w.OK(nil)
		}),
	)
*/
package res
