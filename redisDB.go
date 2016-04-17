package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/redis.v3"
)

type RedisDB struct {
	Client *redis.Client
}

type RedisResource struct {
	db       *RedisDB
	elts     []string
	key      string
	childKey string
	hookKey  string
}

const (
	nameField        = "name"
	itemField        = "item"
	valueField       = "value"
	contentTypeField = "contentType"
	nextIDField      = "nextID"
	nextHookIDField  = "nextHookID"
)

func NewRedisDB() GoBusDB {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	return &RedisDB{client}
}

func mkKeys(elts []string) (string, string, string) {
	key := "root:" + strings.Join(elts, ":")
	childKey := key + ":_children"
	hookKey := key + ":_hooks"
	return key, childKey, hookKey
}

func (db *RedisDB) addResource(elts []string, value, ct, item string) (Resource, error) {
	key, childKey, hookKey := mkKeys(elts)
	name := elts[len(elts)-1]
	db.Client.HSet(key, nameField, name)
	db.Client.HSet(key, valueField, value)
	db.Client.HSet(key, contentTypeField, ct)
	db.Client.HSet(key, itemField, item)
	db.Client.HSet(key, nextIDField, "0")
	db.Client.HSet(key, nextHookIDField, "0")
	parent, err := db.GetResource(elts[:len(elts)-1])
	if err != nil {
		return nil, err
	}
	err = parent.(*RedisResource).AddChildKey(key)
	if err != nil {
		return nil, err
	}
	return &RedisResource{db, elts, key, childKey, hookKey}, nil
}

// Creates the resource defined by the given path
// Missing intermediate resources are automatically created
// The item flag is set on the last resource
// If the resource exists already, an error is returned
func (db *RedisDB) CreateResource(elts []string, item bool) (Resource, error) {
	// check intermediary resources
	for i, _ := range elts[:len(elts)-1] {
		exists, _ := db.ResourceExists(elts[:i+1])
		if !exists {
			db.addResource(elts[:i+1], "", "", "false")
		}
	}
	return db.addResource(elts, "", "", strconv.FormatBool(item))
}

// checks to see if the resource exists
func (db *RedisDB) ResourceExists(elts []string) (bool, error) {
	key, _, _ := mkKeys(elts)
	exists, err := db.Client.Exists(key).Result()
	if err != nil {
		return false, err
	}
	return exists, err
}

// searches the resource identified by the given path and returns it
// if the Resource could not be found returns an error
func (db *RedisDB) GetResource(elts []string) (Resource, error) {
	if len(elts) == 0 { // root resource
		return &RedisResource{db, []string{}, "root", "root:_children", "root:_hooks"}, nil
	}
	key, childKey, hookKey := mkKeys(elts)
	exists, err := db.Client.Exists(key).Result()
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New(fmt.Sprintf("Resource not found: %s", key))
	}
	return &RedisResource{db, elts, key, childKey, hookKey}, nil
}

// deletes a resource
// delete non-leaf resources generates an error
func (db *RedisDB) DeleteResource(elts []string) error {
	key, childKey, hookKey := mkKeys(elts)
	if len(elts) < 1 {
		return errors.New(fmt.Sprintf("Can not delete root resource %s", key))
	}
	res, err := db.GetResource(elts)
	if err != nil {
		return err
	}
	children, err := res.GetChildren()
	if err != nil {
		return err
	}
	if len(children) != 0 {
		return errors.New(fmt.Sprintf("Can not delete non-leaf resource %s", key))
	}
	parent, err := db.GetResource(elts[:len(elts)-1])
	if err != nil {
		return err
	}
	err = parent.(*RedisResource).RemoveChildKey(key)
	if err != nil {
		return err
	}
	return db.Client.Del(key, childKey, hookKey).Err()
}

// helper to add child key to the list of children
// Does not create the child!
func (r *RedisResource) AddChildKey(key string) error {
	return r.db.Client.SAdd(r.childKey, key).Err()
}

// helper to remove child key from the list of children
// Does not delete the child!
func (r *RedisResource) RemoveChildKey(key string) error {
	return r.db.Client.SRem(r.childKey, key).Err()
}

// returns the content-type and the value of a resource
func (r *RedisResource) GetValue() (string, []byte, error) {
	contentType, err := r.db.Client.HGet(r.key, contentTypeField).Result()
	if err != nil {
		return "", nil, err
	}
	value, err := r.db.Client.HGet(r.key, valueField).Result()
	if err != nil {
		return "", nil, err
	}
	return contentType, []byte(value), nil
}

// returns a list with all children's IDs
func (r *RedisResource) GetChildren() ([]string, error) {
	children, err := r.db.Client.SMembers(r.childKey).Result()
	if err != nil {
		return nil, err
	}
	return children, nil
}

func (r *RedisResource) SetValue(contentType string, value []byte) error {
	r.db.Client.HSet(r.key, valueField, string(value))
	r.db.Client.HSet(r.key, contentTypeField, contentType)
	return nil
}

// adds a resource to a collection
// the resource may not be an item
func (r *RedisResource) AddToCollection(contentType string, data []byte) (string, error) {
	item, err := r.db.Client.HGet(r.key, itemField).Result()
	if err != nil {
		return "", err
	}
	if item != "false" {
		return "", errors.New("Can not add to item")
	}
	nextId, err := r.db.Client.HIncrBy(r.key, nextIDField, 1).Result()
	if err != nil {
		return "", err
	}
	name := strconv.FormatInt(nextId-1, 10)
	newElts := append(r.elts, name)
	r.db.addResource(newElts, string(data), contentType, "true")

	return name, nil
}

func (r *RedisResource) Name() (string, error) {
	name, err := r.db.Client.HGet(r.key, nameField).Result()
	if err != nil {
		return "", err
	}
	return name, nil
}

func (r *RedisResource) IsItem() (bool, error) {
	itemStr, err := r.db.Client.HGet(r.key, itemField).Result()
	if err != nil {
		return false, err
	}
	item, err := strconv.ParseBool(itemStr)
	if err != nil {
		return false, err
	}
	return item, nil
}

func (r *RedisResource) AddHook(data []byte) (string, error) {
	nextId, err := r.db.Client.HIncrBy(r.key, nextHookIDField, 1).Result()
	if err != nil {
		return "", err
	}
	hook, err := parseHook(data) // check if hook parses ok
	if err != nil {
		return "", err
	}
	hook.Id = strconv.FormatInt(nextId-1, 10)
	hookData, err := json.Marshal(hook)
	if err != nil {
		return "", err
	}
	r.db.Client.HSet(r.hookKey, hook.Id, string(hookData))
	return hook.Id, nil
}

func (r *RedisResource) DeleteHook(id string) error {
	result, err := r.db.Client.HDel(r.hookKey, id).Result()
	if err != nil {
		return err
	}
	if result < 1 {
		return errors.New(fmt.Sprintf("Could not find hook with id: %s", id))
	}
	return nil
}

func (r *RedisResource) GetHooks() ([]*Hook, error) {
	keys, err := r.db.Client.HKeys(r.hookKey).Result()
	if err != nil {
		return nil, err
	}
	hooks := []*Hook{}
	for _, k := range keys {
		data, err := r.db.Client.HGet(r.hookKey, k).Result()
		if err != nil {
			return nil, err
		}
		parsedHook, err := parseHook([]byte(data)) // check if hook parses ok
		if err != nil {
			return nil, err
		}
		hooks = append(hooks, parsedHook)
	}
	return hooks, nil
}
