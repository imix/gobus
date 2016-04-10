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

// create HandlerDate for the Tests
func createHandlerData(t *testing.T, db GoBusDB, method, callUrl string, data io.Reader) *HandlerData {
	testURL, _ := url.Parse("http://localhost:8080/asdf/qwer")
	w := httptest.NewRecorder()
	r, err := http.NewRequest(method, callUrl, data)
	if err != nil {
		t.Fatal("Could not create request")
	}
	return &HandlerData{
		DB:      db,
		BaseURL: testURL,
		W:       w,
		R:       r,
	}
}

// check a ResponseRecorder for a satus and reports Error(msg) if not correct
func checkCode(t *testing.T, hd *HandlerData, code int, msg string) {
	gotCode := hd.W.(*httptest.ResponseRecorder).Code
	if gotCode != code {
		t.Error(fmt.Sprintf("Got code %d, msg: %s", gotCode, msg))
	}
}

func TestRespond(t *testing.T) {
	db := NewMemoryDB()
	hd := createHandlerData(t, db, "GET", "http://example.com/foo", nil)
	w := hd.W.(*httptest.ResponseRecorder)
	respond(w, hd.R, http.StatusNotFound, "a message")
	checkCode(t, hd, http.StatusNotFound, "Status not set")
	if !strings.Contains(w.Body.String(), "404") {
		t.Error("Body not set status")
	}
	if !strings.Contains(w.Body.String(), "a message") {
		t.Error("Body not set msg")
	}
}

func TestHandlePut(t *testing.T) {
	db := NewMemoryDB()
	resPath := []string{"path", "res"}
	res := db.CreateResource(resPath, true)

	// put to item
	data := strings.NewReader("some data")
	hd := createHandlerData(t, db, "PUT", "http://localhost:8080/asdf/qwer/path/res", data)
	handlePut(hd)
	checkCode(t, hd, http.StatusOK, "Put: 200 not working")
	if strings.Compare(string(res.Value), "some data") != 0 {
		t.Error("Put: Value not working")
	}
	// put to collection
	resPath = []string{"a", "collection"}
	res = db.CreateResource(resPath, false)

	data = strings.NewReader("some data")
	hd = createHandlerData(t, db, "PUT", "http://localhost:8080/asdf/qwer/a/collection", data)
	handlePut(hd)
	checkCode(t, hd, http.StatusConflict, "Put: 409 not working")

	// put to non-existens resource
	data = strings.NewReader("some data")
	hd = createHandlerData(t, db, "PUT", "http://localhost:8080/asdf/qwer/new/item", data)
	handlePut(hd)
	checkCode(t, hd, http.StatusCreated, "Put: 201 not working")
}

func TestHandlePost(t *testing.T) {
	db := NewMemoryDB()
	resPath := []string{"path", "res"}
	res := db.CreateResource(resPath, false)

	// post to existing collection
	data := strings.NewReader("some data")
	hd := createHandlerData(t, db, "POST", "http://localhost:8080/asdf/qwer/path/res", data)
	handlePost(hd)
	checkCode(t, hd, http.StatusCreated, "Post: 201 not working")
	location := hd.W.Header().Get("Location")
	if strings.Compare("http://localhost:8080/asdf/qwer/path/res/0", location) != 0 {
		t.Error("Post: Location wrong")
	}
	if strings.Compare(string(res.Children[0].Value), "some data") != 0 {
		t.Error("Post: Value wrong")
	}
	// post to inexisting resource
	hd = createHandlerData(t, db, "POST", "http://localhost:8080/asdf/qwer/uwld/ere/i", data)
	handlePost(hd)
	checkCode(t, hd, http.StatusNotFound, "Post: 404 not working")

	// post to item
	resPath = []string{"an", "item"}
	res = db.CreateResource(resPath, true)

	hd = createHandlerData(t, db, "POST", "http://localhost:8080/asdf/qwer/an/item", data)
	handlePost(hd)
	checkCode(t, hd, http.StatusConflict, "Post: 409 not working")
}

