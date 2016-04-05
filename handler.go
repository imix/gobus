package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"
)

// creates a standard response
func respond(w http.ResponseWriter, r *http.Request, status int, msg string) {
	w.WriteHeader(status)
	fmt.Fprintf(w, "%s: ", strconv.Itoa(status))
	fmt.Fprintf(w, "%s\n", msg)
	fmt.Fprintf(w, "Request URL: %s\n", r.URL.String())
}

// handle Put requests
func handlePut(db *Resource, w http.ResponseWriter, r *http.Request) {
	components, err := getRelativePath(db.URL.Path, r.URL.Path)
	if err != nil {
		respond(w, r, http.StatusNotFound, "Not Found")
		return
	}
	res, remainder := getResource(db, components)
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		respond(w, r, http.StatusBadRequest, "Invalid Request")
	} else if len(remainder) == 0 {
		if res.IsItem { //update item
			setValue(res, data)
			respond(w, r, http.StatusOK, fmt.Sprintf("Put %s!", data))
			callHooks(res, "PUT")
		} else { //collection
			respond(w, r, http.StatusNotFound, "Can not Put collection")
		}
	} else {
		if containsCommand(remainder) {
			respond(w, r, http.StatusNotFound, "Can not Put command")
		} else {
			res := createResource(res, remainder, len(data) > 0)
			msg := "Resource created"
			if len(data) > 0 { // add value to item
				setValue(res, data)
				msg = fmt.Sprintf("Put %s!", data)
			}
			respond(w, r, http.StatusCreated, msg)
		}
	}
}

// respond with a "Created" (201) and set location to the new url
// the new url is composed of the path with id attached
func respondCreatedNewURL(w http.ResponseWriter, baseUrl *url.URL, id string) {
	newURL := url.URL{
		Scheme: baseUrl.Scheme,
		Host:   baseUrl.Host,
		Path:   path.Join(baseUrl.Path, id),
	}
	w.WriteHeader(http.StatusCreated)
	w.Header().Set("Location", newURL.String())
	fmt.Fprintf(w, "201 Resource Created %s!", newURL.String())
}

// handle Post requests that consider commands
// currently only _hooks is supported
func handlePostCommand(w http.ResponseWriter, r *http.Request, res *Resource, cmd string, data []byte) {
	switch cmd {
	case "_hooks":
		name, err := addHook(res, data)
		if err != nil {
			respond(w, r, http.StatusInternalServerError, "Could not create Hook")
		}
		respondCreatedNewURL(w, r.URL, name)
	}
}

// handle Post requests
// post is permitted on collections and for commands
// post on items is not allowed
func handlePost(db *Resource, w http.ResponseWriter, r *http.Request) {
	components, err := getRelativePath(db.URL.Path, r.URL.Path)
	if err != nil {
		respond(w, r, http.StatusNotFound, "Not Found")
		return
	}
	res, rest := getResource(db, components)
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		respond(w, r, http.StatusBadRequest, "Invalid Request")
	} else if len(rest) == 0 { //found the resource
		if res.IsItem {
			respond(w, r, http.StatusConflict, "Item does not support Post")
		} else {
			name := addToCollection(res, data)
			respondCreatedNewURL(w, r.URL, name)
			callHooks(res, "POST")
		}
	} else if len(rest) == 1 && isCommand(rest[0]) {
		handlePostCommand(w, r, res, rest[0], data)
	} else {
		respond(w, r, http.StatusNotFound, "Not Found")
	}
}

// handles delete of a resource
// can delete resources, hooks
func handleDelete(db *Resource, w http.ResponseWriter, r *http.Request) {
	components, err := getRelativePath(db.URL.Path, r.URL.Path)
	if err != nil {
		respond(w, r, http.StatusNotFound, "Not Found")
		return
	}
	res, rest := getResource(db, components)
	if len(rest) == 0 { //found the resource
		err := deleteResource(db, components)
		if err != nil {
			respond(w, r, http.StatusNotFound, err.Error())
		}
		respond(w, r, http.StatusOK, fmt.Sprintf("Get %s!", res.Value))
	} else if isCommand(rest[0]) {
		switch rest[0] {
		case "_hooks":
			if len(rest) != 2 {
				respond(w, r, http.StatusNotFound, "Not Found")
				return
			}
			err := deleteHook(res, rest[1])
			if err != nil {
				respond(w, r, http.StatusNotFound, "Not Found")
				return
			}
			respond(w, r, http.StatusOK, "Deleted")
		default:
			respond(w, r, http.StatusNotFound, "Not Found")
		}
	} else { //not found
		respond(w, r, http.StatusNotFound, "Not Found")
	}
}

// handles Get request
// returns the value of the resource
// BUG(saes): get on collections returns empty and not the items
func handleGet(db *Resource, w http.ResponseWriter, r *http.Request) {
	components, err := getRelativePath(db.URL.Path, r.URL.Path)
	if err != nil {
		respond(w, r, http.StatusNotFound, "Not Found")
		return
	}
	res, rest := getResource(db, components)
	if len(rest) == 0 { //found the resource
		respond(w, r, http.StatusOK, fmt.Sprintf("Get %s!", res.Value))
	} else if len(rest) == 1 && isCommand(rest[0]) { //resource exists with command
		switch rest[0] {
		case "_hooks":
			json, err := getHooksJson(res)
			if err != nil {
				respond(w, r, http.StatusInternalServerError, "Could not get Hook Json")
				return
			}
			w.Write(json)
		default:
			log.Printf("unimplemented command", rest[0])
			respond(w, r, http.StatusNotFound, "Not Found")
		}
	} else { //not found
		respond(w, r, http.StatusNotFound, "Not Found")
	}
}

func getHandler(db *Resource) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		switch r.Method {
		case "PUT":
			handlePut(db, w, r)
		case "POST":
			handlePost(db, w, r)
		case "DELETE":
			handleDelete(db, w, r)
		default:
			handleGet(db, w, r)
		}
		log.Printf("%s\t%s\t%s",
			r.Method,
			r.RequestURI,
			time.Since(start),
		)
	}
}
