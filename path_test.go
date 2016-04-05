package main

import "testing"

func testSamePath(p1, p2 []string) bool {
	if len(p1) != len(p2) {
		return false
	}
	for i, v := range p1 {
		if v != p2[i] {
			return false
		}
	}
	return true
}

func TestGetRelativePath(t *testing.T) {
	var testdata = []struct {
		Base     string
		New      string
		Outslice []string
		ErrNil   bool
	}{
		{"", "path", []string{"path"}, true},
		{"path/more", "path/more/not", []string{"not"}, true},
		{"path/more", "path", []string{}, false},
		{"/path/more", "more/path", []string{}, false},
	}
	for i, td := range testdata {
		rel, err := getRelativePath(td.Base, td.New)
		if td.ErrNil != (err == nil) {
			t.Error("Relative Test failed Error Nr:", i, err)
		}
		if !td.ErrNil {
			continue
		}
		if !testSamePath(td.Outslice, rel) {
			t.Error("Relative Test failed Compare Nr:", i)
		}
	}
}

func TestSplitPath(t *testing.T) {
	var splittest = []struct {
		Instring string
		Outslice []string
	}{
		{"path", []string{"path"}},
		{"path/", []string{"path"}},
		{"path/more", []string{"path", "more"}},
		{"/path/more", []string{"path", "more"}},
	}
	for i, st := range splittest {
		if !testSamePath(splitPath(st.Instring), st.Outslice) {
			t.Error("Split Test failed Nr: ", i)
		}
	}
}

func TestIsCommand(t *testing.T) {
	if !isCommand("_hooks") {
		t.Error("isCommand _hooks not recognized")
	}
	if isCommand("_asdf") {
		t.Error("isCommand random accepted")
	}
}

func TestContainsCommand(t *testing.T) {
	var pathtests = []struct {
		Instring string
		Contains bool
	}{
		{"path", false},
		{"path/_hooks", true},
		{"path/_hooks/something", true},
		{"path/_hookssomething/a", false},
	}
	for i, st := range pathtests {
		if containsCommand(splitPath(st.Instring)) != st.Contains {
			t.Error("ContainCommand failed Nr: ", i)
		}
	}
}
