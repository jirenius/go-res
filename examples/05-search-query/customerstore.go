package main

import (
	"encoding/json"
	"io/ioutil"
	"net/url"
	"strconv"
	"strings"

	"github.com/dgraph-io/badger"
	"github.com/jirenius/go-res"
	"github.com/jirenius/go-res/store/badgerstore"
	"github.com/rs/xid"
)

// CustomerStore holds all the customers and provides a way to make queries for
// customers matching certain filters.
//
// It implements the store.QueryStore interface which allows simple read/write
// functionality based on an ID string.
//
// CustomerStore uses BadgerDB, a key/value store, where each customer model is
// stored as a single value. Other databases could be used as well: a SQL table
// where each row is a customer model, or a mongoDB collection where each
// customer is a document. What is needed is a wrapper that implements the Store
// and QueryStore interfaces found in package:
//
//  github.com/jirenius/go-res/store
//
type CustomerStore struct {
	*badgerstore.Store
	CustomersQuery *badgerstore.QueryStore
}

// BadgerDB store indexes.
var (
	// Index on lower case name
	idxCustomerName = badgerstore.Index{
		Name: "idxCustomer_name",
		Key: func(v interface{}) []byte {
			customer := v.(Customer)
			return []byte(strings.ToLower(customer.Name))
		},
	}

	// Index on country and lower case name
	idxCustomerCountryName = badgerstore.Index{
		Name: "idxCustomer_country_name",
		Key: func(v interface{}) []byte {
			customer := v.(Customer)
			return []byte(customer.Country + "_" + strings.ToLower(customer.Name))
		},
	}
)

// NewCustomerStore creates a new CustomerStore.
func NewCustomerStore(db *badger.DB) *CustomerStore {
	st := badgerstore.NewStore(db).
		SetType(Customer{}).
		SetPrefix("customer")
	return &CustomerStore{
		Store: st,
		CustomersQuery: badgerstore.NewQueryStore(st, customersIndexQuery).
			AddIndex(idxCustomerName).
			AddIndex(idxCustomerCountryName),
	}
}

// ParseQuery parses and validates a query to pass use with CustomersQuery.
func (st *CustomerStore) ParseQuery(q url.Values) (name string, country string, from int, limit int, err error) {
	return parseQuery(q)
}

// parseQuery validates and returns the values out of the provided url.Values.
// On parse error, parseQuery returns a res.CodeInvalidQuery error.
func parseQuery(q url.Values) (name string, country string, from int, limit int, err error) {
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
		err = &res.Error{Code: res.CodeInvalidQuery, Message: "Limit must be 50 or less."}
		return
	}
	err = nil
	return
}

// customersIndexQuery handles query requests. This method is badgerstore
// specific, and allows for simple index based queries towards the badgerDB
// store.
//
// Other database implementations for store.QueryStore would do it differently.
// A sql implementation might have you generate a proper WHERE statement, where
// as a mongoDB implementation would need a bson query document.
func customersIndexQuery(qs *badgerstore.QueryStore, q url.Values) (*badgerstore.IndexQuery, error) {
	// Parse the query string
	name, country, from, limit, err := parseQuery(q)
	if err != nil {
		return nil, err
	}

	// Get the index and prefix for this search
	var prefix []byte
	var idx badgerstore.Index
	switch {
	case country != "":
		idx = idxCustomerCountryName
		prefix = []byte(country + "_" + name)
	case name != "":
		idx = idxCustomerName
		prefix = []byte(name)
	default:
		idx = idxCustomerName
	}

	return &badgerstore.IndexQuery{
		Index:     idx,
		KeyPrefix: prefix,
		Offset:    from,
		Limit:     limit,
	}, nil
}

// Init bootstraps an empty store with customers loaded from a file. It panics
// on errors.
func (st *CustomerStore) Init() {
	if err := st.Store.Init(func(add func(id string, v interface{})) error {
		dta, err := ioutil.ReadFile("mock_customers.json")
		if err != nil {
			return err
		}
		var customers []Customer
		if err = json.Unmarshal(dta, &customers); err != nil {
			return err
		}
		for _, customer := range customers {
			customer.ID = xid.New().String()
			add(customer.ID, customer)
		}
		return nil
	}); err != nil {
		panic(err)
	}
	// Wait for the badgerDB index to be created
	st.CustomersQuery.Flush()
}
