package main

import (
	"github.com/jirenius/go-res"
	"github.com/jirenius/go-res/store"
)

// CustomerHandler is a handler for customer requests.
type CustomerHandler struct {
	CustomerStore *CustomerStore
}

// SetOption sets the res.Handler options.
func (h *CustomerHandler) SetOption(rh *res.Handler) {
	rh.Option(
		// Handler handels models
		res.Model,
		// Store handler that handles get requests and change events.
		store.Handler{Store: h.CustomerStore, Transformer: h},
		// Set call method handler, for updating the customer's fields.
		res.Call("set", h.setCustomer),
		// Delete call method handler, for deleting customers.
		res.Call("delete", h.deleteCustomer),
	)
}

// RIDToID transforms an external resource ID to a customer ID used by the store.
//
// Since id is equal is to the value of the $id tag in the resource name, we can
// just take it from pathParams.
func (h *CustomerHandler) RIDToID(rid string, pathParams map[string]string) string {
	return pathParams["id"]
}

// IDToRID transforms a customer ID used by the store to an external resource
// ID.
//
// The pattern, p, is the full pattern registered to the service (eg.
// "search.customer.$id") for this resource.
func (h *CustomerHandler) IDToRID(id string, v interface{}, p res.Pattern) string {
	return string(p.ReplaceTag("id", id))
}

// Transform allows us to transform the stored customer model before sending it
// off to external clients. In this example, we do no transformation.
func (h *CustomerHandler) Transform(id string, v interface{}) (interface{}, error) {
	// // We could convert the customer to a type with a different JSON marshaler,
	// // or perhaps return a res.ErrNotFound if a deleted flag is set.
	// return CustomerWithDifferentJSONMarshaler(v.(Customer)), nil
	return v, nil
}

// setCustomer handles call requests to edit customer properties.
func (h *CustomerHandler) setCustomer(r res.CallRequest) {
	// Create a store write transaction.
	txn := h.CustomerStore.Write(r.PathParam("id"))
	defer txn.Close()

	// Get customer value from store
	v, err := txn.Value()
	if err != nil {
		r.Error(err)
		return
	}
	customer := v.(Customer)

	// Parse parameters
	var p struct {
		Name    *string `json:"name"`
		Email   *string `json:"email"`
		Country *string `json:"country"`
	}
	r.ParseParams(&p)

	// Set the provided fields
	if p.Name != nil {
		customer.Name = *p.Name
	}
	if p.Email != nil {
		customer.Email = *p.Email
	}
	if p.Country != nil {
		customer.Country = *p.Country
	}

	// Trim and validate fields
	err = customer.TrimAndValidate()
	if err != nil {
		r.Error(err)
		return
	}

	// Update customer in store.
	// This will produce a change event and a customers query collection event,
	// if any indexed fields were updated.
	err = txn.Update(customer)
	if err != nil {
		r.Error(err)
		return
	}

	// Send success response
	r.OK(nil)
}

// deleteCustomer handles call requests to delete customers.
func (h *CustomerHandler) deleteCustomer(r res.CallRequest) {
	// Create a store write transaction
	txn := h.CustomerStore.Write(r.PathParam("id"))
	defer txn.Close()

	// Delete the customer from the store.
	// This will produce a query event for the customers query collection.
	if err := txn.Delete(); err != nil {
		r.Error(err)
		return
	}

	// Send success response.
	r.OK(nil)
}
