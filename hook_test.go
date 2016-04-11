package main

import (
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
