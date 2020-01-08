package main

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/dgraph-io/badger"
	"github.com/jirenius/go-res"
	"github.com/jirenius/go-res/middleware/resbadger"
	"github.com/rs/xid"
)

type customerQueryHandler struct {
	DB *badger.DB
}

func (h customerQueryHandler) Access(r res.AccessRequest) {
	r.AccessGranted()
}

// SetOption sets the handler methods to the res.Handler object
func (h customerQueryHandler) SetOption(hs *res.Handler) {
	hs.Access = h.Access
	hs.Group = "customers"
	resbadger.BadgerDB{DB: h.DB}.
		QueryCollection().
		WithIndexSet(customerIdxs).
		WithQueryCallback(h.queryCallback).
		SetOption(hs)
	res.Call("newCustomer", h.newCustomer).SetOption(hs)
}

// newCustomer handles call requests to create new customers.
func (h customerQueryHandler) newCustomer(r res.CallRequest) {
	var customer Customer
	r.ParseParams(&customer)

	// Trim and validate call params
	if errMsg := customer.TrimAndValidate(); errMsg != "" {
		r.InvalidParams(errMsg)
		return
	}

	// Create a new Customer ID
	customer.ID = xid.New().String()
	rid := customer.RID()

	// Send a create event. The rmiddleware will store it in badger DB.
	r.Service().With(rid, func(re res.Resource) {
		re.CreateEvent(customer)
	})

	// Send success response with new resource ID
	r.Resource(rid)
}

// queryCallback is a callback that gets a query string from a request
// and returns an index query that is used by the resbadger middleware to
// fetch the matching customers.
func (h customerQueryHandler) queryCallback(idxs *resbadger.IndexSet, rname string, params map[string]string, query url.Values) (*resbadger.IndexQuery, string, error) {
	// Parse the query string
	name, country, from, limit, errMsg := h.parseQuery(query)
	if errMsg != "" {
		return nil, "", &res.Error{Code: res.CodeInvalidQuery, Message: errMsg}
	}

	// Get the index and prefix for this search
	var err error
	var prefix []byte
	var idx resbadger.Index
	switch {
	case country != "":
		idx, err = idxs.GetIndex(idxCustomerCountryName)
		prefix = []byte(country + "_" + name)
	case name != "":
		idx, err = idxs.GetIndex(idxCustomerName)
		prefix = []byte(name)
	default:
		idx, err = idxs.GetIndex(idxCustomerName)
	}

	if err != nil {
		return nil, "", err
	}

	// Create a normalized query string with the properties in a set order.
	// This is used by Resgate to tell if two different looking query strings
	// are essentially the same.
	normalizedQuery := fmt.Sprintf("name=%s&country=%s&from=%d&limit=%d", url.QueryEscape(name), url.QueryEscape(country), from, limit)

	return &resbadger.IndexQuery{
		Index:     idx,
		KeyPrefix: prefix,
		Offset:    from,
		Limit:     limit,
	}, normalizedQuery, nil
}

// parseQuery validates and returns the values out of the provided url.Values.
// On parse error, the return errMsg will be non-empty.
func (h customerQueryHandler) parseQuery(q url.Values) (name string, country string, from int, limit int, errMsg string) {
	var err error
	name = strings.ToLower(q.Get("name"))
	country = q.Get("country")
	from, err = strconv.Atoi(q.Get("from"))
	if err != nil {
		from = 0
	}
	limit, err = strconv.Atoi(q.Get("limit"))
	if err != nil {
		limit = -1
	}
	if from < 0 {
		from = 0
	}
	if limit < 0 {
		limit = 10
	}
	if limit > 50 {
		errMsg = "Limit must be 50 or less."
		return
	}

	errMsg = ""
	return
}
