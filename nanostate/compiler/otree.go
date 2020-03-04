package nanocms_compiler

import (
	"fmt"
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
	if obj != nil && reflect.TypeOf(obj).Elem().Kind() == reflect.Struct {
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

// GetString returns a string, blindly assuming it is one.
// XXX: better implementation needed. :)
func (tree *OTree) GetString(key interface{}) string {
	return tree.Get(key, nil).(string)
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

func (tree *OTree) _to_map(cnt map[string]interface{}, obj interface{}) interface{} {
	if cnt == nil {
		cnt = make(map[string]interface{})
	}

	if reflect.TypeOf(obj).Kind() == reflect.Ptr {
		for _, k := range obj.(*OTree).Keys() {
			v := obj.(*OTree).Get(k, nil)
			if v == nil {
				cnt[k.(string)] = nil
			} else {
				if reflect.TypeOf(v).Kind() == reflect.String {
					cnt[k.(string)] = v
				} else {
					cnt[k.(string)] = tree._to_map(nil, v)
				}
			}
		}
	} else if reflect.TypeOf(obj).Kind() == reflect.Map {
		for k, v := range obj.(map[interface{}]interface{}) {
			if reflect.TypeOf(v).Kind() == reflect.String {
				cnt[k.(string)] = v
			} else {
				cnt[k.(string)] = tree._to_map(nil, v)
			}
		}
	} else if reflect.TypeOf(obj).Kind() == reflect.Slice {
		ret := make([]interface{}, 0)
		for _, k := range obj.([]interface{}) {
			if reflect.TypeOf(k).Kind() == reflect.Ptr {
				ret = append(ret, tree._to_map(nil, k))
			} else {
				fmt.Println("unsupported DSL structure at:", reflect.TypeOf(k).Kind(), k)
			}
		}
		return ret
	} else {
		fmt.Println("unsupported DSL type:", reflect.TypeOf(obj).Kind())
	}
	return cnt
}

// ToYAML exports ordered tree to an unordered YAML (!)
func (tree *OTree) ToYAML() string {
	obj := tree._to_map(nil, tree._data)
	data, _ := yaml.Marshal(&obj)

	return string(data)
}
