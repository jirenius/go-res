package res

import "encoding/json"

// Ref is a resource reference to another resource ID.
// It marshals into a reference object, eg.:
//  {"rid":"userService.user.42"}
type Ref string

// Resource type enum
type ResourceType byte

const (
	TypeUnset ResourceType = iota
	TypeModel
	TypeCollection
)

var refPrefix = []byte(`{"rid":`)

const refSuffix = '}'

// DeleteAction is used for deleted properties in "change" events
var DeleteAction = json.RawMessage(`{"action":"delete"}`)

// MarshalJSON makes Ref implement the json.Marshaler interface.
func (r Ref) MarshalJSON() ([]byte, error) {
	rid, err := json.Marshal(string(r))
	if err != nil {
		return nil, err
	}
	o := make([]byte, len(rid)+8)
	copy(o, refPrefix)
	copy(o[7:], rid)
	o[len(o)-1] = refSuffix
	return o, nil
}

// IsValid returns true if the reference RID is valid, otherwise false.
func (r Ref) IsValid() bool {
	start := true
	for _, r := range r {
		if r == '?' {
			return !start
		}
		if r < 33 || r > 126 || r == '*' || r == '>' {
			return false
		}
		if r == '.' {
			if start {
				return false
			}
			start = true
		} else {
			start = false
		}
	}

	return !start
}
