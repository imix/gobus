package main

import (
	"errors"
	"fmt"
	"net/url"
	"path"
	"strconv"
)

type Resource struct {
	IsItem     bool
	Name       string
	URL        *url.URL
	Value      []byte
	Children   []*Resource
	Hooks      *HookCollection
	NextId     int
	NextHookId int
}

func newResource(name string, item bool, url *url.URL) *Resource {
	return &Resource{
		IsItem:   item,
		Name:     name,
		URL:      url,
		Children: make([]*Resource, 0),
		Hooks:    new(HookCollection),
		NextId:   0,
	}
}

func createResource(res *Resource, elts []string, item bool) *Resource {
	name := elts[0]
	var newURL = new(url.URL)
	*newURL = *res.URL
	newURL.Path = path.Join(newURL.Path, name)
	// XXX maye check whether the elements in elts dont exist already
	// createResource shouldn't be used that way but the interface is not too intuitive...
	// maybe refactor completely
	if len(elts) == 1 {
		newRes := newResource(name, item, newURL)
		res.Children = append(res.Children, newRes)
		return newRes
	} else {
		newRes := newResource(name, false, newURL) // intermediate resources are collections
		res.Children = append(res.Children, newRes)
		return createResource(newRes, elts[1:], item)
	}
}

func getResource(res *Resource, elts []string) (*Resource, []string) {
	if len(elts) == 0 {
		return res, elts
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
		return res, elts
	}
	return getResource(child, elts[1:])
}

// deletes a resource
// delete non-leaf resources generates an error
func deleteResource(res *Resource, elts []string) error {
	parent, _ := getResource(res, elts[:len(elts)-1])
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

func setValue(res *Resource, value []byte) {
	res.Value = value
}

func addToCollection(res *Resource, data []byte) string {
	name := strconv.Itoa(res.NextId)
	newRes := createResource(res, []string{name}, true)
	setValue(newRes, data)
	res.Children = append(res.Children, newRes)
	res.NextId += 1
	return name
}
