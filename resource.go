package main

type GoBusDB interface {
	CreateResource(elts []string, item bool) (Resource, error)
	GetResource(elts []string) (Resource, error)
	DeleteResource(elts []string) error
	ResourceExists(elts []string) (bool, error)
}

type Resource interface {
	Name() (string, error)
	IsItem() (bool, error)
	GetValue() (string, []byte, error)
	SetValue(contentType string, value []byte) error
	GetChildren() ([]string, error)
	AddToCollection(contentType string, data []byte) (string, error)
	AddHook(data []byte) (string, error)
	DeleteHook(id string) error
	GetHooks() ([]*Hook, error)
}
