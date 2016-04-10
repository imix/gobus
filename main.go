package main

import (
	"net/http"
	"net/url"
)

func main() {
	rootURL, _ := url.Parse("http://localhost:8080/")
	db := NewMemoryDB()

	http.HandleFunc("/", getHandler(db, rootURL))
	http.ListenAndServe(":8080", nil)
}
