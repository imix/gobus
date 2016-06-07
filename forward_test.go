package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAddForward(t *testing.T) {
	db := NewRedisDB()

	resPath := []string{"an_item"}
	res, _ := db.CreateResource(resPath, false)
	cmds := []string{"_forward"}
	data := strings.NewReader(`{"url": "http://blup.com/a/hook"}`)
	hd := createHandlerData(t, db, "PUT", "http://localhost:8080/asdf/qwer/an_item/_forward", data)

	handleForwardRequest(hd, res, cmds)
	checkCode(t, hd, http.StatusOK, "AddForward: 200 not working")

	f, _ := res.GetForward()
	if !strings.Contains(f.URL, "http://blup.com/a/hook") {
		t.Error("AddForward: Content not working")
	}

	teardownRedis(db)
}

func TestGetForward(t *testing.T) {
	db := NewRedisDB()

	resPath := []string{"an_item"}
	res, _ := db.CreateResource(resPath, false)
	data := []byte(`{"url": "http://blup.com/a/hook"}`)
	res.AddForward(data)

	cmds := []string{"_forward"}
	hd := createHandlerData(t, db, "GET", "http://localhost:8080/an_item/_forward", nil)

	handleForwardRequest(hd, res, cmds)
	checkCode(t, hd, http.StatusOK, "AddForward: 200 not working")

	if !strings.Contains(hd.W.(*httptest.ResponseRecorder).Body.String(), `{"url":"http://blup.com/a/hook"}`) {
		t.Error("GetForward: Content not working")
	}

	teardownRedis(db)
}

func TestDeleteForward(t *testing.T) {
	db := NewRedisDB()

	resPath := []string{"an_item"}
	res, _ := db.CreateResource(resPath, false)
	data := []byte(`{"url": "http://blup.com/a/hook"}`)
	res.AddForward(data)

	cmds := []string{"_forward"}
	hd := createHandlerData(t, db, "DELETE", "http://localhost:8080/an_item/_forward", nil)

	handleForwardRequest(hd, res, cmds)
	checkCode(t, hd, http.StatusOK, "AddForward: 200 not working")

	f, _ := res.GetForward()
	if !strings.Contains(f.URL, "") {
		t.Error("DeleteForward: Content not working")
	}

	teardownRedis(db)
}

func TestHandleForwarding(t *testing.T) {
	db := NewRedisDB()
	resPath := []string{"an_item"}
	res, _ := db.CreateResource(resPath, false)

	// start server for forwarding test
	c := make(chan []byte, 256)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := ioutil.ReadAll(r.Body)
		c <- b
		w.Write([]byte("answer"))
	}))
	defer ts.Close()

	data := []byte(fmt.Sprintf(`{"url":"%s"}`, ts.URL))
	err := res.AddForward(data)
	if err != nil {
		t.Error("Handle Forwarding: add forward failed")
	}

	// test forward called
	postdata := "some forward data"
	reader := strings.NewReader("some forward data")
	hd := createHandlerData(t, db, "POST", "http://localhost:8080/asdf/qwer/an_item/another/path", reader)
	handleRequest(hd)
	var receivedData []byte
	receivedData = <-c

	if bytes.Compare([]byte(postdata), receivedData) != 0 {
		t.Error("Data wrong")
	}
	teardownRedis(db)
}
