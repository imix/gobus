package main

import (
	"errors"
	"fmt"
	"strconv"
)

type Resource struct {
	IsItem     bool
	Name       string
	Value      []byte
	Children   []*Resource
	Hooks      *HookCollection
	NextId     int
	NextHookId int
}

type GoBusDB interface {
	CreateResource(elts []string, item bool) *Resource
	GetResource(elts []string) (*Resource, error)
	DeleteResource(elts []string) error
	ResourceSetValue(elts []string, value []byte) error
	AddToCollection(elts []string, data []byte) (string, error)
	AddHook(comps []string, data []byte) (string, error)
	DeleteHook(comps []string, cmds []string) error
	GetHooks(comps []string) ([]*Hook, error)
}

type MemoryDB struct {
	RootResource *Resource
}

func NewMemoryDB() GoBusDB {
	res := newResource("root", false)
	return &MemoryDB{RootResource: res}
}

func newResource(name string, item bool) *Resource {
	return &Resource{
		IsItem:   item,
		Name:     name,
		Children: make([]*Resource, 0),
		Hooks:    new(HookCollection),
		NextId:   0,
	}
}

// Creates the resource defined by the given path
// Missing intermediate resources are automatically created
// The item flag is set on the last resource
// If the resource exists already, an error is returned
func (db *MemoryDB) CreateResource(elts []string, item bool) *Resource {
	return createResourceGo(db.RootResource, elts, item)
}

// recursive helper for CreateResource
func createResourceGo(res *Resource, elts []string, item bool) *Resource {
	name := elts[0]
	var thisRes *Resource = nil
	for _, c := range res.Children {
		if c.Name == elts[0] {
			thisRes = c
			break
		}
	}
	if len(elts) == 1 {
		if thisRes != nil {
			return nil
		}
		newRes := newResource(name, item)
		res.Children = append(res.Children, newRes)
		return newRes
	}
	if thisRes == nil {
		newRes := newResource(name, false)
		res.Children = append(res.Children, newRes)
		thisRes = newRes
	}
	return createResourceGo(thisRes, elts[1:], item)
}

// searches the resource identified by the given path and returns it
// if the Resource could not be found returns an error
func (db *MemoryDB) GetResource(elts []string) (*Resource, error) {
	return getResourceGo(db.RootResource, elts)
}

func getResourceGo(res *Resource, elts []string) (*Resource, error) {
	if len(elts) == 0 {
		return res, nil
	}
	name := elts[0]

	var child *Resource = nil
	for _, c := range res.Children {
		if c.Name == name {
			child = c
			break
		}
	}
	if child == nil {
		return res, errors.New("Could not find resource")
	}
	return getResourceGo(child, elts[1:])
}

// deletes a resource
// delete non-leaf resources generates an error
func (db *MemoryDB) DeleteResource(elts []string) error {
	if len(elts) < 1 {
		return errors.New("Need path to find resource")
	}
	parent, err := db.GetResource(elts[:len(elts)-1])
	if err != nil {
		return err
	}
	name := elts[len(elts)-1]
	var idx int = -1
	for i, c := range parent.Children {
		if c.Name == name {
			idx = i
			if len(c.Children) > 0 {
				return errors.New("Can not delete non-leaf resource")
			}
			break
		}
	}
	if idx == -1 {
		return errors.New(fmt.Sprintf("Could not find resource: %s", name))
	}
	a := parent.Children
	a[idx] = a[len(a)-1]
	a[len(a)-1] = nil
	parent.Children = a[:len(a)-1]
	return nil
}

func (db *MemoryDB) ResourceSetValue(elts []string, value []byte) error {
	res, err := db.GetResource(elts)
	if err != nil {
		return nil
	}
	res.Value = value
	return nil
}

func (db *MemoryDB) AddToCollection(elts []string, data []byte) (string, error) {
	res, err := db.GetResource(elts)
	if err != nil {
		return "", err
	}
	if res.IsItem {
		return "", errors.New("Can not add to items")
	}
	name := strconv.Itoa(res.NextId)
	resPath := append(elts, name)
	newRes := db.CreateResource(resPath, true)
	db.ResourceSetValue(resPath, data)
	res.Children = append(res.Children, newRes)
	res.NextId += 1
	return name, nil
}
