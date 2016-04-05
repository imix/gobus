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

func TestRespond(t *testing.T) {
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "http://example.com/foo", nil)
	if err != nil {
		t.Fatal("Could not create request")
	}
	respond(w, r, http.StatusNotFound, "a message")
	if w.Code != http.StatusNotFound {
		t.Error("Status not set")
	}
	if !strings.Contains(w.Body.String(), "404") {
		t.Error("Body not set status")
	}
	if !strings.Contains(w.Body.String(), "a message") {
		t.Error("Body not set msg")
	}
}

func TestHandlePut(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:8080/asdf/qwer")
	db := newResource("root", false, testURL)
	resPath := []string{"path", "res"}
	res := createResource(db, resPath, true)

	// put to item
	w := httptest.NewRecorder()
	data := strings.NewReader("some data")
	r, err := http.NewRequest("PUT", "http://localhost:8080/asdf/qwer/path/res", data)
	if err != nil {
		t.Fatal("Could not create request")
	}
	handlePut(db, w, r)
	if w.Code != http.StatusOK {
		t.Error("Put: 200 not working")
	}
	if strings.Compare(string(res.Value), "some data") != 0 {
		t.Error("Put: Value not working")
	}
	// put to collection
	resPath = []string{"a", "collection"}
	res = createResource(db, resPath, false)

	w = httptest.NewRecorder()
	data = strings.NewReader("some data")
	r, err = http.NewRequest("PUT", "http://localhost:8080/asdf/qwer/a/collection", data)
	if err != nil {
		t.Fatal("Could not create request")
	}
	handlePut(db, w, r)
	if w.Code != http.StatusNotFound {
		t.Error("Put: 404 not working")
	}

	// put to non-existens resource
	w = httptest.NewRecorder()
	data = strings.NewReader("some data")
	r, err = http.NewRequest("PUT", "http://localhost:8080/asdf/qwer/new/item", data)
	if err != nil {
		t.Fatal("Could not create request")
	}
	handlePut(db, w, r)
	if w.Code != http.StatusCreated {
		t.Error("Put: 201 not working")
	}
}

func TestHandlePost(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:8080/asdf/qwer")
	db := newResource("root", false, testURL)
	resPath := []string{"path", "res"}
	res := createResource(db, resPath, false)

	// post to existing collection
	w := httptest.NewRecorder()
	data := strings.NewReader("some data")
	r, err := http.NewRequest("POST", "http://localhost:8080/asdf/qwer/path/res", data)
	if err != nil {
		t.Fatal("Could not create request")
	}
	handlePost(db, w, r)
	if w.Code != http.StatusCreated {
		t.Error("Post: 201 not working")
	}
	location := w.Header().Get("Location")
	if strings.Compare("http://localhost:8080/asdf/qwer/path/res/0", location) != 0 {
		t.Error("Post: Location wrong")
	}
	if strings.Compare(string(res.Children[0].Value), "some data") != 0 {
		t.Error("Post: Value wrong")
	}
	// post to inexisting resource
	w = httptest.NewRecorder()
	r, err = http.NewRequest("POST", "http://localhost:8080/asdf/qwer/uwld/ere/i", data)
	if err != nil {
		t.Fatal("Could not create request")
	}
	handlePost(db, w, r)
	if w.Code != http.StatusNotFound {
		t.Error("Post: 404 not working")
	}

	// post to item
	resPath = []string{"an", "item"}
	res = createResource(db, resPath, true)

	w = httptest.NewRecorder()
	r, err = http.NewRequest("POST", "http://localhost:8080/asdf/qwer/an/item", data)
	if err != nil {
		t.Fatal("Could not create request")
	}
	handlePost(db, w, r)
	if w.Code != http.StatusConflict {
		t.Error("Post: 409 not working")
	}
}

func TestHandlePostCommands(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:8080/asdf/qwer")
	db := newResource("root", false, testURL)
	resPath := []string{"an_item"}
	res := createResource(db, resPath, false)

	// start server for hook test
	c := make(chan []byte)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := ioutil.ReadAll(r.Body)
		c <- b
	}))
	defer ts.Close()

	// test hook created
	w := httptest.NewRecorder()
	hookData := strings.NewReader(fmt.Sprintf(`{"name": "hook_name", "url": "%s"}`, ts.URL))
	r, err := http.NewRequest("POST", "http://localhost:8080/asdf/qwer/an_item/_hooks", hookData)
	if err != nil {
		t.Fatal("Could not create request")
	}
	handlePost(db, w, r)
	if w.Code != http.StatusCreated {
		t.Error("Post Hook: 201 not working")
	}
	if !strings.Contains(res.Hooks.Hooks[0].Name, "hook_name") {
		t.Error("Post Hook: Content not working")
	}

	// test hook called
	w = httptest.NewRecorder()
	postdata := strings.NewReader("any data")
	r, err = http.NewRequest("POST", "http://localhost:8080/asdf/qwer/an_item", postdata)
	if err != nil {
		t.Fatal("Could not create request")
	}
	handlePost(db, w, r)
	var data []byte
	data = <-c
	var hookevent HookEvent
	err = json.Unmarshal(data, &hookevent)
	if err != nil {
		t.Error(err)
	}
}

