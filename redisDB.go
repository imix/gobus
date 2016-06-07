package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/bsm/redis-lock"

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
	forward  string
	lock     *lock.Lock
}

const (
	nameField        = "name"
	itemField        = "item"
	valueField       = "value"
	contentTypeField = "contentType"
	nextIDField      = "nextID"
	nextHookIDField  = "nextHookID"
	forwardField     = "forward"
)

func NewRedisDB() GoBusDB {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	return &RedisDB{client}
}

func mkKeys(elts []string) (string, string, string, error) {
	for _, e := range elts {
		if isCommand(e) || strings.HasSuffix(e, "-lock") {
			return "", "", "", errors.New(fmt.Sprintf("Path contains illegal name %s", e))
		}
	}
	key := "root:" + strings.Join(elts, ":")
	childKey := key + ":_children"
	hookKey := key + ":_hooks"
	return key, childKey, hookKey, nil
}

func mkResource(db *RedisDB, elts []string, key, childKey, hookKey, forward string) Resource {
	lock := lock.NewLock(db.Client, key+"-lock", nil)
	return &RedisResource{db, elts, key, childKey, hookKey, forward, lock}
}

func (db *RedisDB) addResource(elts []string, value, ct, item string) (Resource, error) {
	key, childKey, hookKey, err := mkKeys(elts)
	if err != nil {
		return nil, err
	}

	name := elts[len(elts)-1]
	db.Client.HSet(key, nameField, name)
	db.Client.HSet(key, valueField, value)
	db.Client.HSet(key, contentTypeField, ct)
	db.Client.HSet(key, itemField, item)
	db.Client.HSet(key, nextIDField, "0")
	db.Client.HSet(key, nextHookIDField, "0")
	db.Client.HSet(key, forwardField, "{}")
	parent, err := db.GetResource(elts[:len(elts)-1])
	if err != nil {
		return nil, err
	}
	err = parent.(*RedisResource).addChildKey(key)
	if err != nil {
		return nil, err
	}
	return mkResource(db, elts, key, childKey, hookKey, "{}"), nil
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
	key, _, _, err := mkKeys(elts)
	if err != nil {
		return false, err
	}
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
		return mkResource(db, []string{}, "root", "root:_children", "root:_hooks", "{}"), nil
	}
	key, childKey, hookKey, err := mkKeys(elts)
	if err != nil {
		return nil, err
	}
	exists, err := db.Client.Exists(key).Result()
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New(fmt.Sprintf("Resource not found: %s", key))
	}
	forward, err := db.Client.HGet(key, forwardField).Result()
	if err != nil {
		return nil, err
	}
	return mkResource(db, elts, key, childKey, hookKey, forward), nil
}

// deletes a resource
// delete non-leaf resources generates an error
func (r *RedisResource) Delete() error {
	r.lock.Lock()
	defer r.lock.Unlock()
	elts, key, childKey, hookKey := r.elts, r.key, r.childKey, r.hookKey
	if len(elts) == 0 {
		return errors.New(fmt.Sprintf("Can not delete root resource %s", key))
	}
	children, err := r.getChildren()
	if err != nil {
		return err
	}
	if len(children) != 0 {
		return errors.New(fmt.Sprintf("Can not delete non-leaf resource %s", key))
	}
	parent, err := r.db.GetResource(elts[:len(elts)-1])
	if err != nil {
		return err
	}
	err = parent.(*RedisResource).removeChildKey(key)
	if err != nil {
		return err
	}
	return r.db.Client.Del(key, childKey, hookKey).Err()
}

// helper to add child key to the list of children
// Does not create the child!
func (r *RedisResource) AddChildKey(key string) error {
	return r.addChildKey(key)
}

func (r *RedisResource) addChildKey(key string) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.db.Client.SAdd(r.childKey, key).Err()
}

// helper to remove child key from the list of children
// Does not delete the child!
func (r *RedisResource) RemoveChildKey(key string) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.removeChildKey(key)
}

func (r *RedisResource) removeChildKey(key string) error {
	return r.db.Client.SRem(r.childKey, key).Err()
}

// returns the content-type and the value of a resource
func (r *RedisResource) GetValue() (string, []byte, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
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
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.getChildren()
}

