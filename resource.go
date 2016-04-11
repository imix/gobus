package main

type Resource struct {
	IsItem      bool
	Name        string
	Value       []byte
	ContentType string
	Children    []string
	Hooks       *HookCollection
}

type GoBusDB interface {
	CreateResource(elts []string, item bool) *Resource
	GetResource(elts []string) (*Resource, error)
	DeleteResource(elts []string) error
	ResourceSetValue(elts []string, contentType string, value []byte) error
	AddToCollection(elts []string, contentType string, data []byte) (string, error)
	AddHook(comps []string, data []byte) (string, error)
	DeleteHook(comps []string, cmds []string) error
	GetHooks(comps []string) ([]*Hook, error)
}