func TestHandleDelete(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:8080/asdf/qwer")
	db := newResource("root", false, testURL)
	resPath := []string{"path", "res"}
	createResource(db, resPath, true)

	// delete intermediate resource
	w := httptest.NewRecorder()
	r, err := http.NewRequest("DELETE", "http://localhost:8080/asdf/qwer/path", nil)
	if err != nil {
		t.Fatal("Could not create request")
	}
	handleDelete(db, w, r)
	if w.Code != http.StatusNotFound {
		t.Error("Get: 404 not working")
	}

	// delete existing resource
	w = httptest.NewRecorder()
	r, err = http.NewRequest("DELETE", "http://localhost:8080/asdf/qwer/path/res", nil)
	if err != nil {
		t.Fatal("Could not create request")
	}
	handleDelete(db, w, r)
	if w.Code != http.StatusOK {
		t.Error("Delete: 200 not working")
	}
	deletedRes, _ := getResource(db, resPath)
	if strings.Compare(deletedRes.Name, "path") != 0 {
		t.Error("Delete not working")
	}

	// delete a not-existing resource
	w = httptest.NewRecorder()
	r, err = http.NewRequest("DELETE", "http://localhost:8080/asdf/qwer/asdfas/res", nil)
	if err != nil {
		t.Fatal("Could not create request")
	}
	handleDelete(db, w, r)
	if w.Code != http.StatusNotFound {
		t.Error("Get: 404 not working")
	}
}

func TestHandleDeleteCommands(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:8080/asdf/qwer")
	db := newResource("root", false, testURL)
	resPath := []string{"path", "res"}
	res := createResource(db, resPath, true)
	hookData := []byte(fmt.Sprintf(`{"name": "a_hook", "url": "http://test.com/a/hook"}`))
	addHook(res, hookData)

	// delete unknown hook
	w := httptest.NewRecorder()
	r, err := http.NewRequest("DELETE", "http://localhost:8080/asdf/qwer/path/res/_hooks/1", nil)
	if err != nil {
		t.Fatal("Could not create request")
	}
	handleDelete(db, w, r)
	if w.Code != http.StatusNotFound {
		t.Error("Delete: 404 not working")
	}

	// delete weird path
	w = httptest.NewRecorder()
	r, err = http.NewRequest("DELETE", "http://localhost:8080/asdf/qwer/path/res/_hooks/0/gugu", nil)
	if err != nil {
		t.Fatal("Could not create request")
	}
	handleDelete(db, w, r)
	if w.Code != http.StatusNotFound {
		t.Error("Delete: 404 not working")
	}

	// now delete the hook
	w = httptest.NewRecorder()
	r, err = http.NewRequest("DELETE", "http://localhost:8080/asdf/qwer/path/res/_hooks/0", nil)
	if err != nil {
		t.Fatal("Could not create request")
	}
	handleDelete(db, w, r)
	if w.Code != http.StatusOK {
		t.Error("Delete: 200 not working")
	}
	if len(res.Hooks.Hooks) > 0 {
		t.Error("Delete Hook not working")
	}
}

func TestHandleGet(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:8080/asdf/qwer")
	db := newResource("root", false, testURL)
	res := createResource(db, []string{"path", "res"}, true)

	// request an existing resource
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "http://localhost:8080/asdf/qwer/path/res", nil)
	if err != nil {
		t.Fatal("Could not create request")
	}
	setValue(res, []byte("blup"))
	handleGet(db, w, r)
	if w.Code != http.StatusOK {
		t.Error("Get: 200 not working")
	}
	if !strings.Contains(w.Body.String(), "blup") {
		t.Error("Get: content not set")
	}

	// request a not-existing resource
	w = httptest.NewRecorder()
	r, err = http.NewRequest("GET", "http://localhost:8080/asdf/qwer/asdfas/res", nil)
	if err != nil {
		t.Fatal("Could not create request")
	}
	handleGet(db, w, r)
	if w.Code != http.StatusNotFound {
		t.Error("Get: 404 not working")
	}
}

func TestHandleGetHooks(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:8080/asdf/qwer")
	db := newResource("root", false, testURL)
	res := createResource(db, []string{"path", "res"}, false)
	hookData := []byte(fmt.Sprintf(`{"name": "a_hook", "url": "http://test.com/a/hook"}`))
	addHook(res, hookData)

	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "http://localhost:8080/asdf/qwer/path/res/_hooks", nil)
	if err != nil {
		t.Fatal("Could not create request")
	}
	handleGet(db, w, r)

	if w.Code != http.StatusOK {
		t.Error("Get hooks: 200 not working")
	}
	if !strings.Contains(w.Body.String(), "test.com") {
		t.Error("Get: content not set")
	}
}
