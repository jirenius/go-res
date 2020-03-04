// This example uses modapp components to render the view.
// Read more about it here: https://resgate.io/docs/writing-clients/using-modapp/
const { Elem, Txt, Button, Input, Transition } = window["modapp-base-component"];
const { CollectionList, CollectionSelect, ModelTxt } = window["modapp-resource-component"];
const ResClient = resclient.default;

// Creating the client instance.
let client = new ResClient('ws://localhost:8080');

// Error message component
class ErrorComponent extends Txt {
    constructor() {
        super('', { className: 'error' });
    }

    // Shows an error for 7 seconds
    showError(err) {
        if (err && err.code && err.code == 'system.connectionError') {
            err = "Failed to connect to Resgate. Make sure it is up and running on default port 8080.";
        }
        this.setText(err && err.message ? err.message : String(err));
        clearTimeout(this.errTimer);
        this.errTimer = setTimeout(() => this.setText(''), 7000);
    }
}

// Search top bar component
class SearchComponent extends Elem {
    constructor() {
        super(n => n.elem('div', { className: 'search' }, [
            n.elem('label', { attributes: { for: 'search-name' } }, [
                n.component(new Txt("Filter")),
                n.component('name', new Input("", { attributes: { id: 'search-name' } }))
            ]),
            n.elem('label', { attributes: { for: 'search-country' } }, [
                n.component(new Txt("Country")),
                n.component('country', new CollectionSelect(null, country => ({ text: country, value: country }), {
                    placeholder: { text: "All countries", value: "" }
                }))
            ]),
            n.component(new Button("Search", () => search(0), { className: 'primary' }))
        ]));
    }

    setCountries(countries) {
        this.getNode('country').setCollection(countries);
    }

    getValues() {
        return {
            name: this.getNode('name').getValue(),
            country: this.getNode('country').getSelected()
        };
    }
}

// Previous/Next navigation component
class NavigateComponent extends Elem {
    constructor() {
        super(n => n.elem('div', { className: 'navigate' }, [
            n.component('previous', new Button("Previous", () => search(this.from - this.limit, 'slideRight'), { className: "previous" })),
            n.component('next', new Button("Next", () => search(this.from + this.limit, 'slideLeft'), { className: "next" }))
        ]));
        this.setSpan(0, 0, 0);
    }

    setSpan(from, limit, count) {
        this.from = from;
        this.limit = limit;
        this.getNode('previous').setDisabled(from == 0);
        this.getNode('next').setDisabled(count < limit);
    }
}

// Edit/Create customer modal component
class CustomerModal extends Elem {
    constructor(title, buttonText, customer, callback) {
        super(n => n.elem('div', { className: 'modal' }, [
            n.elem('div', { className: 'modal-content shadow' }, [
                n.component('button', new Button("❌", () => this.close(), { className: 'modal-close' })),
                n.elem('div', { className: 'edit' }, [
                    n.component(new Txt(title, { tagName: 'h3', className: 'edit-title' })),
                    n.elem('div', { className: 'edit-input' }, [
                        n.elem('label', [
                            n.component(new Txt("Name", { className: 'span' })),
                            n.component('nameInput', new Input(customer.name || ""))
                        ]),
                        n.elem('label', [
                            n.component(new Txt("Email", { className: 'span' })),
                            n.component('emailInput', new Input(customer.email || ""))
                        ]),
                        n.elem('label', [
                            n.component(new Txt("Country", { className: 'span' })),
                            n.component('countrySelect', new CollectionSelect(countries, country => ({ text: country, value: country }), {
                                placeholder: { text: "Not set", value: "" },
                                selected: customer.country || ""
                            }))
                        ])
                    ])
                ]),
                n.elem('div', { className: 'action' }, [
                    n.component(new Button(buttonText, () => {
                        callback({
                            name: this.getNode('nameInput').getValue(),
                            email: this.getNode('emailInput').getValue(),
                            country: this.getNode('countrySelect').getSelected()
                        }, this);
                    }))
                ]),
                n.component('errMsg', new ErrorComponent())
            ])
        ]));
    }
    open() { this.render(document.body); }
    close() { this.unrender(); }
    showError(err) {
        this.getNode('errMsg').showError(err);
    }
}

