package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// creates a new recorder and a request
func setupResponseRequest(
	t *testing.T, method,
	url string, data io.Reader) (*httptest.ResponseRecorder, *http.Request) {

	w := httptest.NewRecorder()
	r, err := http.NewRequest(method, url, data)
	if err != nil {
		t.Fatal("Could not create request")
	}
	return w, r
}

// check a ResponseRecorder for a satus and reports Error(msg) if not correct
func checkCode(t *testing.T, w *httptest.ResponseRecorder, code int, msg string) {
	if w.Code != code {
		t.Error(msg)
	}
}

func TestRespond(t *testing.T) {
	w, r := setupResponseRequest(t, "GET", "http://example.com/foo", nil)
	respond(w, r, http.StatusNotFound, "a message")
	checkCode(t, w, http.StatusNotFound, "Status not set")
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
	data := strings.NewReader("some data")
	w, r := setupResponseRequest(t, "PUT", "http://localhost:8080/asdf/qwer/path/res", data)
	handlePut(db, w, r)
	checkCode(t, w, http.StatusOK, "Put: 200 not working")
	if strings.Compare(string(res.Value), "some data") != 0 {
		t.Error("Put: Value not working")
	}
	// put to collection
	resPath = []string{"a", "collection"}
	res = createResource(db, resPath, false)

	data = strings.NewReader("some data")
	w, r = setupResponseRequest(t, "PUT", "http://localhost:8080/asdf/qwer/a/collection", data)
	handlePut(db, w, r)
	checkCode(t, w, http.StatusNotFound, "Put: 404 not working")

	// put to non-existens resource
	data = strings.NewReader("some data")
	w, r = setupResponseRequest(t, "PUT", "http://localhost:8080/asdf/qwer/new/item", data)
	handlePut(db, w, r)
	checkCode(t, w, http.StatusCreated, "Put: 201 not working")
}

func TestHandlePost(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:8080/asdf/qwer")
	db := newResource("root", false, testURL)
	resPath := []string{"path", "res"}
	res := createResource(db, resPath, false)

	// post to existing collection
	data := strings.NewReader("some data")
	w, r := setupResponseRequest(t, "POST", "http://localhost:8080/asdf/qwer/path/res", data)
	handlePost(db, w, r)
	checkCode(t, w, http.StatusCreated, "Post: 201 not working")
	location := w.Header().Get("Location")
	if strings.Compare("http://localhost:8080/asdf/qwer/path/res/0", location) != 0 {
		t.Error("Post: Location wrong")
	}
	if strings.Compare(string(res.Children[0].Value), "some data") != 0 {
		t.Error("Post: Value wrong")
	}
	// post to inexisting resource
	w, r = setupResponseRequest(t, "POST", "http://localhost:8080/asdf/qwer/uwld/ere/i", data)
	handlePost(db, w, r)
	checkCode(t, w, http.StatusNotFound, "Post: 404 not working")

	// post to item
	resPath = []string{"an", "item"}
	res = createResource(db, resPath, true)

	w, r = setupResponseRequest(t, "POST", "http://localhost:8080/asdf/qwer/an/item", data)
	handlePost(db, w, r)
	checkCode(t, w, http.StatusConflict, "Post: 409 not working")
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
	hookData := strings.NewReader(fmt.Sprintf(`{"name": "hook_name", "url": "%s"}`, ts.URL))
	w, r := setupResponseRequest(t, "POST", "http://localhost:8080/asdf/qwer/an_item/_hooks", hookData)
	handlePost(db, w, r)
	checkCode(t, w, http.StatusCreated, "Post Hook: 201 not working")
	if !strings.Contains(res.Hooks.Hooks[0].Name, "hook_name") {
		t.Error("Post Hook: Content not working")
	}

	// test hook called
	postdata := strings.NewReader("any data")
	w, r = setupResponseRequest(t, "POST", "http://localhost:8080/asdf/qwer/an_item", postdata)
	handlePost(db, w, r)
	var data []byte
	data = <-c
	var hookevent HookEvent
	err := json.Unmarshal(data, &hookevent)
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
	w, r := setupResponseRequest(t, "DELETE", "http://localhost:8080/asdf/qwer/path", nil)
	handleDelete(db, w, r)
	checkCode(t, w, http.StatusNotFound, "Get: 404 not working")

	// delete existing resource
	w, r = setupResponseRequest(t, "DELETE", "http://localhost:8080/asdf/qwer/path/res", nil)
	handleDelete(db, w, r)
	checkCode(t, w, http.StatusOK, "Delete: 200 not working")
	deletedRes, _ := getResource(db, resPath)
	if strings.Compare(deletedRes.Name, "path") != 0 {
		t.Error("Delete not working")
	}

	// delete a not-existing resource
	w, r = setupResponseRequest(t, "DELETE", "http://localhost:8080/asdf/qwer/asdfas/res", nil)
	handleDelete(db, w, r)
	checkCode(t, w, http.StatusNotFound, "Get: 404 not working")
}

