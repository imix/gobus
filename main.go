package main

import (
	"net/http"
	"net/url"
)

func main() {
	rootURL, _ := url.Parse("http://localhost:8080/")
	db := newResource("root", false, rootURL)
	http.HandleFunc("/", getHandler(db))
	http.ListenAndServe(":8080", nil)
}
