package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"strings"
)

type Forward struct {
	URL string `json:"url"`
}

func parseForward(data []byte) (*Forward, error) {
	var forward Forward
	err := json.Unmarshal(data, &forward)
	return &forward, err
}

// checks comps if a resource in the path of the request has a forward defined
// returns the first resource which has the forward defined
// returns nil when no forward was found
// returns nil when the resource with the forward is followed by a command
func getForwardResource(hd *HandlerData, comps, cmds []string) (Resource, error) {
	for i, _ := range comps {
		exists, err := hd.DB.ResourceExists(comps[:i+1])
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, nil
		}
		res, err := hd.DB.GetResource(comps[:i+1])
		forward, err := res.GetForward()
		if err != nil {
			return nil, err
		}
		if forward.URL != "" {
			if (len(comps) == i+1) && (len(cmds) > 0) {
				return nil, nil
			}
			return res, nil
		}
	}
	return nil, nil
}

// handles requests to the _forward command
// all calls to URLs below this resource are forwarded to the given destination
// headers, body and the remaining URL are forwarded untouched to the destination
// the response is returned to the caller
func forwardRequest(hd *HandlerData, res Resource, cmds []string) {
	forward, err := res.GetForward()
	if err != nil {
		respond(hd, http.StatusInternalServerError, "Could not get Forward")
		return
	}
	elts := res.GetElts()
	if err != nil {
		respond(hd, http.StatusInternalServerError, "Could not get Forward Elts")
		return
	}

	thisPath := path.Join(hd.BaseURL.Path, path.Join(elts...))
	relativePath := strings.TrimPrefix(hd.R.URL.Path, thisPath)
	target, err := url.Parse(forward.URL)
	if err != nil {
		respond(hd, http.StatusInternalServerError, "Could not get Parse Forward")
		return
	}

	targetQuery := target.RawQuery
	director := func(req *http.Request) {
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = path.Join(target.Path, relativePath)
		if targetQuery == "" || req.URL.RawQuery == "" {
			req.URL.RawQuery = targetQuery + req.URL.RawQuery
		} else {
			req.URL.RawQuery = targetQuery + "&" + req.URL.RawQuery
		}
	}
	reverseProxy := &httputil.ReverseProxy{Director: director}

	reverseProxy.ServeHTTP(hd.W, hd.R)
}

// deletes an existing forward
func deleteForward(hd *HandlerData, res Resource, cmds []string) {
	if len(cmds) != 1 {
		respond(hd, http.StatusNotFound, "Not Found")
		return
	}
	err := res.DeleteForward()
	if err != nil {
		respond(hd, http.StatusNotFound, "Not Found")
		return
	}
	respond(hd, http.StatusOK, "Forward Deleted")
}

// returns the forward if it exists
func getForward(hd *HandlerData, res Resource, cmds []string) {
	if len(cmds) != 1 {
		respond(hd, http.StatusNotFound, "Not Found")
		return
	}
	forward, err := res.GetForward()
	if err != nil {
		respond(hd, http.StatusNotFound, "Not Found")
		return
	}
	data, err := json.Marshal(forward)
	if err != nil {
		respond(hd, http.StatusInternalServerError, "Could not get Forwards Json")
		return
	}
	hd.W.Write(data)
}

// create a new hook, returns the ID of the hook
func putForward(hd *HandlerData, res Resource, cmds []string) {
	if len(cmds) != 1 {
		respond(hd, http.StatusNotFound, "Not Found")
		return
	}
	body := hd.R.Body
	data, err := ioutil.ReadAll(body)
	body.Close()
	if err != nil {
		respond(hd, http.StatusBadRequest, "Invalid Request")
		return
	}
	err = res.AddForward(data)
	if err != nil {
		respond(hd, http.StatusInternalServerError, "Could not add Forward")
		return
	}
	respond(hd, http.StatusOK, "Forward put.")
}

// handles requests for the _forward command
func handleForwardRequest(hd *HandlerData, res Resource, cmds []string) {
	switch hd.R.Method {
	case "DELETE":
		deleteForward(hd, res, cmds)
	case "GET":
		getForward(hd, res, cmds)
	case "PUT":
		putForward(hd, res, cmds)
	default:
		respond(hd, http.StatusMethodNotAllowed, "Method not allowed for forwards.")
	}
}