func TestHandleDeleteCommands(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:8080/asdf/qwer")
	db := newResource("root", false, testURL)
	resPath := []string{"path", "res"}
	res := createResource(db, resPath, true)
	hookData := []byte(fmt.Sprintf(`{"name": "a_hook", "url": "http://test.com/a/hook"}`))
	addHook(res, hookData)

	// delete unknown hook
	w, r := setupResponseRequest(t, "DELETE", "http://localhost:8080/asdf/qwer/path/res/_hooks/1", nil)
	handleDelete(db, w, r)
	checkCode(t, w, http.StatusNotFound, "Get: 404 not working")

	// delete weird path
	w, r = setupResponseRequest(t, "DELETE", "http://localhost:8080/asdf/qwer/path/res/_hooks/0/gugu", nil)
	handleDelete(db, w, r)
	checkCode(t, w, http.StatusNotFound, "Get: 404 not working")

	// now delete the hook
	w, r = setupResponseRequest(t, "DELETE", "http://localhost:8080/asdf/qwer/path/res/_hooks/0", nil)
	handleDelete(db, w, r)
	checkCode(t, w, http.StatusOK, "Delete: 200 not working")
	if len(res.Hooks.Hooks) > 0 {
		t.Error("Delete Hook not working")
	}
}

func TestHandleGet(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:8080/asdf/qwer")
	db := newResource("root", false, testURL)
	res := createResource(db, []string{"path", "res"}, true)

	// request an existing resource
	w, r := setupResponseRequest(t, "GET", "http://localhost:8080/asdf/qwer/path/res", nil)
	setValue(res, []byte("blup"))
	handleGet(db, w, r)
	checkCode(t, w, http.StatusOK, "Get: 200 not working")
	if !strings.Contains(w.Body.String(), "blup") {
		t.Error("Get: content not set")
	}

	// request a not-existing resource
	w, r = setupResponseRequest(t, "GET", "http://localhost:8080/asdf/qwer/asdfas/res", nil)
	handleGet(db, w, r)
	checkCode(t, w, http.StatusNotFound, "Get: 404 not working")
}

func TestHandleGetHooks(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:8080/asdf/qwer")
	db := newResource("root", false, testURL)
	res := createResource(db, []string{"path", "res"}, false)
	hookData := []byte(fmt.Sprintf(`{"name": "a_hook", "url": "http://test.com/a/hook"}`))
	addHook(res, hookData)

	w, r := setupResponseRequest(t, "GET", "http://localhost:8080/asdf/qwer/path/res/_hooks", nil)
	handleGet(db, w, r)
	checkCode(t, w, http.StatusOK, "Get hooks: 200 not working")
	if !strings.Contains(w.Body.String(), "test.com") {
		t.Error("Get: content not set")
	}
}