func TestHandlePostCommands(t *testing.T) {
	db := NewMemoryDB()
	resPath := []string{"an_item"}
	res := db.CreateResource(resPath, false)

	// start server for hook test
	c := make(chan []byte)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := ioutil.ReadAll(r.Body)
		c <- b
	}))
	defer ts.Close()

	// test hook created
	hookData := strings.NewReader(fmt.Sprintf(`{"name": "hook_name", "url": "%s"}`, ts.URL))
	hd := createHandlerData(t, db, "POST", "http://localhost:8080/asdf/qwer/an_item/_hooks", hookData)
	handlePost(hd)
	checkCode(t, hd, http.StatusCreated, "Post Hook: 201 not working")
	if !strings.Contains(res.Hooks.Hooks[0].Name, "hook_name") {
		t.Error("Post Hook: Content not working")
	}

	// test hook called
	postdata := strings.NewReader("any data")
	hd = createHandlerData(t, db, "POST", "http://localhost:8080/asdf/qwer/an_item", postdata)
	handlePost(hd)
	var data []byte
	data = <-c
	var hookevent HookEvent
	err := json.Unmarshal(data, &hookevent)
	if err != nil {
		t.Error(err)
	}
}

func TestHandleDelete(t *testing.T) {
	db := NewMemoryDB()
	resPath := []string{"path", "res"}
	db.CreateResource(resPath, true)

	// delete intermediate resource
	hd := createHandlerData(t, db, "DELETE", "http://localhost:8080/asdf/qwer/path", nil)
	handleDelete(hd)
	checkCode(t, hd, http.StatusNotFound, "Get: 404 not working")

	// delete existing resource
	hd = createHandlerData(t, db, "DELETE", "http://localhost:8080/asdf/qwer/path/res", nil)
	handleDelete(hd)
	checkCode(t, hd, http.StatusOK, "Delete: 200 not working")
	deletedRes, _ := db.GetResource(resPath)
	if strings.Compare(deletedRes.Name, "path") != 0 {
		t.Error("Delete not working")
	}

	// delete a not-existing resource
	hd = createHandlerData(t, db, "DELETE", "http://localhost:8080/asdf/qwer/asdfas/res", nil)
	handleDelete(hd)
	checkCode(t, hd, http.StatusNotFound, "Get: 404 not working")
}

func TestHandleDeleteCommands(t *testing.T) {
	db := NewMemoryDB()
	resPath := []string{"path", "res"}
	res := db.CreateResource(resPath, true)
	hookData := []byte(fmt.Sprintf(`{"name": "a_hook", "url": "http://test.com/a/hook"}`))
	db.AddHook(resPath, hookData)

	// delete unknown hook
	hd := createHandlerData(t, db, "DELETE", "http://localhost:8080/asdf/qwer/path/res/_hooks/1", nil)
	handleDelete(hd)
	checkCode(t, hd, http.StatusNotFound, "Delete Unknown: 404 not working")

	// delete weird path
	hd = createHandlerData(t, db, "DELETE", "http://localhost:8080/asdf/qwer/path/res/_hooks/0/gugu", nil)
	handleDelete(hd)
	checkCode(t, hd, http.StatusNotFound, "Delete Weird: 404 not working")

	// now delete the hook
	hd = createHandlerData(t, db, "DELETE", "http://localhost:8080/asdf/qwer/path/res/_hooks/0", nil)
	handleDelete(hd)
	checkCode(t, hd, http.StatusOK, "Delete Hook: 200 not working")
	if len(res.Hooks.Hooks) > 0 {
		t.Error("Delete Hook not working")
	}
}

func TestHandleGet(t *testing.T) {
	db := NewMemoryDB()
	resPath := []string{"path", "res"}
	db.CreateResource(resPath, true)

	// request an existing resource
	hd := createHandlerData(t, db, "GET", "http://localhost:8080/asdf/qwer/path/res", nil)
	db.ResourceSetValue(resPath, []byte("blup"))
	handleGet(hd)
	checkCode(t, hd, http.StatusOK, "Get: 200 not working")
	if !strings.Contains(hd.W.(*httptest.ResponseRecorder).Body.String(), "blup") {
		t.Error("Get: content not set")
	}

	// request a not-existing resource
	hd = createHandlerData(t, db, "GET", "http://localhost:8080/asdf/qwer/asdfas/res", nil)
	handleGet(hd)
	checkCode(t, hd, http.StatusNotFound, "Get: 404 not working")
}

func TestHandleGetHooks(t *testing.T) {
	db := NewMemoryDB()
	resPath := []string{"path", "res"}
	db.CreateResource(resPath, false)
	hookData := []byte(fmt.Sprintf(`{"name": "a_hook", "url": "http://test.com/a/hook"}`))
	db.AddHook(resPath, hookData)

	hd := createHandlerData(t, db, "GET", "http://localhost:8080/asdf/qwer/path/res/_hooks", nil)
	handleGet(hd)
	checkCode(t, hd, http.StatusOK, "Get hooks: 200 not working")
	if !strings.Contains(hd.W.(*httptest.ResponseRecorder).Body.String(), "test.com") {
		t.Error("Get: content not set")
	}
}
