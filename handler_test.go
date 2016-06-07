package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func teardownRedis(db GoBusDB) {
	db.(*RedisDB).Client.FlushDb()
}

//XXX test nested collections

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
	db := NewRedisDB()
	hd := createHandlerData(t, db, "GET", "http://example.com/foo", nil)
	w := hd.W.(*httptest.ResponseRecorder)
	respond(hd, http.StatusNotFound, "a message")
	checkCode(t, hd, http.StatusNotFound, "Status not set")
	if !strings.Contains(w.Body.String(), "404") {
		t.Error("Body not set status")
	}
	if !strings.Contains(w.Body.String(), "a message") {
		t.Error("Body not set msg")
	}
	teardownRedis(db)
}

func TestHandlePut(t *testing.T) {
	db := NewRedisDB()
	resPath := []string{"path", "res"}
	res, _ := db.CreateResource(resPath, true)

	// put to item
	data := strings.NewReader("some data àL")
	hd := createHandlerData(t, db, "PUT", "http://localhost:8080/asdf/qwer/path/res", data)
	handleRequest(hd)
	checkCode(t, hd, http.StatusOK, "Put: 200 not working")
	res, _ = db.GetResource(resPath)
	_, value, _ := res.GetValue()
	if strings.Compare(string(value), "some data àL") != 0 {
		t.Error("Put: Value not working")
	}
	// put to collection
	resPath = []string{"a", "collection"}
	res, _ = db.CreateResource(resPath, false)

	data = strings.NewReader("some data")
	hd = createHandlerData(t, db, "PUT", "http://localhost:8080/asdf/qwer/a/collection", data)
	handleRequest(hd)
	checkCode(t, hd, http.StatusMethodNotAllowed, "Put: 405 not working")

	// put to non-existens resource
	hd = createHandlerData(t, db, "PUT", "http://localhost:8080/asdf/qwer/new/item", data)
	handleRequest(hd)
	checkCode(t, hd, http.StatusCreated, "Put: 201 not working")
	teardownRedis(db)
}

func TestHandlePost(t *testing.T) {
	db := NewRedisDB()
	resPath := []string{"path", "res"}
	db.CreateResource(resPath, false)

	// post to existing collection
	data := strings.NewReader("some data")
	hd := createHandlerData(t, db, "POST", "http://localhost:8080/asdf/qwer/path/res", data)
	handleRequest(hd)
	checkCode(t, hd, http.StatusCreated, "Post: 201 not working")
	location := hd.W.Header().Get("Location")
	if strings.Compare("http://localhost:8080/asdf/qwer/path/res/0", location) != 0 {
		t.Error("Post: Location wrong")
	}
	child, err := db.GetResource(append(resPath, "0"))
	if err != nil {
		t.Fatal("Post: Could not get Resource")
	}
	_, value, _ := child.GetValue()
	if strings.Compare(string(value), "some data") != 0 {
		t.Error("Post: Value wrong")
	}
	// post to inexisting resource
	hd = createHandlerData(t, db, "POST", "http://localhost:8080/asdf/qwer/uwld/ere/i", data)
	handleRequest(hd)
	checkCode(t, hd, http.StatusNotFound, "Post: 404 not working")

	// post to item
	resPath = []string{"an", "item"}
	db.CreateResource(resPath, true)

	hd = createHandlerData(t, db, "POST", "http://localhost:8080/asdf/qwer/an/item", data)
	handleRequest(hd)
	checkCode(t, hd, http.StatusMethodNotAllowed, "Post: 405 not working")
	teardownRedis(db)
}

