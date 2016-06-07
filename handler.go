package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

type HandlerData struct {
	DB      GoBusDB
	BaseURL *url.URL
	W       http.ResponseWriter
	R       *http.Request
}

// creates a standard response
func respond(hd *HandlerData, status int, msg string) {
	w, r := hd.W, hd.R
	w.WriteHeader(status)
	fmt.Fprintf(w, "%s: ", strconv.Itoa(status))
	fmt.Fprintf(w, "%s\n", msg)
	fmt.Fprintf(w, "Request URL: %s\n", r.URL.String())
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

// posts an item into a collection, returns the name of the item (which is generated)
func postCollection(hd *HandlerData, res Resource) {
	body := hd.R.Body
	data, err := ioutil.ReadAll(body)
	body.Close()
	if err != nil {
		respond(hd, http.StatusBadRequest, "Invalid Request")
		return
	}
	contentType := hd.R.Header.Get("Content-Type")
	name, err := res.AddToCollection(contentType, data)
	if err != nil {
		log.Printf("Internal error, could not get hooks: ", err.Error())
		return
	}
	respondCreatedNewURL(hd.W, hd.R.URL, name)

	callHooks(res, "POST", hd.BaseURL.Path)
}

// gets a collection, returns a list of children in the collection
func getCollection(hd *HandlerData, res Resource) {
	abs_ids, err := res.GetChildren()
	if err != nil {
		respond(hd, http.StatusInternalServerError, "Could not get children")
		return
	}
	// child keys are absolute, have to convert them to relative
	var ids = []string{}
	for _, c := range abs_ids {
		elts := strings.Split(c, ":")
		ids = append(ids, elts[len(elts)-1])
	}
	json, err := json.Marshal(ids)
	if err != nil {
		respond(hd, http.StatusInternalServerError, "Could not get Collection Json")
		return
	}
	hd.W.Write(json)
}

// handles methods on collections
func handleCollection(hd *HandlerData, res Resource) {
	switch hd.R.Method {
	case "DELETE":
		deleteResource(hd, res)
	case "GET":
		getCollection(hd, res)
	case "POST":
		postCollection(hd, res)
	default:
		respond(hd, http.StatusMethodNotAllowed, "Method not allowed for collection.")
	}
}

// puts an item
func putItem(hd *HandlerData, res Resource) {
	body := hd.R.Body
	data, err := ioutil.ReadAll(body)
	body.Close()
	if err != nil {
		respond(hd, http.StatusBadRequest, "Invalid Request")
		return
	}
	contentType := hd.R.Header.Get("Content-Type")
	err = res.SetValue(contentType, data)
	if err != nil {
		respond(hd, http.StatusInternalServerError, "Could not set item value.")
		return
	}
	respond(hd, http.StatusOK, fmt.Sprintf("Put %s!", data))

	callHooks(res, "PUT", hd.BaseURL.Path)
}

func getItem(hd *HandlerData, res Resource) {
	ct, value, err := res.GetValue()
	if err != nil {
		respond(hd, http.StatusInternalServerError, "Could not get item value.")
		return
	}
	w := hd.W
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", ct)
	w.Write(value)
}

// deletes an resource (item or collection)
func deleteResource(hd *HandlerData, res Resource) {
	// call Hooks before executing delete
	callHooks(res, "DELETE", hd.BaseURL.Path)

	err := res.Delete()
	if err != nil {
		respond(hd, http.StatusNotFound, "Could not delete Item")
		return
	}
	respond(hd, http.StatusOK, fmt.Sprintf("Item deleted!"))
}

// handles methods on items
func handleItem(hd *HandlerData, res Resource) {
	switch hd.R.Method {
	case "DELETE":
		deleteResource(hd, res)
	case "GET":
		getItem(hd, res)
	case "PUT":
		putItem(hd, res)
	default:
		respond(hd, http.StatusMethodNotAllowed, "Method not allowed for items.")
	}
}

func handleExistingResource(hd *HandlerData, res Resource) {
	isitem, err := res.IsItem()
	if err != nil {
		respond(hd, http.StatusInternalServerError, "Could not get Resource type.")
		return
	}
	if isitem {
		handleItem(hd, res)
	} else {
		handleCollection(hd, res)
	}
}

// creates an inexisting resource
// only put is permitted
// if a body is present, an item is created
// if no body is present, a collection is created
func handleInexistingResource(hd *HandlerData, comps []string) {
	if hd.R.Method != "PUT" {
		respond(hd, http.StatusNotFound, "Resource not found.")
		return
	}
	body := hd.R.Body
	data, err := ioutil.ReadAll(body)
	body.Close()
	if err != nil {
		respond(hd, http.StatusBadRequest, "Invalid Request")
		return
	}
	res, err := hd.DB.CreateResource(comps, len(data) > 0)
	if err != nil {
		respond(hd, http.StatusInternalServerError, "Could not create Resource")
		return
	}
	msg := "Resource created"
	if len(data) > 0 { // add value to item
		contentType := hd.R.Header.Get("Content-Type")
		res.SetValue(contentType, data)
		msg = fmt.Sprintf("Put %s!", data)
	}
	respond(hd, http.StatusCreated, msg)
}

// handles any type of command in a request
func handleCommand(hd *HandlerData, res Resource, cmds []string) {
	switch cmds[0] {
	case "_hooks":
		handleHookRequest(hd, res, cmds)
	case "_forward":
		handleForwardRequest(hd, res, cmds)
	default:
		log.Printf("unimplemented command", cmds)
		respond(hd, http.StatusNotFound, "Not Found")
	}
}

// handles a request
func handleRequest(hd *HandlerData) {
	comps, cmds, err := disectPath(hd.BaseURL.Path, hd.R.URL.Path)
	if err != nil {
		respond(hd, http.StatusNotFound, "Not Found")
		return
	}
	// check security
	/*if accessAllowed() {
	respond
	*/
	res, err := getForwardResource(hd, comps, cmds)
	if err != nil {
		respond(hd, http.StatusInternalServerError, "Could not get ForwardResource")
		return
	}
	if res != nil {
		forwardRequest(hd, res, comps)
		return
	}
	exists, err := hd.DB.ResourceExists(comps)
	if err != nil {
		respond(hd, http.StatusInternalServerError, "Could not get Resource")
		return
	}
	if !exists {
		if len(cmds) == 0 {
			handleInexistingResource(hd, comps)
		} else {
			respond(hd, http.StatusNotFound, "Can not execute command on inexisting resource")
		}
		return
	}
	res, err = hd.DB.GetResource(comps)
	if err != nil {
		respond(hd, http.StatusInternalServerError, "Resource Not Found")
		return
	}
	if len(cmds) > 0 {
		handleCommand(hd, res, cmds)
		return
	}
	handleExistingResource(hd, res)
}

// creates a http handler for handling requests
func getHandler(db GoBusDB, baseURL *url.URL) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hd := &HandlerData{
			DB:      db,
			BaseURL: baseURL,
			W:       w,
			R:       r,
		}
		start := time.Now()

		handleRequest(hd)

		log.Printf("%s\t%s\t%s",
			r.Method,
			r.RequestURI,
			time.Since(start),
		)
	}
}
