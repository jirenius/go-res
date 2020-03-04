// This example uses modapp components to render the view.
// Read more about it here: https://resgate.io/docs/writing-clients/using-modapp/
const { Elem, Txt, Button, Input } = window["modapp-base-component"];
const { CollectionList, ModelTxt } = window["modapp-resource-component"];
const ResClient = resclient.default;

// Creating the client instance.
let client = new ResClient('ws://localhost:8080');

// Error handling
let errMsg = new Txt();
let errTimer = null;
errMsg.render(document.getElementById('error-msg'));
let showError = (err) => {
	errMsg.setText(err && err.message ? err.message : String(err));
	clearTimeout(errTimer);
	errTimer = setTimeout(() => errMsg.setText(''), 7000);
};

// Add new click callback
document.getElementById('add-new').addEventListener('click', () => {
	let newTitle = document.getElementById('new-title');
	let newAuthor = document.getElementById('new-author');
	client.call('library.books', 'new', {
		title: newTitle.value,
		author: newAuthor.value
	}).then(() => {
		// Clear values on successful add
		newTitle.value = "";
		newAuthor.value = "";
	}).catch(showError);
});

// Get the collection from the service.
client.get('library.books').then(books => {
	// Render the collection of books
	new Elem(n =>
		n.component(new CollectionList(books, book => {
			let c = new Elem(n =>
				n.elem('div', { className: 'list-item' }, [
					n.elem('div', { className: 'card shadow' }, [
						// View card mode
						n.elem('div', { className: 'view' }, [
							n.elem('div', { className: 'action' }, [
								n.component(new Button(`Edit`, () => {
									c.getNode('titleInput').setValue(book.title);
									c.getNode('authorInput').setValue(book.author);
									c.addClass('editing');
								})),
								n.component(new Button(`Delete`, () => books.call('delete', { id: book.id }).catch(showError)))
							]),
							n.elem('div', { className: 'title' }, [
								n.component(new ModelTxt(book, book => book.title, { tagName: 'h3' }))
							]),
							n.elem('div', { className: 'author' }, [
								n.component(new Txt("By ")),
								n.component(new ModelTxt(book, book => book.author))
							])
						]),
						// Edit card mode
						n.elem('div', { className: 'edit' }, [
							n.elem('div', { className: 'action' }, [
								n.component(new Button(`OK`, () => {
									book.set({
										title: c.getNode('titleInput').getValue(),
										author: c.getNode('authorInput').getValue()
									})
										.then(() => c.removeClass('editing'))
										.catch(showError);
								})),
								n.component(new Button(`Cancel`, () => c.removeClass('editing')))
							]),
							n.elem('div', { className: 'edit-input' }, [
								n.elem('label', [
									n.component(new Txt("Title", { className: 'span' })),
									n.component('titleInput', new Input())
								]),
								n.elem('label', [
									n.component(new Txt("Author", { className: 'span' })),
									n.component('authorInput', new Input())
								])
							])
						])
					])
				])
			);
			return c;
		}, { className: 'list' }))
	).render(document.getElementById('books'));
}).catch(err => showError(err.code === 'system.connectionError'
	? "Connection error. Are NATS Server and Resgate running?"
	: err
));
