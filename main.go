package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	rootURL, _ := url.Parse("http://localhost:8080/")
	db := NewRedisDB()

	http.HandleFunc("/", getHandler(db, rootURL))
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
