package nanocms_compiler

import (
	"fmt"
	"reflect"

	"github.com/go-yaml/yaml"
)

/*
A representation of an object tree, preserving ordering.
*/

type OTree struct {
	_data map[interface{}]interface{}
	_kidx []interface{}
}

func NewOTree() *OTree {
	return new(OTree).Flush()
}

// Flush the content of the tree
func (tree *OTree) Flush() *OTree {
	tree._data = make(map[interface{}]interface{})
	tree._kidx = make([]interface{}, 0)
	return tree
}

// LoadMapSlice loads a yaml.MapSlice object that keeps the ordering
func (tree *OTree) LoadMapSlice(data yaml.MapSlice) *OTree {
	for _, item := range data {
		kind := reflect.TypeOf(item.Value).Kind()
		switch kind {
		case reflect.Slice:
			tree.Set(item.Key, tree.getMapSlice(item.Value.(yaml.MapSlice), nil))
		case reflect.String:
			tree.Set(item.Key, item.Value)
		default:
			panic(fmt.Errorf("Unknown type '%s' while loading state", kind))
		}
	}
	return tree
}

func (tree *OTree) getArray(data interface{}) []interface{} {
	cnt := make([]interface{}, 0)

	for _, el := range data.([]interface{}) {
		cnt = append(cnt, tree.getMapSlice(el.(yaml.MapSlice), nil))
	}

	return cnt
}

func (tree *OTree) getMapSlice(data yaml.MapSlice, cnt *OTree) *OTree {
	if cnt == nil {
		cnt = NewOTree()
	}
	for _, item := range data {
		if item.Value != nil {
			kind := reflect.TypeOf(item.Value).Kind()
			switch kind {
			case reflect.Slice:
				i_val_t := reflect.TypeOf(item.Value)
				if i_val_t.Kind() == reflect.Slice && i_val_t.Elem().Kind() == reflect.Interface {
					cnt.Set(item.Key, tree.getArray(item.Value))
				} else {
					cnt.Set(item.Key, tree.getMapSlice(item.Value.(yaml.MapSlice), nil))
				}
			case reflect.String:
				cnt.Set(item.Key, item.Value)
			case reflect.Bool:
				cnt.Set(item.Key, item.Value)
			default:
				panic(fmt.Errorf("Unknown type '%s' while loading state", kind))
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

func (tree *OTree) _to_structure(cnt map[string]interface{}, obj interface{}) interface{} {
	if obj == nil {
		return nil
	}

	if cnt == nil {
		cnt = make(map[string]interface{})
	}
	objType := reflect.TypeOf(obj).Kind()
	if objType == reflect.Ptr {
		for _, obj_k := range obj.(*OTree).Keys() {
			cnt[obj_k.(string)] = tree._to_structure(nil, obj.(*OTree).Get(obj_k, nil))
		}
	} else if objType == reflect.Map {
		for obj_k := range obj.(map[interface{}]interface{}) {
			cnt[obj_k.(string)] = tree._to_structure(nil, obj.(map[interface{}]interface{})[obj_k])
		}
	} else if objType == reflect.Slice {
		arr := make([]interface{}, 0)
		for _, element := range obj.([]interface{}) {
			arr = append(arr, tree._to_structure(nil, element))
		}
		return arr
	} else if objType == reflect.String {
		return obj.(string)
	} else if objType == reflect.Bool {
		return obj.(bool)
	} else {
		fmt.Println("unsupported DSL type:", objType)
	}

	return cnt
}

// ToYAML exports ordered tree to an unordered YAML (!)
func (tree *OTree) ToYAML() string {
	obj := tree._to_structure(nil, tree._data)
	data, _ := yaml.Marshal(&obj)

	return string(data)
}

func (tree *OTree) Serialise() map[string]interface{} {
	obj := tree._to_structure(nil, tree._data)
	shallowObj := make(map[string]interface{})
	for k, v := range obj.(map[string]interface{}) {
		shallowObj[k] = v
	}
	return shallowObj
}
