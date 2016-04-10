package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseHook(t *testing.T) {
	hookData := []byte(`{"name": "hook name", "url": "http://www.test.ch/my/resource"}`)
	hook, _ := parseHook(hookData)
	if strings.Compare(hook.Name, "hook name") != 0 {
		t.Error("Hook Name not set")
	}
	if strings.Compare(hook.URL, "http://www.test.ch/my/resource") != 0 {
		t.Error("Hook URL not set")
	}
}

func TestAddHook(t *testing.T) {
	db := NewMemoryDB()
	hookData := []byte(`{"name": "hook_name", "url": "http://www.test.ch/my/resource"}`)
	resPath := []string{"path"}
	db.CreateResource(resPath, true)
	name, err := db.AddHook(resPath, hookData)
	if err != nil {
		t.Error("Hook Add failed", err)
	}
	if strings.Compare(name, "0") != 0 {
		t.Error("Hook Id not set")
	}
}

func TestDeleteHook(t *testing.T) {
	db := NewMemoryDB()
	hookData := []byte(`{"name": "hook_name", "url": "http://www.test.ch/my/resource"}`)
	resPath := []string{"path"}
	db.CreateResource(resPath, true)
	name, _ := db.AddHook(resPath, hookData)
	cmds := []string{"_hooks", name}
	err := db.DeleteHook(resPath, cmds)
	if err != nil {
		t.Error("Hook Delete failed", err)
	}

	err = db.DeleteHook(resPath, cmds)
	if err == nil {
		t.Error("Hook Delete Inexisting failed:", err)
	}
}

func TestCallHook(t *testing.T) {
	db := NewMemoryDB()
	resPath := []string{"path"}
	db.CreateResource(resPath, true)

	c := make(chan []byte)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := ioutil.ReadAll(r.Body)
		c <- b
	}))
	defer ts.Close()

	hookData := []byte(fmt.Sprintf(`{"name": "hook_name", "url": "%s"}`, ts.URL))
	_, err := db.AddHook(resPath, hookData)
	if err != nil {
		t.Error(err)
	}
	hooks, _ := db.GetHooks(resPath)
	callHooks(hooks, "POST", true, "http://a_resource.com/res")
	var data []byte
	data = <-c
	var hookevent HookEvent
	err = json.Unmarshal(data, &hookevent)
	if err != nil {
		t.Error(err)
	}
}