// Create new customer button component
class NewCustomerComponent extends Elem {
    constructor() {
        super(n => n.elem('div', { className: 'new' }, [
            n.component(new Button("New customer", () => {
                new CustomerModal("New customer", "OK", { name: "", country: "", email: "" }, (o, modal) => {
                    client.call('search.customers', 'newCustomer', o)
                        .then(() => modal.close())
                        .catch(err => modal.showError(err));
                }).open();
            }, { className: 'primary' }))
        ]));
    }
}

// Customer info card component
class CustomerComponent extends Elem {
    constructor(customer) {
        super(n =>
            n.elem('div', { className: 'list-item' }, [
                n.elem('div', { className: 'card shadow' }, [
                    n.elem('div', { className: 'view' }, [
                        n.elem('div', { className: 'action' }, [
                            n.component(new Button(`Edit`, () => {
                                new CustomerModal("Edit customer", "OK", customer, (o, modal) => {
                                    customer.set(o)
                                        .then(() => modal.close())
                                        .catch(err => modal.showError(err));
                                }).open();
                            })),
                            n.component(new Button(`Delete`, () => customer.call('delete')))
                        ]),
                        n.elem('div', { className: 'avatar' }),
                        n.elem('div', { className: 'name' }, [
                            n.component(new ModelTxt(customer, customer => customer.name, { tagName: 'h3' }))
                        ]),
                        n.elem('div', { className: 'country' }, [
                            n.component(new ModelTxt(customer, customer => customer.country))
                        ]),
                        n.elem('div', { className: 'email' }, [
                            n.component(new ModelTxt(customer, customer => customer.email))
                        ])
                    ])
                ])
            ])
        );
    }
}

// Create & render the different components where we want them.
let searchComponent = new SearchComponent();
searchComponent.render(searchDiv);
let errorComponent = new ErrorComponent();
errorComponent.render(errorDiv);
let navigateComponent = new NavigateComponent();
navigateComponent.render(navigateDiv);
let actionComponent = new NewCustomerComponent();
actionComponent.render(actionDiv);
let customersComponent = new Transition();
customersComponent.render(customersDiv);

// Load countries and populate the searchComponent's select list.
// Then store the countries in a global variable.
let countries = null;
client.get('search.countries').then(collection => {
    countries = collection;
    searchComponent.setCountries(countries);
    search(0);
}).catch(err => errorComponent.showError(err));

// Sends a search query based on the values in the search top bar.
// Renders the results once they return.
function search(from, transition) {
    let { name, country } = searchComponent.getValues();
    let limit = 10;

    // Get the collection from the service.
    client.get(`search.customers?name=${name}&country=${country}&from=${from}&limit=${limit}`).then(customers => {
        // Set the customer list
        navigateComponent.setSpan(from, limit, customers.length);
        customersComponent[transition || "fade"](new Elem(n =>
            n.elem('div', { className: "customers" }, [
                // This text is static and won't change even on events.
                // By updating it on customers add/remove events, this can be resolved.
                n.component(new Txt(
                    customers.length > 0
                        ? `Showing customers ${from + 1} to ${from + customers.length}` +
                        (country ? ` from ${country}` : '') +
                        (name ? ` with names starting with ${name}` : '')
                        : `No customers to show`
                )),
                // CollectionList renders a list of components.
                // It takes care of add/remove events and makes nice slide transitions to show/hide customers.
                n.component(new CollectionList(customers, customer => new CustomerComponent(customer), { className: 'list' }))
            ])
        ));
    }).catch(err => errorComponent.showError(err));
}


