package nanocms_compiler

import (
	"github.com/go-yaml/yaml"
	"reflect"
)

/*
A representation of an object tree, preserving ordering.
*/

type OTree struct {
	_data map[interface{}]interface{}
	_kidx []interface{}
}

func NewOTree() *OTree {
	otr := new(OTree)
	otr._data = make(map[interface{}]interface{})
	otr._kidx = make([]interface{}, 0)
	return otr
}

func (tree *OTree) LoadMapSlice(data yaml.MapSlice) *OTree {
	for _, item := range data {
		switch reflect.TypeOf(item.Value).Kind() {
		case reflect.Slice:
			tree.Set(item.Key, tree.getMapSlice(item.Value.(yaml.MapSlice), nil))
		default:
			tree.Set(item.Key, item.Value)
		}
	}
	return tree
}

func (tree *OTree) getArray(data []interface{}) []interface{} {
	cnt := make([]interface{}, 0)
	for _, elem := range data {
		switch reflect.TypeOf(elem).Elem().Kind() {
		case reflect.Struct:
			cnt = append(cnt, tree.getMapSlice(elem.(yaml.MapSlice), nil))
		default:
			cnt = append(cnt, elem)
		}
	}
	return cnt
}

func (tree *OTree) getMapSlice(data yaml.MapSlice, cnt *OTree) *OTree {
	if cnt == nil {
		cnt = NewOTree()
	}
	for _, item := range data {
		if item.Value != nil {
			switch reflect.TypeOf(item.Value).Kind() {
			case reflect.Slice:
				if reflect.TypeOf(item.Value).Elem().Kind() == reflect.Interface {
					cnt.Set(item.Key, tree.getArray(item.Value.([]interface{})))
				} else {
					cnt.Set(item.Key, tree.getMapSlice(item.Value.(yaml.MapSlice), cnt))
				}
			default:
				cnt.Set(item.Key, item.Value)
			}
		} else {
			cnt.Set(item.Key, nil)
		}
	}
	return cnt
}

// Set the key/value
func (tree *OTree) Set(key interface{}, value interface{}) *OTree {
	if tree.Exists(key) {
		tree._data[key] = value
	} else {
		tree._kidx = append(tree._kidx, key)
		tree._data[key] = value
	}
	return tree
}

// Get key with the default
func (tree *OTree) Get(key interface{}, bydefault interface{}) interface{} {
	if tree.Exists(key) {
		return tree._data[key]
	}
	return bydefault
}

// GetBranch of the current tree. If branch is not an OTree object or not found, nil is returned.
func (tree *OTree) GetBranch(key interface{}) *OTree {
	obj := tree.Get(key, nil)
	if reflect.TypeOf(obj).Elem().Kind() == reflect.Struct {
		return obj.(*OTree)
	}

	return nil
}

// GetList returns an object as an array of the interfaces. If an object is not a slice, nil is returned.
func (tree *OTree) GetList(key interface{}) []interface{} {
	obj := tree.Get(key, nil)
	if reflect.TypeOf(obj).Kind() == reflect.Slice {
		return obj.([]interface{})
	}
	return nil
}

// Check if key is there
func (tree *OTree) Exists(key interface{}) bool {
	_, ex := tree._data[key]
	return ex
}

// Delete key. Nothing happens if the key wasn't there.
func (tree *OTree) Delete(key interface{}) *OTree {
	if tree.Exists(key) {
		for i, k := range tree._kidx {
			if k == key {
				delete(tree._data, key)
				tree._kidx = append(tree._kidx[:i], tree._kidx[i+1:]...)
				return tree
			}
		}
	}
	return tree
}

// Return keys
func (tree *OTree) Keys() []interface{} {
	return tree._kidx
}

func (tree *OTree) Items() [][]interface{} {
	return nil
}
