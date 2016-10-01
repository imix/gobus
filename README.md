# GOBUS
## Introduction
gobus is a rest communication platform. It permits the communication between processes through rest calls. It can be used in microservice architectures, Internet of Things (IoT) or other rest based systems. There is no need for configuration before running gobus, all configuration (e.g. creation of resources) is done through rest calls. 

It has the following features:
  * items
  * collections
  * [resthooks](http://resthooks.org/) for "real-time" communication.
  * forwarding (reverse-proxy)

Gobus can handle any type of content, be it text or binary. Content related to the use of gobus (e.g. hooks) use json. It uses http://redis.io/ as datastore.

## Install
To build gobus you need to have [golang](https://golang.org/) installed. In the src directory in your GOPATH do the following.
```
git clone https://github.com/imix/gobus.git
cd gobus
go build
```
Additionally you need to install redis. You can install it via your operating system (e.g. dnf, apt-get) or from http://redis.io/topics/quickstart.

Optionally you can run the tests and view the coverage (redis needs to be running for the tests):
```
go test -coverprofile=coverage.out && go tool cover -func=coverage.out
```

## Usage
Before running gobus, redis has to be running. You can then start gobus by calling the executable:
```
./gobus
```
This will listen on localhost:8080 for your requests. To configure the port, set the environement variable PORT to the desired port.

Gobus can be used from any programming language supporting http calls. For go you can use [gbclient](https://github.com/imix/gbclient) to use gobus. The following examples use [curl](https://curl.haxx.se/) from the command line to show how to use gobus.


### Items
Creating a resource is done with a Put request
```
curl -X PUT -d '{ "some": "data" }' http://localhost:8080/my/item
```

To get the content of an item which has been created before, just make a regular GET request:
```
curl http://localhost:8080/my/item
```

Deleting is straightforward.
```
curl -X DELETE http://localhost:8080/my/item
```

### Collections
Creating a collection is done with an empty PUT request
```
curl -X PUT http://localhost:8080/my/collection
```

Adding an item to a collection is done with a POST request:
```
curl -X POST -d "some item data" http://localhost:8080/my/collection
```
The location of the newly created resource is returned in the "Location" header as well as in the body

To get the content of a collection, just make a regular GET request. This returns a list with all items in the collection.
```
curl http://localhost:8080/my/collection
```
Then you can get the items you like via another GET request.

Collections can only be deleted when they are empty.
```
curl -X DELETE http://localhost:8080/my/collection
```

### Hooks
If a process is interested in a certain resource, it can create a hook on it. Every time the hooked resource is modified (e.g. by a put), the hook url is called with details on what happend in a json structure.
Creating a hook is done with a POST request to the url "\_hooks" with json data, the resource has to exist:
```
curl -X POST -d '{"name": "a_hook", "url": "http://localhost:8090/my/hook"}' http://localhost:8080/my/item/_hooks
```
The json data in the response has the following fileds:
  * name: the name of the hook
  * method: the method that was called on the resource
  * item: whether the modified resource is an item or a collection
  * url: the url to the modified resource


### Forwards
Gobus can act as a reverse proxy on defined resources. To forward any calls to /my/item to http://localhost:8090/my/item/\_forward, perform a PUT on an existing resource with the following json content:
```
curl -X PUT -d '{"url": "http://localhost:8090/my/forward"}' http://localhost:8080/my/item/_forward
```
Everything except commands is forwarded to the given url. This permits e.g. to delete a forward:
```
curl -X DELETE http://localhost:8080/my/item/_forward
```