func TestHandleDelete(t *testing.T) {
	db := NewRedisDB()
	resPath := []string{"path", "res"}
	db.CreateResource(resPath, true)

	// delete intermediate resource
	hd := createHandlerData(t, db, "DELETE", "http://localhost:8080/asdf/qwer/path", nil)
	handleRequest(hd)
	checkCode(t, hd, http.StatusNotFound, "Delete intermdiate: 404 not working")

	// delete existing resource
	hd = createHandlerData(t, db, "DELETE", "http://localhost:8080/asdf/qwer/path/res", nil)
	handleRequest(hd)
	checkCode(t, hd, http.StatusOK, "Delete: 200 not working")
	_, err := db.GetResource(resPath)
	if err == nil {
		t.Error("Could not delete resource")
	}

	// delete a not-existing resource
	hd = createHandlerData(t, db, "DELETE", "http://localhost:8080/asdf/qwer/asdfas/res", nil)
	handleRequest(hd)
	checkCode(t, hd, http.StatusNotFound, "Delete inexisting: 404 not working")
	teardownRedis(db)
}

func TestHandleDeleteCommands(t *testing.T) {
	db := NewRedisDB()
	resPath := []string{"path", "res"}
	res, _ := db.CreateResource(resPath, true)
	hookData := []byte(fmt.Sprintf(`{"name": "a_hook", "url": "http://test.com/a/hook"}`))
	res.AddHook(hookData)

	// delete unknown hook
	hd := createHandlerData(t, db, "DELETE", "http://localhost:8080/asdf/qwer/path/res/_hooks/1", nil)
	handleRequest(hd)
	checkCode(t, hd, http.StatusNotFound, "Delete Unknown: 404 not working")

	// delete weird path
	hd = createHandlerData(t, db, "DELETE", "http://localhost:8080/asdf/qwer/path/res/_hooks/0/gugu", nil)
	handleRequest(hd)
	checkCode(t, hd, http.StatusNotFound, "Delete Weird: 404 not working")

	// now delete the hook
	hd = createHandlerData(t, db, "DELETE", "http://localhost:8080/asdf/qwer/path/res/_hooks/0", nil)
	handleRequest(hd)
	checkCode(t, hd, http.StatusOK, "Delete Hook: 200 not working")
	hooks, _ := res.GetHooks()
	if len(hooks) > 0 {
		t.Error("Delete Hook not working")
	}
	teardownRedis(db)
}

func TestHandleGet(t *testing.T) {
	db := NewRedisDB()
	resPath := []string{"path", "res"}
	res, _ := db.CreateResource(resPath, true)

	// request an existing resource
	hd := createHandlerData(t, db, "GET", "http://localhost:8080/asdf/qwer/path/res", nil)
	res.SetValue("text", []byte("blup"))
	handleRequest(hd)
	checkCode(t, hd, http.StatusOK, "Get: 200 not working")
	if !strings.Contains(hd.W.(*httptest.ResponseRecorder).Body.String(), "blup") {
		t.Error("Get: content not set")
	}
	if hd.W.Header().Get("Content-Type") != "text" {
		t.Error("Get: contentType not set")
	}

	// request a not-existing resource
	hd = createHandlerData(t, db, "GET", "http://localhost:8080/asdf/qwer/asdfas/res", nil)
	handleRequest(hd)
	checkCode(t, hd, http.StatusNotFound, "Get: 404 not working")
	teardownRedis(db)
}

func TestHandleGetCollection(t *testing.T) {
	db := NewRedisDB()
	resPath := []string{"path", "res"}
	db.CreateResource(resPath, false)
	hd := createHandlerData(t, db, "GET", "http://localhost:8080/asdf/qwer/path/res", nil)
	handleRequest(hd)
	if !strings.Contains(hd.W.(*httptest.ResponseRecorder).Body.String(), "[]") {
		t.Error("Get: content not set")
	}

	data := strings.NewReader("some data")
	hd = createHandlerData(t, db, "POST", "http://localhost:8080/asdf/qwer/path/res", data)
	handleRequest(hd)
	hd = createHandlerData(t, db, "GET", "http://localhost:8080/asdf/qwer/path/res", nil)
	handleRequest(hd)
	if !strings.Contains(hd.W.(*httptest.ResponseRecorder).Body.String(), "[\"0\"]") {
		t.Error("Get: content not set")
	}

	teardownRedis(db)
}
