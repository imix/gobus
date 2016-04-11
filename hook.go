package main

import "encoding/json"

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
