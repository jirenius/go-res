package res

type ObserveEvent struct {
	// Resource ID
	RID string

	// Path parameters parsed from the resource ID
	PathParams map[string]string

	// Name of the event, such as "change", "add", "remove", or custom.
	Name string

	// Event payload
	Payload interface{}
}
