package main

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/jirenius/go-res"
	"github.com/jirenius/go-res/store"
	"github.com/rs/xid"
)

// CustomerQueryHandler is a handler for customer query collection requests.
type CustomerQueryHandler struct {
	CustomerStore *CustomerStore
}

// SetOption sets the handler methods to the res.Handler object
func (h *CustomerQueryHandler) SetOption(rh *res.Handler) {
	rh.Option(
		// Handler handels a collection
		res.Collection,
		// QueryStore handler that handles get requests and change events.
		store.QueryHandler{
			QueryStore:          h.CustomerStore.CustomersQuery,
			QueryRequestHandler: h.queryRequestHandler,
			// The transformer transforms the QueryStore's resulting collection
			// of id strings, []string{"a","b"}, into a collection of resource
			// references, []res.Ref{"search.customer.a","search.customer.b"}.
			Transformer: store.IDToRIDCollectionTransformer(func(id string) string {
				return "search.customer." + id
			}),
		},
		// NewCustomer call method handler, for creating new customers.
		res.Call("newCustomer", h.newCustomer),
	)
}

// newCustomer handles call requests to create new customers.
func (h *CustomerQueryHandler) newCustomer(r res.CallRequest) {
	// Parse request parameters into a customer model
	var customer Customer
	r.ParseParams(&customer)

	// Trim and validate parameters
	if err := customer.TrimAndValidate(); err != nil {
		r.Error(err)
		return
	}

	// Create a new ID for the customer
	customer.ID = xid.New().String()

	// Create a store write transaction
	txn := h.CustomerStore.Write(customer.ID)
	defer txn.Close()

	// Add the customer to the store.
	// This will produce a query event for the customers query collection.
	if err := txn.Create(customer); err != nil {
		r.Error(err)
		return
	}

	// Return a resource reference to a new customer
	r.Resource("search.customer." + customer.ID)
}

// queryRequestHandler takes an incoming request and returns url.Values that
// can be passed to the customer QueryStore.
func (h *CustomerQueryHandler) queryRequestHandler(rname string, pathParams map[string]string, q url.Values) (url.Values, string, error) {
	// Parse the query string
	name, country, from, limit, err := h.CustomerStore.ParseQuery(q)
	if err != nil {
		return nil, "", err
	}

	// Create query for the customers query store.
	cq := url.Values{
		"name":    {name},
		"country": {country},
		"from":    {strconv.Itoa(from)},
		"limit":   {strconv.Itoa(limit)},
	}

	// Create a normalized query string with the properties in a set order. This
	// is used by Resgate to tell if two different looking query strings are
	// essentially the same.
	normalizedQuery := fmt.Sprintf("name=%s&country=%s&from=%d&limit=%d", url.QueryEscape(name), url.QueryEscape(country), from, limit)

	return cq, normalizedQuery, nil
}
