package res

import "encoding/json"

// Ref is a resource reference to another resource ID.
// It marshals into a reference object, eg.:
//  {"rid":"userService.user.42"}
type Ref string

var refPrefix = []byte(`{"rid":`)

// AddEvent is used as event payload on "add" events
type AddEvent struct {
	Value interface{} `json:"value"`
	Idx   int         `json:"idx"`
}

// RemoveEvent is used as event payload on "remove" events
type RemoveEvent struct {
	Idx int `json:"idx"`
}

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
	o[len(o)-1] = '}'
	return o, nil
}
