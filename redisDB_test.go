package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateResourceOneLevelItem(t *testing.T) {
	db := NewRedisDB()

	elts := []string{"level0"}
	db.CreateResource(elts, true)
	res, _ := db.GetResource(elts)
	if item, _ := res.IsItem(); !item {
		t.Error("Item not properly set")
	}
	if name, _ := res.Name(); name != "level0" {
		t.Error("Resource not properly inserted")
	}
	teardownRedis(db)
}

func TestCreateResourceTwoLevelItem(t *testing.T) {
	db := NewRedisDB()

	elts := []string{"level0", "level1"}
	res, _ := db.CreateResource(elts, true)
	if item, _ := res.IsItem(); !item {
		t.Error("Item not properly set")
	}
	root, _ := db.GetResource([]string{}) // get root
	children, _ := root.GetChildren()
	if len(children) != 1 {
		t.Fatal("Child not properly inserted.")
	}
	res, err := db.GetResource([]string{"level0"})
	if err != nil {
		t.Error("Level0 not properly set")
	}
	children, _ = res.GetChildren()
	if len(children) != 1 {
		t.Fatal("Level0 child not properly inserted.")
	}
	if item, _ := res.IsItem(); item {
		t.Error("Level0 item not properly set")
	}
	if name, _ := res.Name(); name != "level0" {
		t.Error("Level0 not properly named")
	}
	res, err = db.GetResource(elts)
	if err != nil {
		t.Error("Level1 not properly set")
	}
	if name, _ := res.Name(); name != "level1" {
		t.Error("Level1 not properly named")
	}
	teardownRedis(db)
}

func TestCreateResourceTwoLevelCollection(t *testing.T) {
	db := NewRedisDB()

	elts := []string{"level0", "level1"}
	res, _ := db.CreateResource(elts, false)
	if item, _ := res.IsItem(); item {
		t.Error("Item not properly set")
	}
	res, _ = db.GetResource([]string{"level0"})
	if name, _ := res.Name(); name != "level0" {
		t.Error("Level0 not properly named")
	}
	if item, _ := res.IsItem(); item {
		t.Error("Level0 item not properly set")
	}
	res, _ = db.GetResource([]string{"level0", "level1"})
	if name, _ := res.Name(); name != "level1" {
		t.Error("Level1 resource not properly set")
	}
	teardownRedis(db)
}

func TestCreateResourceThreeLevelCollection(t *testing.T) {
	db := NewRedisDB()

	elts := []string{"level0", "level1", "level2"}
	res, _ := db.CreateResource(elts, false)
	if item, _ := res.IsItem(); item {
		t.Error("Item not properly set")
	}
	res, _ = db.GetResource(elts)
	if name, _ := res.Name(); name != "level2" {
		t.Error("Level2 resource not properly set")
	}
	teardownRedis(db)
}

func TestCreateMultipleResources(t *testing.T) {
	db := NewRedisDB()

	elts := []string{"level01"}
	db.CreateResource(elts, false)

	elts = []string{"level02"}
	db.CreateResource(elts, false)

	elts = []string{"level03"}
	db.CreateResource(elts, false)

	res, _ := db.GetResource([]string{})
	if children, _ := res.GetChildren(); len(children) != 3 {
		t.Error("multiple resources not properly set")
	}
	teardownRedis(db)
}

func TestGetResource(t *testing.T) {
	db := NewRedisDB()
	elts := []string{"level0", "level1", "level2"}
	db.CreateResource(elts, false)

	res, err := db.GetResource(elts)
	if err != nil {
		t.Error("Error on Get")
	}
	if name, _ := res.Name(); name != "level2" {
		t.Error("level2 not found")
	}
	teardownRedis(db)
}

func TestGetInexistingResource(t *testing.T) {
	db := NewRedisDB()
	elts := []string{"level0", "level1", "level2"}
	db.CreateResource(elts, false)
	elts = []string{"level0", "level1", "level2", "level3", "level4"}
	_, err := db.GetResource(elts)
	if err == nil {
		t.Error("found but should not")
	}
	teardownRedis(db)
}

func TestDeleteResource(t *testing.T) {
	db := NewRedisDB()
	elts := []string{"level0", "level1", "level2"}
	db.CreateResource(elts, false)

	err := db.DeleteResource(elts)
	if err != nil {
		t.Error("delete error")
	}
	elts = []string{"level0", "level1"}
	res, _ := db.GetResource(elts)
	if children, _ := res.GetChildren(); len(children) > 0 {
		t.Error("delete failed")
	}
	err = db.DeleteResource([]string{"level0"})
	if err == nil {
		t.Error("delete should not be possible on non-leave resources")
	}
	teardownRedis(db)
}

func TestAddCollection(t *testing.T) {
	db := NewRedisDB()
	elts := []string{"level0"}
	res, _ := db.CreateResource(elts, false)

	name, err := res.AddToCollection("text", []byte("bla"))
	if err != nil {
		t.Error("add to collection error")
	}
	if name != "0" {
		t.Error("add index wrong")
	}
	newRes, _ := db.GetResource([]string{"level0", "0"})
	_, value, _ := newRes.GetValue()
	if bytes.Compare(value, []byte("bla")) != 0 {
		t.Error("wrong data after add")
	}

	name, _ = res.AddToCollection("text", []byte("1bla"))
	if strings.Compare(name, "1") != 0 {
		t.Error("add index 1 wrong")
	}
	newRes, _ = db.GetResource([]string{"level0", "1"})
	_, value, _ = newRes.GetValue()
	if bytes.Compare(value, []byte("1bla")) != 0 {
		t.Error("wrong data after add")
	}
	teardownRedis(db)
}

func TestAddHook(t *testing.T) {
	db := NewRedisDB()
	hookData := []byte(`{"name": "hook_name", "url": "http://www.test.ch/my/resource"}`)
	resPath := []string{"path"}
	res, _ := db.CreateResource(resPath, true)
	name, err := res.AddHook(hookData)
	if err != nil {
		t.Error("Hook Add failed", err)
	}
	if strings.Compare(name, "0") != 0 {
		t.Error("Hook Id not set")
	}
}

func TestDeleteHook(t *testing.T) {
	db := NewRedisDB()
	hookData := []byte(`{"name": "hook_name", "url": "http://www.test.ch/my/resource"}`)
	resPath := []string{"path"}
	res, _ := db.CreateResource(resPath, true)
	name, _ := res.AddHook(hookData)
	err := res.DeleteHook(name)
	if err != nil {
		t.Error("Hook Delete failed", err)
	}

	err = res.DeleteHook(name)
	if err == nil {
		t.Error("Hook Delete Inexisting failed:", err)
	}
	teardownRedis(db)
}

func TestCallHook(t *testing.T) {
	db := NewRedisDB()
	resPath := []string{"path"}
	res, _ := db.CreateResource(resPath, true)

	c := make(chan []byte)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := ioutil.ReadAll(r.Body)
		c <- b
	}))
	defer ts.Close()

	hookData := []byte(fmt.Sprintf(`{"name": "hook_name", "url": "%s"}`, ts.URL))
	_, err := res.AddHook(hookData)
	if err != nil {
		t.Error(err)
	}
	hooks, _ := res.GetHooks()
	callHooks(hooks, "POST", true, "http://a_resource.com/res")
	var data []byte
	data = <-c
	var hookevent HookEvent
	err = json.Unmarshal(data, &hookevent)
	if err != nil {
		t.Error(err)
	}
	teardownRedis(db)
}
