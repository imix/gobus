# GOBUS
## Introduction
gobus is a rest communication platform. It permits the communication between processes through rest calls. There is no need for configuration before running gobus, all configuration (e.g. creation of resources) is done through rest calls. Gobus supports [resthooks](http://resthooks.org/) for "real-time" communication.

It currently uses an in-memory datastore which is never saved to disk. This means your data is lost once you stop gobus.

## Install
To build gobus you need to have [golang](https://golang.org/) installed. In the src directory in your GOPATH do the following.
```
git clone https://github.com/imix/gobus.git
cd gobus
go build
```

## Usage
You can start gobus by calling the executable:
```
./gobus
```
This will listen on localhost:8080 for your requests. This is not yet configurable.

### Creating Resources
Creating a resource is done with a Put request
```
curl -X PUT -d '{ "some": "data" }' http://localhost:8080/my/item
```

Creating a collection is done with an empty Put request
```
curl -X PUT http://localhost:8080/my/collection
```

Adding an item to a collection is done with a POST request:
```
curl -X POST -d "some item data" http://localhost:8080/my/collection
```
The location of the newly created resource is returned in the "Location" header as well as in the body

### Getting a resource
To get the content of a resource which has been created before, just make a regular GET request:
```
curl http://localhost:8080/my/item
```

### Hooks
If a process is interested in a certain resource, it can create a hook on it. Every time the hooked resource is modified (e.g. by a put), the hook url is called with details on what happend in a json structure.
Creating a hook is done with a Put request with json data:
```
curl -X PUT -d '{"name": "a_hook", "url": "http://localhost:8090/my/hook"}' http://localhost:8080/my/item
```
The json data in the response has the following fileds:
  * name: the name of the hook
  * method: the method that was called on the resource
  * item: whether the modified resource is an item or a collection
  * url: the url to the modified resource


### Deleting
Deleting is straightforward. Only "leaf-resources" can be deleted.
```
curl -X DELETE http://localhost:8080/my/item
```
