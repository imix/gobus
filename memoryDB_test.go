package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewRessource(t *testing.T) {
	newRes := newResource("asdf", true)
	if newRes.Name != "asdf" {
		t.Error("Name not equal")
	}
	if !newRes.IsItem {
		t.Error("Item not set")
	}
	if newRes.NextId != 0 {
		t.Error("NextID not 0")
	}
}

func TestCreateResourceOneLevelItem(t *testing.T) {
	db := NewMemoryDB()

	elts := []string{"level0"}
	res := db.CreateResource(elts, true)
	if !res.IsItem {
		t.Error("Item not properly set")
	}
	if db.(*MemoryDB).RootResource.Children[0].Name != res.Name {
		t.Error("Resource not properly inserted")
	}
}

func TestCreateResourceTwoLevelItem(t *testing.T) {
	db := NewMemoryDB()

	elts := []string{"level0", "level1"}
	res := db.CreateResource(elts, true)
	root := db.(*MemoryDB).RootResource
	if !res.IsItem {
		t.Error("Item not properly set")
	}
	if root.Children[0].Name != "level0" {
		t.Error("Level0 not properly named")
	}
	if root.Children[0].IsItem {
		t.Error("Level0 item not properly set")
	}
	if root.Children[0].Children[0].Name != res.Name {
		t.Error("Level1 resource not properly set")
	}
}

func TestCreateResourceTwoLevelCollection(t *testing.T) {
	db := NewMemoryDB()

	elts := []string{"level0", "level1"}
	res := db.CreateResource(elts, false)
	root := db.(*MemoryDB).RootResource
	if res.IsItem {
		t.Error("Item not properly set")
	}
	if root.Children[0].Name != "level0" {
		t.Error("Level0 not properly named")
	}
	if root.Children[0].IsItem {
		t.Error("Level0 item not properly set")
	}
	if root.Children[0].Children[0].Name != res.Name {
		t.Error("Level1 resource not properly set")
	}
}

func TestCreateResourceThreeLevelCollection(t *testing.T) {
	db := NewMemoryDB()

	elts := []string{"level0", "level1", "level2"}
	res := db.CreateResource(elts, false)
	root := db.(*MemoryDB).RootResource
	if res.IsItem {
		t.Error("Item not properly set")
	}
	if root.Children[0].Children[0].Children[0].Name != res.Name {
		t.Error("Level2 resource not properly set")
	}
}

func TestCreateMultipleResources(t *testing.T) {
	db := NewMemoryDB()

	elts := []string{"level01"}
	db.CreateResource(elts, false)

	elts = []string{"level02"}
	db.CreateResource(elts, false)

	elts = []string{"level03"}
	db.CreateResource(elts, false)

	if len(db.(*MemoryDB).RootResource.Children) != 3 {
		t.Error("multiple resources not properly set")
	}
}

func TestGetResource(t *testing.T) {
	db := NewMemoryDB()
	elts := []string{"level0", "level1", "level2"}
	db.CreateResource(elts, false)

	res, err := db.GetResource(elts)
	if err != nil {
		t.Error("Error on Get")
	}
	if res.Name != "level2" {
		t.Error("level2 not found")
	}
}

func TestGetInexistingResource(t *testing.T) {
	db := NewMemoryDB()
	elts := []string{"level0", "level1", "level2"}
	db.CreateResource(elts, false)
	elts = []string{"level0", "level1", "level2", "level3", "level4"}
	_, err := db.GetResource(elts)
	if err == nil {
		t.Error("found but should not")
	}
}

func TestDeleteResource(t *testing.T) {
	db := NewMemoryDB()
	elts := []string{"level0", "level1", "level2"}
	db.CreateResource(elts, false)
	root := db.(*MemoryDB).RootResource

	err := db.DeleteResource(elts)
	if err != nil {
		t.Error("delete error")
	}
	if len(root.Children[0].Children[0].Children) > 0 {
		t.Error("delete failed")
	}
	err = db.DeleteResource([]string{"level0"})
	if err == nil {
		t.Error("delete should not be possible on non-leave resources")
	}
}

func TestAddCollection(t *testing.T) {
	db := NewMemoryDB()
	elts := []string{"level0"}
	db.CreateResource(elts, false)

	name, err := db.AddToCollection(elts, "text", []byte("bla"))
	if err != nil {
		t.Error("add to collection error")
	}
	if strings.Compare(name, "0") != 0 {
		t.Error("add index wrong")
	}
	newRes, _ := db.GetResource([]string{"level0", "0"})
	if bytes.Compare(newRes.Value, []byte("bla")) != 0 {
		t.Error("wrong data after add")
	}

	name, _ = db.AddToCollection(elts, "text", []byte("1bla"))
	if strings.Compare(name, "1") != 0 {
		t.Error("add index 1 wrong")
	}
	newRes, _ = db.GetResource([]string{"level0", "1"})
	if bytes.Compare(newRes.Value, []byte("1bla")) != 0 {
		t.Error("wrong data after add")
	}
}
