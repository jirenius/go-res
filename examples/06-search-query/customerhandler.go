package main

import (
	"encoding/json"
	"io/ioutil"
	"strings"

	"github.com/dgraph-io/badger"
	"github.com/jirenius/go-res"
	"github.com/jirenius/go-res/middleware/resbadger"
	"github.com/rs/xid"
)

// Indices used for customer models.
const (
	idxCustomerName        = "idxCustomer_name"
	idxCustomerCountryName = "idxCustomer_country_name"
)

// Definition of the indexes for the customer models.
var customerIdxs = &resbadger.IndexSet{
	Indexes: []resbadger.Index{
		// Index on lower case name
		resbadger.Index{
			Name: idxCustomerName,
			Key: func(v interface{}) []byte {
				customer := v.(Customer)
				return []byte(strings.ToLower(customer.Name))
			},
		},
		// Index on country and lower case name
		resbadger.Index{
			Name: idxCustomerCountryName,
			Key: func(v interface{}) []byte {
				customer := v.(Customer)
				return []byte(customer.Country + "_" + strings.ToLower(customer.Name))
			},
		},
	},
}

type customerHandler struct {
	DB *badger.DB
}

func (h customerHandler) Access(r res.AccessRequest) {
	r.AccessGranted()
}

// SetOption sets the handler methods to the res.Handler object.
func (h customerHandler) SetOption(hs *res.Handler) {
	hs.Access = h.Access
	hs.Group = "customers"
	m := resbadger.BadgerDB{DB: h.DB}.
		Model().
		WithType(Customer{}).
		WithIndexSet(customerIdxs)
	m.SetOption(hs)
	res.Call("set", h.setCustomer).SetOption(hs)
	res.Call("delete", h.deleteCustomer).SetOption(hs)
	res.OnRegister(func(s *res.Service, pattern string) {
		// Load default data, and rebuild their indexes, unless already loaded
		if !h.hasDefaultData() {
			h.populateDefault()
			m.RebuildIndexes(pattern)
		}
	}).SetOption(hs)
}

// setCustomer handles call requests to edit customer properties.
func (h customerHandler) setCustomer(r res.CallRequest) {
	// Parse and validate parameters
	var p struct {
		Name    *string `json:"name"`
		Email   *string `json:"email"`
		Country *string `json:"country"`
	}
	r.ParseParams(&p)
	if errMsg := customerTrimAndValidate(p.Name, p.Email, p.Country); errMsg != "" {
		r.InvalidParams(errMsg)
		return
	}
	// Populate map with updated fields
	changed := make(map[string]interface{}, 3)
	if p.Name != nil {
		changed["name"] = *p.Name
	}
	if p.Email != nil {
		changed["email"] = *p.Email
	}
	if p.Country != nil {
		changed["country"] = *p.Country
	}
	// Send a change event with updated fields
	r.ChangeEvent(changed)
	// Send success response
	r.OK(nil)
}

// deleteCustomer handles call requests to delete customers.
func (h customerHandler) deleteCustomer(r res.CallRequest) {
	// Send a delete event.
	// The middleware will delete the item from the database.
	r.DeleteEvent()
	// Send success response
	r.OK(nil)
}

// populateDefault loads the mock_customers.json file and imports it
// into the database, unless it has been imported before.
func (h customerHandler) populateDefault() {
	// Load file
	dta, err := ioutil.ReadFile("mock_customers.json")
	panicOnError(err)
	// Decode file content
	var result []Customer
	err = json.Unmarshal(dta, &result)
	panicOnError(err)
	// Write content to Badger DB
	wb := h.DB.NewWriteBatch()
	defer wb.Cancel()
	for _, customer := range result {
		customer.ID = xid.New().String()
		m, err := json.Marshal(customer)
		panicOnError(err)
		err = wb.Set([]byte(customer.RID()), m)
		panicOnError(err)
	}
	panicOnError(wb.Flush())
}

// hasDefaultData checks if customer data is imported.
func (h customerHandler) hasDefaultData() bool {
	imported := true
	err := h.DB.Update(func(txn *badger.Txn) error {
		_, err := txn.Get([]byte("customers_imported"))
		if err == badger.ErrKeyNotFound {
			err = txn.Set([]byte("customers_imported"), []byte{})
			imported = false
		}
		return err
	})
	panicOnError(err)
	return imported
}

func panicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
