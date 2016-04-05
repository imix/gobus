package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
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
	ModifiedResource string `json:"url"`
}

func parseHook(data []byte) (*Hook, error) {
	var hook Hook
	err := json.Unmarshal(data, &hook)
	return &hook, err

}

func addHook(res *Resource, data []byte) (string, error) {
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

func deleteHook(res *Resource, id string) error {
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

func callHooks(res *Resource, method string) {
	for _, h := range res.Hooks.Hooks {
		var event = HookEvent{
			Name:             h.Name,
			Method:           method,
			Item:             res.IsItem,
			ModifiedResource: res.URL.String(),
		}
		data, err := json.Marshal(event)
		if err != nil {
			log.Printf("Failed to marshal hook %s", h.Name)
			continue
		}
		go http.Post(h.URL, "application/json", bytes.NewReader(data))
	}
}

func getHooksJson(res *Resource) ([]byte, error) {
	return json.Marshal(res.Hooks)
}