func (r *RedisResource) getChildren() ([]string, error) {
	children, err := r.db.Client.SMembers(r.childKey).Result()
	if err != nil {
		return nil, err
	}
	return children, nil
}

func (r *RedisResource) SetValue(contentType string, value []byte) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.db.Client.HSet(r.key, valueField, string(value))
	r.db.Client.HSet(r.key, contentTypeField, contentType)
	return nil
}

// adds a resource to a collection
// the resource may not be an item
func (r *RedisResource) AddToCollection(contentType string, data []byte) (string, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
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
	r.lock.Lock()
	defer r.lock.Unlock()
	name, err := r.db.Client.HGet(r.key, nameField).Result()
	if err != nil {
		return "", err
	}
	return name, nil
}

func (r *RedisResource) IsItem() (bool, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
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

func (r *RedisResource) GetElts() []string {
	return r.elts
}

// creates a new hook, generating a new ID for the hook
func (r *RedisResource) AddHook(data []byte) (string, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	nextId, err := r.db.Client.HIncrBy(r.key, nextHookIDField, 1).Result()
	if err != nil {
		return "", err
	}
	id := strconv.FormatInt(nextId-1, 10)
	err = r.setHook(id, data)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *RedisResource) GetHook(id string) (*Hook, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.getHook(id)
}

func (r *RedisResource) getHook(id string) (*Hook, error) {
	data, err := r.db.Client.HGet(r.hookKey, id).Result()
	if err != nil {
		return nil, err
	}
	parsedHook, err := parseHook([]byte(data)) // check if hook parses ok
	if err != nil {
		return nil, err
	}
	return parsedHook, nil
}

// sets the value of a hook with a known ID
func (r *RedisResource) SetHook(id string, data []byte) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.setHook(id, data)
}

// sets the value of a hook with a known ID
// internal version without locking
func (r *RedisResource) setHook(id string, data []byte) error {
	hook, err := parseHook(data) // check if hook parses ok
	if err != nil {
		return err
	}
	hook.Id = id // make sure hook ID is correct
	hookData, err := json.Marshal(hook)
	if err != nil {
		return err
	}
	r.db.Client.HSet(r.hookKey, hook.Id, string(hookData))
	return nil
}

func (r *RedisResource) DeleteHook(id string) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	result, err := r.db.Client.HDel(r.hookKey, id).Result()
	if err != nil {
		return err
	}
	if result < 1 {
		return errors.New(fmt.Sprintf("Could not find hook with id: %s", id))
	}
	return nil
}

func (r *RedisResource) GetHooksIDs() ([]string, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.getHooksIDs()
}

func (r *RedisResource) getHooksIDs() ([]string, error) {
	keys, err := r.db.Client.HKeys(r.hookKey).Result()
	if err != nil {
		return nil, err
	}
	return keys, nil
}

func (r *RedisResource) GetHooks() ([]*Hook, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	keys, err := r.getHooksIDs()
	if err != nil {
		return nil, err
	}
	hooks := []*Hook{}
	for _, k := range keys {
		parsedHook, err := r.getHook(k)
		if err != nil {
			return nil, err
		}
		hooks = append(hooks, parsedHook)
	}
	return hooks, nil
}

func (r *RedisResource) GetForward() (*Forward, error) {
	return parseForward([]byte(r.forward))
}

// sets the value of the forward
func (r *RedisResource) AddForward(data []byte) error {
	r.lock.Lock()
	defer r.lock.Unlock()
	forward, err := parseForward(data) // check if hook parses ok
	if err != nil {
		return err
	}
	forwardData, err := json.Marshal(forward)
	if err != nil {
		return err
	}
	err = r.db.Client.HSet(r.key, forwardField, string(forwardData)).Err()
	if err != nil {
		return err
	}
	r.forward = string(forwardData)
	return nil
}

func (r *RedisResource) DeleteForward() error {
	r.lock.Lock()
	defer r.lock.Unlock()
	err := r.db.Client.HSet(r.key, forwardField, "{}").Err()
	if err != nil {
		return err
	}
	r.forward = "{}"
	return nil
}
