package main

type GoBusDB interface {
	CreateResource(elts []string, item bool) (Resource, error)
	GetResource(elts []string) (Resource, error)
	ResourceExists(elts []string) (bool, error)
}

type Resource interface {
	Name() (string, error)
	Delete() error
	IsItem() (bool, error)
	GetElts() []string
	GetValue() (string, []byte, error)
	SetValue(contentType string, value []byte) error
	GetChildren() ([]string, error)
	AddToCollection(contentType string, data []byte) (string, error)
	SetHook(id string, data []byte) error
	AddHook(data []byte) (string, error)
	DeleteHook(id string) error
	GetHook(id string) (*Hook, error)
	GetHooks() ([]*Hook, error)
	GetHooksIDs() ([]string, error)
	DeleteForward() error
	GetForward() (*Forward, error)
	AddForward(data []byte) error
}
