package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"path"
)

type HookCollection struct {
	Hooks      []*Hook `json:"hooks"`
	NextHookId int     `json:"-"`
}

type Hook struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
	//Fields      []string // limit fields to get
	//Methods     []string // limit methods to forward
}

type HookEvent struct {
	Name             string `json:"name"`
	Method           string `json:"method"`
	Item             bool   `json:"item"` // is the affected resource an item or a collection
	ModifiedResource string `json:"path"` // relative path to the resource (caller needs to know server)
}

func parseHook(data []byte) (*Hook, error) {
	var hook Hook
	err := json.Unmarshal(data, &hook)
	return &hook, err

}

func callHooks(res Resource, method, basePath string) {
	comps := res.GetElts()
	resURL := path.Join(basePath, path.Join(comps...))
	hooks, err := res.GetHooks()
	if err != nil {
		log.Printf("Internal error, could not get hooks: %v", err.Error())
		return
	}
	isitem, err := res.IsItem()
	if err != nil {
		panic(err)
		log.Printf("Internal error, could not get isitem: %v", err.Error())
		return
	}
	for _, h := range hooks {
		var event = HookEvent{
			Name:             h.Name,
			Method:           method,
			Item:             isitem,
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

// hook handlers
// deletes an existing hook
func deleteHook(hd *HandlerData, res Resource, cmds []string) {
	if len(cmds) != 2 {
		respond(hd, http.StatusNotFound, "Not Found")
		return
	}
	err := res.DeleteHook(cmds[1])
	if err != nil {
		respond(hd, http.StatusNotFound, "Not Found")
		return
	}
	respond(hd, http.StatusOK, "Deleted")
}

// returns either a single hook specified by the ID or a list of all hooks
func getHook(hd *HandlerData, res Resource, cmds []string) {
	w := hd.W
	switch len(cmds) {
	case 1: // get on the _hooks collection
		ids, err := res.GetHooksIDs()
		if err != nil {
			respond(hd, http.StatusInternalServerError, "Could not get Hooks")
			break
		}
		data, err := json.Marshal(ids)
		if err != nil {
			respond(hd, http.StatusInternalServerError, "Could not get Hooks Json")
			break
		}
		w.Write(data)
	case 2: // get a specific hook
		hook, err := res.GetHook(cmds[1])
		if err != nil {
			respond(hd, http.StatusInternalServerError, "Could not get Hook")
			break
		}
		data, err := json.Marshal(hook)
		if err != nil {
			respond(hd, http.StatusInternalServerError, "Could not get Hook Json")
			break
		}
		w.Write(data)
	default:
		respond(hd, http.StatusNotFound, "Hooks do not have sub-elements.")
	}
}

// create a new hook, returns the ID of the hook
func postHook(hd *HandlerData, res Resource, cmds []string) {
	if len(cmds) == 1 {
		body := hd.R.Body
		data, err := ioutil.ReadAll(body)
		body.Close()
		if err != nil {
			respond(hd, http.StatusBadRequest, "Invalid Request")
			return
		}
		name, err := res.AddHook(data)
		if err != nil {
			respond(hd, http.StatusInternalServerError, "Could not create Hook")
			return
		}
		respondCreatedNewURL(hd.W, hd.R.URL, name)
	} else {
		respond(hd, http.StatusMethodNotAllowed, "Method not allowed for hooks.")
	}
}

// puts a hook, only permitted for existing hooks
// new hooks have to be created with post
func putHook(hd *HandlerData, res Resource, cmds []string) {
	if len(cmds) == 2 {
		_, err := res.GetHook(cmds[1])
		if err != nil {
			respond(hd, http.StatusInternalServerError, "Could not get Hook")
			return
		}
		body := hd.R.Body
		data, err := ioutil.ReadAll(body)
		body.Close()
		if err != nil {
			respond(hd, http.StatusBadRequest, "Invalid Request")
			return
		}
		err = res.SetHook(cmds[1], data)
		if err != nil {
			respond(hd, http.StatusInternalServerError, "Could not set Hook")
			return
		}
		respond(hd, http.StatusOK, "Hook updated.")
	} else {
		respond(hd, http.StatusMethodNotAllowed, "Put only allowed on existing hooks.")
	}
}

// handles requests for the _hook command
func handleHookRequest(hd *HandlerData, res Resource, cmds []string) {
	switch hd.R.Method {
	case "DELETE":
		deleteHook(hd, res, cmds)
	case "GET":
		getHook(hd, res, cmds)
	case "POST":
		postHook(hd, res, cmds)
	case "PUT":
		putHook(hd, res, cmds)
	default:
		respond(hd, http.StatusMethodNotAllowed, "Method not allowed for hooks.")
	}
}
