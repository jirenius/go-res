package res

type Middlewares struct {
	Access      []func(AccessHandler) AccessHandler
	GetResource []func(GetHandler) GetHandler
	Call        []func(CallHandler) CallHandler
	New         []func(NewHandler) NewHandler
	Auth        []func(AuthHandler) AuthHandler
	ApplyChange []func(ApplyChangeHandler) ApplyChangeHandler
	ApplyAdd    []func(ApplyAddHandler) ApplyAddHandler
	ApplyRemove []func(ApplyRemoveHandler) ApplyRemoveHandler
	ApplyCreate []func(ApplyCreateHandler) ApplyCreateHandler
	ApplyDelete []func(ApplyDeleteHandler) ApplyDeleteHandler
}
