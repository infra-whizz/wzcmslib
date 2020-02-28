/*
Basic type converter from Starlark to Go interface
*/

package nanocms_compiler

import (
	"fmt"
	"go.starlark.net/starlark"
)

type StarTuple struct {
	v []interface{}
}

func NewStarTuple() *StarTuple {
	st := new(StarTuple)
	st.v = make([]interface{}, 0)
	return st
}

func (st *StarTuple) Value() []interface{} {
	return st.v
}

////////
type StarType struct {
	v starlark.Value
}

// Constructor
func NewStarType(v starlark.Value) *StarType {
	st := new(StarType)
	st.v = v
	return st
}

// StarType tells a type of returned Starlark value
func (st *StarType) StarType() string {
	return st.v.Type()
}

func (st *StarType) parseTuple(t starlark.Tuple) *StarTuple {
	out := NewStarTuple()
	for i := 0; i < t.Len(); i++ {
		v := t.Index(i)
		switch v.Type() {
		case "string":
			out.v = append(out.v, v.(starlark.String).GoString())
		case "int":
			out.v = append(out.v, st.parseInt(v.(starlark.Int)))
		case "tuple":
			out.v = append(out.v, st.parseTuple(v.(starlark.Tuple)))
		case "dict":
			out.v = append(out.v, st.parseDict(v.(*starlark.Dict)))
		case "list":
			out.v = append(out.v, st.parseList(v.(*starlark.List)))
		default:
			out.v = append(out.v, v)
		}
	}
	return out
}

func (st *StarType) parseBool(b starlark.Bool) bool {
	return bool(b)
}

func (st *StarType) parseInt(i starlark.Int) int64 {
	out, _ := i.Int64()
	return out
}

func (st *StarType) parseHashable(h starlark.Value) interface{} {
	var out interface{}
	switch h.Type() {
	case "string":
		out = h.(starlark.String).GoString()
	case "int":
		out = st.parseInt(h.(starlark.Int))
	case "tuple":
		out = st.parseTuple(h.(starlark.Tuple))
	case "bool":
		out = st.parseBool(h.(starlark.Bool))
	default:
		fmt.Println("DEBUG: Uncasted hashable:", h.Type())
		out = h
	}
	return out
}

func (st *StarType) parseDict(d *starlark.Dict) map[interface{}]interface{} {
	out := make(map[interface{}]interface{})
	for _, k := range d.Keys() {
		v, _, _ := d.Get(k)
		hk := st.parseHashable(k)
		switch v.Type() {
		case "dict":
			out[hk] = st.parseDict(v.(*starlark.Dict))
		case "list":
			out[hk] = st.parseList(v.(*starlark.List))
		case "string":
			out[hk] = v.(starlark.String).GoString()
		case "int":
			out[hk] = st.parseInt(v.(starlark.Int))
		case "bool":
			out[hk] = st.parseBool(v.(starlark.Bool))
		default:
			fmt.Println("DEBUG:", v.Type())
			out[hk] = v
		}
	}
	return out
}

func (st *StarType) parseList(l *starlark.List) []interface{} {
	var out []interface{}
	for i := 0; i < l.Len(); i++ {
		v := l.Index(i)
		switch v.Type() {
		case "dict":
			out = append(out, st.parseDict(v.(*starlark.Dict)))
		case "list":
			out = append(out, st.parseList(v.(*starlark.List)))
		case "tuple":
			out = append(out, st.parseTuple(v.(starlark.Tuple)))
		case "int":
			out = append(out, st.parseInt(v.(starlark.Int)))
		case "bool":
			out = append(out, st.parseBool(v.(starlark.Bool)))
		case "string":
			out = append(out, v.(starlark.String).GoString())
		default:
			out = append(out, v)
		}
	}
	return out
}

// Interface returns Go interface from Starlark type
func (st *StarType) Interface() interface{} {
	var out interface{}
	switch st.StarType() {
	case "list":
		out = st.parseList(st.v.(*starlark.List))
	case "dict":
		out = st.parseDict(st.v.(*starlark.Dict))
	case "tuple":
		out = st.parseTuple(st.v.(starlark.Tuple))
	case "int":
		out = st.parseInt(st.v.(starlark.Int))
	case "bool":
		out = st.parseBool(st.v.(starlark.Bool))
	case "string":
		out = st.v.(starlark.String).GoString()
	default:
		out = st.v
	}
	return out
}
