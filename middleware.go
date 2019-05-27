package res

type Middlewares struct {
	Access      []func(AccessHandler) AccessHandler
	Get         []func(GetHandler) GetHandler
	Call        []func(CallHandler) CallHandler
	New         []func(NewHandler) NewHandler
	Auth        []func(AuthHandler) AuthHandler
	ApplyChange []func(ApplyChangeHandler) ApplyChangeHandler
	ApplyAdd    []func(ApplyAddHandler) ApplyAddHandler
	ApplyRemove []func(ApplyRemoveHandler) ApplyRemoveHandler
	ApplyCreate []func(ApplyCreateHandler) ApplyCreateHandler
	ApplyDelete []func(ApplyDeleteHandler) ApplyDeleteHandler
}

// onEvent []func(Resource, event string)
// onChange []func(Resource, change, rollback map[string]interface{})
// onAdd []func(Resource, v interface{}, idx int)
// onRemove []func(Resource, idx int, removed interface{})
// onCreate []func(Resource, resource interface{})
// onDelete []func(Resource, resource interface{})
