package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
)

type HookCollection struct {
	Hooks      []*Hook `json:"hooks"`
	NextHookId int     `json:"-"`
}

type Hook struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
	//Fields      []string // limit fields to get
	//Methods     []string // limit methods to forward
}

type HookEvent struct {
	Name             string `json:"name"`
	Method           string `json:"method"`
	Item             bool   `json:"item"` // is the affected resource an item or a collection
	ModifiedResource string `json:"path"` // relative path to the resource (caller needs to know server)
}

func parseHook(data []byte) (*Hook, error) {
	var hook Hook
	err := json.Unmarshal(data, &hook)
	return &hook, err

}

func (db *MemoryDB) AddHook(comps []string, data []byte) (string, error) {
	res, err := db.GetResource(comps)
	if err != nil {
		return "", err
	}
	hook, err := parseHook(data)
	if err != nil {
		return "", err
	}
	id := strconv.Itoa(res.Hooks.NextHookId)
	hook.Id = id
	res.Hooks.Hooks = append(res.Hooks.Hooks, hook)
	res.Hooks.NextHookId += 1
	return hook.Id, nil
}

func (db *MemoryDB) DeleteHook(comps []string, cmds []string) error {
	res, err := db.GetResource(comps)
	if err != nil {
		return err
	}
	if len(cmds) != 2 {
		return errors.New("Path not correct for delete")
	}
	id := cmds[1]
	var idx int = -1
	for i, h := range res.Hooks.Hooks {
		if h.Id == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return errors.New(fmt.Sprintf("Hook does not exist: %s", id))
	}
	a := res.Hooks.Hooks
	a[idx] = a[len(a)-1]
	a[len(a)-1] = nil
	res.Hooks.Hooks = a[:len(a)-1]
	return nil
}

func (db *MemoryDB) GetHooks(comps []string) ([]*Hook, error) {
	res, err := db.GetResource(comps)
	if err != nil {
		return nil, err
	}
	return res.Hooks.Hooks, nil
}
