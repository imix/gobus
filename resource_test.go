package main

import (
	"bytes"
	"net/url"
	"strings"
	"testing"
)

func TestNewRessource(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:8080/asdf/qwer")
	newRes := newResource("asdf", true, testURL)
	if newRes.Name != "asdf" {
		t.Error("Name not equal")
	}
	if !newRes.IsItem {
		t.Error("Item not set")
	}
	if newRes.URL != testURL {
		t.Error("URL not set")
	}
	if newRes.NextId != 0 {
		t.Error("NextID not 0")
	}
}

func TestCreateResourceOneLevelItem(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:8080/asdf/qwer")
	root := newResource("root", false, testURL)

	elts := []string{"level0"}
	res := createResource(root, elts, true)
	if !res.IsItem {
		t.Error("Item not properly set")
	}
	if root.Children[0] != res {
		t.Error("Resource not properly inserted")
	}
}

func TestCreateResourceTwoLevelItem(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:8080/asdf/qwer")
	root := newResource("root", false, testURL)

	elts := []string{"level0", "level1"}
	res := createResource(root, elts, true)
	if !res.IsItem {
		t.Error("Item not properly set")
	}
	if root.Children[0] == res {
		t.Error("Resource not properly inserted")
	}
	if root.Children[0].Name != "level0" {
		t.Error("Level0 not properly named")
	}
	if root.Children[0].IsItem {
		t.Error("Level0 item not properly set")
	}
	if root.Children[0].Children[0] != res {
		t.Error("Level1 resource not properly set")
	}
}

func TestCreateResourceTwoLevelCollection(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:8080/asdf/qwer")
	root := newResource("root", false, testURL)

	elts := []string{"level0", "level1"}
	res := createResource(root, elts, false)
	if res.IsItem {
		t.Error("Item not properly set")
	}
	expectedURL := "http://localhost:8080/asdf/qwer/level0/level1"
	if strings.Compare(res.URL.String(), expectedURL) != 0 {
		t.Error("URL not properly set")
	}
	if root.Children[0] == res {
		t.Error("Resource not properly inserted")
	}
	if root.Children[0].Name != "level0" {
		t.Error("Level0 not properly named")
	}
	if root.Children[0].IsItem {
		t.Error("Level0 item not properly set")
	}
	if root.Children[0].Children[0] != res {
		t.Error("Level1 resource not properly set")
	}
}

func TestCreateResourceThreeLevelCollection(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:8080/asdf/qwer")
	root := newResource("root", false, testURL)

	elts := []string{"level0", "level1", "level2"}
	res := createResource(root, elts, false)
	if res.IsItem {
		t.Error("Item not properly set")
	}
	expectedURL := "http://localhost:8080/asdf/qwer/level0/level1/level2"
	if strings.Compare(res.URL.String(), expectedURL) != 0 {
		t.Error("URL not properly set")
	}
	if root.Children[0].Children[0].Children[0] != res {
		t.Error("Level2 resource not properly set")
	}
}

func TestCreateMultipleResources(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:8080/asdf/qwer")
	root := newResource("root", false, testURL)

	elts := []string{"level01"}
	createResource(root, elts, false)

	elts = []string{"level02"}
	createResource(root, elts, false)

	elts = []string{"level03"}
	createResource(root, elts, false)

	if len(root.Children) != 3 {
		t.Error("multiple resources not properly set")
	}
}

func TestGetResource(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:8080/asdf/qwer")
	root := newResource("root", false, testURL)
	elts := []string{"level0", "level1", "level2"}
	createResource(root, elts, false)

	res, remainder := getResource(root, elts)
	if len(remainder) > 0 {
		t.Error("remainder too long")
	}
	if res.Name != "level2" {
		t.Error("level2 not found")
	}
}

func TestGetResourceWithRemainder(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:8080/asdf/qwer")
	root := newResource("root", false, testURL)
	elts := []string{"level0", "level1", "level2"}
	createResource(root, elts, false)
	elts = []string{"level0", "level1", "level2", "level3", "level4"}
	res, remainder := getResource(root, elts)
	if res.Name != "level2" {
		t.Error("level2 not found")
	}
	if remainder[0] != "level3" && remainder[1] != "level4" {
		t.Error("remaidner not correct")
	}
}

func TestDeleteResource(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:8080/asdf/qwer")
	root := newResource("root", false, testURL)
	elts := []string{"level0", "level1", "level2"}
	createResource(root, elts, false)

	err := deleteResource(root, elts)
	if err != nil {
		t.Error("delete error")
	}
	if len(root.Children[0].Children[0].Children) > 0 {
		t.Error("delete failed")
	}
	err = deleteResource(root, []string{"level0"})
	if err == nil {
		t.Error("delete should not be possible on non-leave resources")
	}
}

func TestAddCollection(t *testing.T) {
	testURL, _ := url.Parse("http://localhost:8080/asdf/qwer")
	root := newResource("root", false, testURL)
	elts := []string{"level0"}
	res := createResource(root, elts, false)

	name := addToCollection(res, []byte("bla"))
	if strings.Compare(name, "0") != 0 {
		t.Error("add index wrong")
	}
	newRes, _ := getResource(root, []string{"level0", "0"})
	if bytes.Compare(newRes.Value, []byte("bla")) != 0 {
		t.Error("wrong data after add")
	}

	name = addToCollection(res, []byte("1bla"))
	if strings.Compare(name, "1") != 0 {
		t.Error("add index 1 wrong")
	}
	newRes, _ = getResource(root, []string{"level0", "1"})
	if bytes.Compare(newRes.Value, []byte("1bla")) != 0 {
		t.Error("wrong data after add")
	}
}
