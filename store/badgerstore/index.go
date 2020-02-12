package badgerstore

import (
	"bytes"
	"fmt"

	"github.com/dgraph-io/badger"
)

// Index defines an index used for a resource.
//
// When used on Model resource, an index entry will be added for each model entry.
// An index entry will have no value (nil), and the key will have the following structure:
//    <Name>:<Key>\x00<RID>
// Where:
// * <Name> is the name of the Index (so keep it rather short)
// * <Key> is the index value as returned from the Key callback
// * <RID> is the resource ID of the indexed model
type Index struct {
	// Index name
	Name string
	// Key callback is called with a resource item of the type defined by Type,
	// and should return the string to use as index value.
	// It does not have to be unique.
	//
	// Example index by Country and lower case Name on a user model:
	// 	func(v interface{}) {
	// 		user := v.(UserModel)
	// 		return []byte(user.Country + "_" + strings.ToLower(user.Name))
	// 	}
	Key func(interface{}) []byte
}

// IndexQuery represents a query towards an index.
type IndexQuery struct {
	// Index used
	Index Index
	// KeyPrefix to match against the index key
	KeyPrefix []byte
	// FilterKeys for keys in the query collection. May be nil.
	FilterKeys func(key []byte) bool
	// Offset from which item to start.
	Offset int
	// Limit how many items to read. Negative means unlimited.
	Limit int
	// Reverse flag to tell if order is reversed
	Reverse bool
}

// Byte that separates the index key prefix from the resource ID.
const idSeparator = byte(0)

// Max initial buffer size for results, and default size for limit set to -1.
const resultBufSize = 256

// Max int value.
const maxInt = int(^uint(0) >> 1)

func (idx Index) getKey(rname []byte, value []byte) []byte {
	b := make([]byte, len(idx.Name)+len(value)+len(rname)+2)
	copy(b, idx.Name)
	offset := len(idx.Name)
	b[offset] = ':'
	offset++
	copy(b[offset:], value)
	offset += len(value)
	b[offset] = idSeparator
	copy(b[offset+1:], rname)
	return b
}

func (idx Index) getQuery(keyPrefix []byte) []byte {
	b := make([]byte, len(idx.Name)+len(keyPrefix)+1)
	copy(b, idx.Name)
	offset := len(idx.Name)
	b[offset] = ':'
	offset++
	copy(b[offset:], keyPrefix)
	return b
}

// FetchCollection fetches a collection of resource references based on the query.
func (iq *IndexQuery) FetchCollection(db *badger.DB) ([]string, error) {
	offset := iq.Offset
	limit := iq.Limit

	// Quick exit if we are fetching zero items
	if limit == 0 {
		return nil, nil
	}

	// Set "unlimited" limit to max int value
	if limit < 0 {
		limit = maxInt
	}

	// Prepare a slice to store the results in
	buf := resultBufSize
	if limit > 0 && limit < resultBufSize {
		buf = limit
	}
	result := make([]string, 0, buf)

	queryPrefix := iq.Index.getQuery(iq.KeyPrefix)
	qplen := len(queryPrefix)

	filter := iq.FilterKeys
	namelen := len(iq.Index.Name) + 1

	if err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		opts.Reverse = iq.Reverse
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Seek(queryPrefix); it.ValidForPrefix(queryPrefix); it.Next() {
			k := it.Item().Key()
			idx := bytes.LastIndexByte(k, idSeparator)
			if idx < 0 {
				return fmt.Errorf("index entry [%s] is invalid", k)
			}
			// Validate that a query with ?-mark isn't mistaken for a hit
			// when matching the ? separator for the resource ID.
			if qplen > idx {
				continue
			}

			// If we have a key filter, validate against it
			if filter != nil {
				if !filter(k[namelen:idx]) {
					continue
				}
			}

			// Skip until we reach the offset we are searching from
			if offset > 0 {
				offset--
				continue
			}

			// Add resource ID reference to result
			result = append(result, string(k[idx+1:]))

			limit--
			if limit == 0 {
				return nil
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return result, nil
}
