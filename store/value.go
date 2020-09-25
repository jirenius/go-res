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
	ValueTypeReference
	ValueTypeSoftReference
	ValueTypeData
	ValueTypeDelete

	// Deprecated and replaced with ValueTypeReference.
	ValueTypeResource = ValueTypeReference
)

// valueObject represents a resource reference or an action
type valueObject struct {
	RID    *string         `json:"rid"`
	Soft   bool            `json:"soft"`
	Action *string         `json:"action"`
	Data   json.RawMessage `json:"data"`
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
	Type  ValueType
	RID   string
	Inner json.RawMessage
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

		switch {
		case mvo.RID != nil:
			// Invalid to have both RID and Action or Data set, or if RID is empty
			if mvo.Action != nil || mvo.Data != nil || *mvo.RID == "" {
				return errInvalidValue
			}
			v.RID = *mvo.RID
			if !res.Ref(v.RID).IsValid() {
				return errInvalidValue
			}
			if mvo.Soft {
				v.Type = ValueTypeSoftReference
			} else {
				v.Type = ValueTypeReference
			}

		case mvo.Action != nil:
			// Invalid to have both Action and Data set, or if action is not actionDelete
			if mvo.Data != nil || *mvo.Action != actionDelete {
				return errInvalidValue
			}
			v.Type = ValueTypeDelete

		case mvo.Data != nil:
			v.Inner = mvo.Data
			dc := mvo.Data[0]
			// Is data containing a primitive?
			if dc == '{' || dc == '[' {
				v.Type = ValueTypeData
			} else {
				v.RawMessage = mvo.Data
				v.Type = ValueTypePrimitive
			}

		default:
			return errInvalidValue
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
	case ValueTypeData:
		return bytes.Equal(v.Inner, w.Inner)
	case ValueTypePrimitive:
		return bytes.Equal(v.RawMessage, w.RawMessage)
	case ValueTypeReference:
		fallthrough
	case ValueTypeSoftReference:
		return v.RID == w.RID
	}

	return true
}
