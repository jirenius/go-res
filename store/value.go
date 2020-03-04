package store

import (
	"bytes"
	"encoding/json"
	"errors"

	res "github.com/jirenius/go-res"
)

// ValueType is an enum reprenting the value type
type ValueType byte

// Value type constants
const (
	ValueTypeNone ValueType = iota
	ValueTypePrimitive
	ValueTypeResource
	ValueTypeDelete
)

// valueObject represents a resource reference or an action
type valueObject struct {
	RID    *string `json:"rid"`
	Action *string `json:"action"`
}

// DeleteValue is a predeclared delete action value
var DeleteValue = Value{
	RawMessage: json.RawMessage(`{"action":"delete"}`),
	Type:       ValueTypeDelete,
}

// Value represents a RES value
// https://github.com/resgateio/resgate/blob/master/docs/res-protocol.md#values
type Value struct {
	json.RawMessage
	Type ValueType
	RID  string
}

var errInvalidValue = errors.New("invalid value")

const (
	actionDelete = "delete"
)

// UnmarshalJSON sets *v to the RES value represented by the JSON encoded data
func (v *Value) UnmarshalJSON(data []byte) error {
	err := v.RawMessage.UnmarshalJSON(data)
	if err != nil {
		return err
	}

	// Get first non-whitespace character
	var c byte
	i := 0
	for {
		c = v.RawMessage[i]
		if c != 0x20 && c != 0x09 && c != 0x0A && c != 0x0D {
			break
		}
		i++
	}

	switch c {
	case '{':
		var mvo valueObject
		err = json.Unmarshal(v.RawMessage, &mvo)
		if err != nil {
			return err
		}

		if mvo.RID != nil {
			// Invalid to have both RID and Action set, or if RID is empty
			if mvo.Action != nil || *mvo.RID == "" {
				return errInvalidValue
			}
			v.Type = ValueTypeResource
			v.RID = *mvo.RID
			if !res.Ref(v.RID).IsValid() {
				return errInvalidValue
			}
		} else {
			// Must be an action of type actionDelete
			if mvo.Action == nil || *mvo.Action != actionDelete {
				return errInvalidValue
			}
			v.Type = ValueTypeDelete
		}
	case '[':
		return errInvalidValue
	default:
		v.Type = ValueTypePrimitive
	}

	return nil
}

// MarshalJSON returns the embedded json.RawMessage as the JSON encoding.
func (v Value) MarshalJSON() ([]byte, error) {
	if v.RawMessage == nil {
		return []byte("null"), nil
	}
	return v.RawMessage, nil
}

// Equal reports whether v and w is equal in type and value
func (v Value) Equal(w Value) bool {
	if v.Type != w.Type {
		return false
	}

	switch v.Type {
	case ValueTypePrimitive:
		return bytes.Equal(v.RawMessage, w.RawMessage)
	case ValueTypeResource:
		return v.RID == w.RID
	}

	return true
}
