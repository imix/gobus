package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"time"
)

type HandlerData struct {
	DB      GoBusDB
	BaseURL *url.URL
	W       http.ResponseWriter
	R       *http.Request
}

// creates a standard response
func respond(w http.ResponseWriter, r *http.Request, status int, msg string) {
	w.WriteHeader(status)
	fmt.Fprintf(w, "%s: ", strconv.Itoa(status))
	fmt.Fprintf(w, "%s\n", msg)
	fmt.Fprintf(w, "Request URL: %s\n", r.URL.String())
}

func buildResURL(baseURL *url.URL, relPath []string) string {
	resURL := *baseURL
	resURL.Path = path.Join(resURL.Path, path.Join(relPath...))
	return resURL.String()
}

func callHooks(hooks []*Hook, method string, isItem bool, resURL string) {
	for _, h := range hooks {
		var event = HookEvent{
			Name:             h.Name,
			Method:           method,
			Item:             isItem,
			ModifiedResource: resURL,
		}
		data, err := json.Marshal(event)
		if err != nil {
			log.Printf("Failed to marshal hook %s", h.Name)
			continue
		}
		go http.Post(h.URL, "application/json", bytes.NewReader(data))
	}
}

func getHooksJson(res *Resource) ([]byte, error) {
	return json.Marshal(res.Hooks)
}

// handle Put requests
func handlePut(hd *HandlerData) {
	db, w, r := hd.DB, hd.W, hd.R
	comps, cmds, err := disectPath(hd.BaseURL.Path, r.URL.Path)
	if err != nil {
		respond(w, r, http.StatusNotFound, "Not Found")
		return
	}
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		respond(w, r, http.StatusBadRequest, "Invalid Request")
		return
	}
	res, err := db.GetResource(comps)
	if err != nil { // create resource if not existing
		res := db.CreateResource(comps, len(data) > 0)
		if res == nil {
			respond(w, r, http.StatusInternalServerError, "Could not create Resource")
			return
		}
		msg := "Resource created"
		if len(data) > 0 { // add value to item
			db.ResourceSetValue(comps, data)
			msg = fmt.Sprintf("Put %s!", data)
		}
		respond(w, r, http.StatusCreated, msg)
	}
	if cmds != nil {
		respond(w, r, http.StatusNotFound, "Can not Put command")
		return
	}
	// resource exists
	if res.IsItem { //update item
		db.ResourceSetValue(comps, data)
		respond(w, r, http.StatusOK, fmt.Sprintf("Put %s!", data))
		hooks, err := db.GetHooks(comps)
		if err != nil {
			log.Printf("Internal error, could not get hooks: ", err.Error())
			return
		}
		callHooks(hooks, "PUT", res.IsItem, buildResURL(hd.BaseURL, comps))
	} else { //collection
		respond(w, r, http.StatusConflict, "Can not Put collection")
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
func handlePostCommand(hd *HandlerData, comps, cmds []string, data []byte) {
	db, w, r := hd.DB, hd.W, hd.R
	if len(cmds) > 1 {
		respond(w, r, http.StatusBadRequest, "Invalid Request")
		return
	}
	switch cmds[0] {
	case "_hooks":
		name, err := db.AddHook(comps, data)
		if err != nil {
			respond(w, r, http.StatusInternalServerError, "Could not create Hook")
		}
		respondCreatedNewURL(w, r.URL, name)
	}
}

// handle Post requests
// post is permitted on collections and for commands
// post on items is not allowed
func handlePost(hd *HandlerData) {
	db, w, r := hd.DB, hd.W, hd.R
	comps, cmds, err := disectPath(hd.BaseURL.Path, r.URL.Path)
	if err != nil {
		respond(w, r, http.StatusNotFound, "Not Found")
		return
	}
	res, err := db.GetResource(comps)
	if err != nil {
		respond(w, r, http.StatusNotFound, "Not Found")
		return
	}
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		respond(w, r, http.StatusBadRequest, "Invalid Request")
		return
	}
	if cmds == nil { //found the resource no command
		if res.IsItem {
			respond(w, r, http.StatusConflict, "Item does not support Post")
		} else {
			name, err := db.AddToCollection(comps, data)
			if err != nil {
				log.Printf("Internal error, could not get hooks: ", err.Error())
				return
			}
			respondCreatedNewURL(w, r.URL, name)
			hooks, err := db.GetHooks(comps)
			if err != nil {
				log.Printf("Internal error, could not get hooks: ", err.Error())
				return
			}
			callHooks(hooks, "POST", res.IsItem, buildResURL(hd.BaseURL, comps))
		}
	} else {
		handlePostCommand(hd, comps, cmds, data)
	}
}

// handles delete of a resource
// can delete resources, hooks
func handleDelete(hd *HandlerData) {
	db, w, r := hd.DB, hd.W, hd.R
	comps, cmds, err := disectPath(hd.BaseURL.Path, r.URL.Path)
	if err != nil {
		respond(w, r, http.StatusNotFound, "Not Found")
		return
	}
	res, err := db.GetResource(comps)
	if err != nil {
		respond(w, r, http.StatusNotFound, "Not Found")
		return
	}
	if cmds == nil { //no command
		err := db.DeleteResource(comps)
		if err != nil {
			respond(w, r, http.StatusNotFound, err.Error())
			return
		}
		respond(w, r, http.StatusOK, fmt.Sprintf("Get %s!", res.Value))
	} else {
		switch cmds[0] {
		case "_hooks":
			err := db.DeleteHook(comps, cmds)
			if err != nil {
				respond(w, r, http.StatusNotFound, "Not Found")
				return
			}
			respond(w, r, http.StatusOK, "Deleted")
		default:
			respond(w, r, http.StatusNotFound, "Not Found")
		}
	}
}

// handles Get request
// returns the value of the resource
// BUG(saes): get on collections returns empty and not the items
func handleGet(hd *HandlerData) {
	db, w, r := hd.DB, hd.W, hd.R
	comps, cmds, err := disectPath(hd.BaseURL.Path, r.URL.Path)
	if err != nil {
		respond(w, r, http.StatusNotFound, "Not Found")
		return
	}
	res, err := db.GetResource(comps)
	if err != nil {
		respond(w, r, http.StatusNotFound, "Not Found")
		return
	}
	if cmds == nil { //resource without commadn
		respond(w, r, http.StatusOK, fmt.Sprintf("Get %s!", res.Value))
	} else { //resource exists with command
		switch cmds[0] {
		case "_hooks":
			json, err := getHooksJson(res)
			if err != nil {
				respond(w, r, http.StatusInternalServerError, "Could not get Hook Json")
				return
			}
			w.Write(json)
		default:
			log.Printf("unimplemented command", cmds)
			respond(w, r, http.StatusNotFound, "Not Found")
		}
	}
}

func getHandler(db GoBusDB, baseURL *url.URL) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hd := &HandlerData{
			DB:      db,
			BaseURL: baseURL,
			W:       w,
			R:       r,
		}
		start := time.Now()
		switch r.Method {
		case "PUT":
			handlePut(hd)
		case "POST":
			handlePost(hd)
		case "DELETE":
			handleDelete(hd)
		default:
			handleGet(hd)
		}
		log.Printf("%s\t%s\t%s",
			r.Method,
			r.RequestURI,
			time.Since(start),
		)
	}
}
