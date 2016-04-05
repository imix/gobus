package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
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
	testURL, _ := url.Parse("http://localhost:8080/asdf/qwer")
	root := newResource("root", false, testURL)
	hookData := []byte(`{"name": "hook_name", "url": "http://www.test.ch/my/resource"}`)
	name, err := addHook(root, hookData)
	if err != nil {
		t.Error("Hook Add failed", err)
	}
	if strings.Compare(name, "0") != 0 {
		t.Error("Hook Id not set")
	}
}

func TestDeleteHook(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:8080/asdf/qwer")
	root := newResource("root", false, testURL)
	hookData := []byte(`{"name": "hook_name", "url": "http://www.test.ch/my/resource"}`)
	name, _ := addHook(root, hookData)
	err := deleteHook(root, name)
	if err != nil {
		t.Error("Hook Delete failed", err)
	}

	err = deleteHook(root, name)
	if err == nil {
		t.Error("Hook Delete Inexisting failed:", err)
	}
}

func TestCallHook(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:8080/asdf/qwer")
	root := newResource("root", false, testURL)

	c := make(chan []byte)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := ioutil.ReadAll(r.Body)
		c <- b
	}))
	defer ts.Close()

	hookData := []byte(fmt.Sprintf(`{"name": "hook_name", "url": "%s"}`, ts.URL))
	_, err := addHook(root, hookData)
	if err != nil {
		t.Error(err)
	}
	callHooks(root, "POST")
	var data []byte
	data = <-c
	var hookevent HookEvent
	err = json.Unmarshal(data, &hookevent)
	if err != nil {
		t.Error(err)
	}
}
