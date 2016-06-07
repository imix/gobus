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

func TestHandleGetHooks(t *testing.T) {
	db := NewRedisDB()
	resPath := []string{"path", "res"}
	res, _ := db.CreateResource(resPath, false)
	hookData := []byte(fmt.Sprintf(`{"name": "a_hook", "url": "http://test.com/a/hook"}`))
	res.AddHook(hookData)

	hd := createHandlerData(t, db, "GET", "http://localhost:8080/asdf/qwer/path/res/_hooks/0", nil)
	handleRequest(hd)
	checkCode(t, hd, http.StatusOK, "Get hook: 200 not working")
	if !strings.Contains(hd.W.(*httptest.ResponseRecorder).Body.String(), "test.com/a/hook") {
		t.Error("Get Hook: content not set")
	}

	hd = createHandlerData(t, db, "GET", "http://localhost:8080/asdf/qwer/path/res/_hooks", nil)
	handleRequest(hd)
	checkCode(t, hd, http.StatusOK, "Get hooks: 200 not working")
	if !strings.Contains(hd.W.(*httptest.ResponseRecorder).Body.String(), "[\"0\"]") {
		t.Error("Get: content not set")
	}

	hd = createHandlerData(t, db, "GET", "http://localhost:8080/asdf/qwer/path/res/_hooks/0/a", nil)
	handleRequest(hd)
	checkCode(t, hd, http.StatusNotFound, "Get hooks: 404 not working")
	teardownRedis(db)
}

func TestHandlePutHooks(t *testing.T) {
	db := NewRedisDB()
	resPath := []string{"path", "res"}
	res, _ := db.CreateResource(resPath, false)
	hookData := []byte(fmt.Sprintf(`{"name": "a_hook", "url": "http://test.com/a/hook"}`))
	res.AddHook(hookData)

	data := strings.NewReader(`{"name": "a_hook", "url": "http://blup.com/a/hook"}`)
	hd := createHandlerData(t, db, "PUT", "http://localhost:8080/asdf/qwer/path/res/_hooks/0", data)
	handleRequest(hd)
	checkCode(t, hd, http.StatusOK, "Get hook: 200 not working")
	h, _ := res.GetHook("0")
	if !strings.Contains(h.URL, "blup.com/a/hook") {
		t.Error("Get Hook: content not set")
	}

	teardownRedis(db)
}

func TestHandleHooking(t *testing.T) {
	db := NewRedisDB()
	resPath := []string{"an_item"}
	res, err := db.CreateResource(resPath, false)
	if err != nil {
		t.Fatal(err)
	}

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
	handleRequest(hd)
	checkCode(t, hd, http.StatusCreated, "Post Hook: 201 not working")
	hooks, err := res.GetHooks()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(hooks[0].Name, "hook_name") {
		t.Error("Post Hook: Content not working")
	}

	// test hook called
	postdata := strings.NewReader("any data")
	hd = createHandlerData(t, db, "POST", "http://localhost:8080/asdf/qwer/an_item", postdata)
	handleRequest(hd)
	var data []byte
	data = <-c
	var hookevent HookEvent
	err = json.Unmarshal(data, &hookevent)
	if err != nil {
		t.Error(err)
	}
	teardownRedis(db)
}
