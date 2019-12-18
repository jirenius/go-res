package res

import "encoding/json"

// Ref is a resource reference to another resource ID.
// It marshals into a reference object, eg.:
//  {"rid":"userService.user.42"}
type Ref string

// ResourceType enum
type ResourceType byte

// Resource type enum values
const (
	TypeUnset ResourceType = iota
	TypeModel
	TypeCollection
)

var refPrefix = []byte(`{"rid":`)

const refSuffix = '}'

// DeleteAction is used for deleted properties in "change" events
var DeleteAction = &struct{ json.RawMessage }{RawMessage: json.RawMessage(`{"action":"delete"}`)}

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

// UnmarshalJSON makes Ref implement the json.Unmarshaler interface.
func (r *Ref) UnmarshalJSON(b []byte) error {
	var p struct {
		RID string `json:"rid"`
	}
	err := json.Unmarshal(b, &p)
	if err != nil {
		return err
	}
	*r = Ref(p.RID)
	return nil
}

// IsValid returns true if the reference RID is valid, otherwise false.
func (r Ref) IsValid() bool {
	start := true
	for _, c := range r {
		if c == '?' {
			return !start
		}
		if c < 33 || c > 126 || c == '*' || c == '>' {
			return false
		}
		if c == '.' {
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
