# GOBUS
## Introduction

## Install
```
go build
```

## Usage
You can start gobus by calling the executable:
```
./gobus
```

### Creating Resources
Creating a resource is done with a Put request
```
curl -X PUT -d '{ "some": "data" }' http://localhost:8080/my/item
```

Creating a collection is done with an empty Put request
```
curl -X PUT http://localhost:8080/my/collection
```

### Hooks

### Deleting
