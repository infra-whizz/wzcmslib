/*
Nanostate compiler.

Currently just a static YAML instructions loader according to the Nanostate specs.
*/

package nanocms_compiler

import (
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-yaml/yaml"
	"io/ioutil"
	"os"
	"strings"
)

type NstCompiler struct {
	// Index of all states that should be included
	_states     map[string]*OTree
	_functions  *CDLFunc
	_unresolved *RefList
	tree        *OTree
	rootStateId string
}

func NewNstCompiler() *NstCompiler {
	nstc := new(NstCompiler)
	nstc.tree = nil
	nstc._states = make(map[string]*OTree)
	nstc._unresolved = NewRefList()
	nstc._functions = NewCDLFunc()

	return nstc
}

// LoadFile loads a nanostate from the YAML file
func (nstc *NstCompiler) LoadFile(nstpath string) error {
	var err error
	if !strings.HasSuffix(nstpath, ".st") { // This is not a storage file from IBM's Lotus Domino :-)
		err = errors.New("State file should have suffix \".st\"")
	} else {
		fh, err := os.Open(nstpath)
		if err == nil {
			defer fh.Close()
			data, err := ioutil.ReadAll(fh)
			if err == nil {
				id, err := nstc.loadBytes(data)
				nstc.loadStarlarkFile(id, strings.TrimSuffix(nstpath, ".st")+".fc")
				return err
			}
		}
	}

	return err
}

// Load starlark file. This is optional step, since the file is also optional.
func (nstc *NstCompiler) loadStarlarkFile(id string, srcpath string) {
	nfo, err := os.Stat(srcpath)
	if err == nil && nfo.Mode().IsRegular() {
		nstc._functions.ImportSource(id, srcpath)
	}
}

// Load bytes of the state
func (nstc *NstCompiler) loadBytes(src []byte) (string, error) {
	var data yaml.MapSlice
	var err error
	if err := yaml.Unmarshal(src, &data); err != nil {
		panic(err)
	}
	state := NewOTree().LoadMapSlice(data)

	if state.Exists("id") {
		if nstc.rootStateId == "" {
			nstc.rootStateId = state.GetString("id")
		}
		nstc._states[state.GetString("id")] = state
	} else {
		err = fmt.Errorf("%s", "State has no ID")
	}

	nstc._unresolved.FindRefs(state)
	nstc._unresolved.MarkStateResolved(state.GetString("id"))

	return state.GetString("id"), err
}

// Cycle compiles current state and returns a next state Id to be found and loaded, if any.
// If returns an empty string, then no more cycles are found and Tree is ready.
func (nstc *NstCompiler) Cycle() string {
	// Resolve includes
	for _, id := range nstc._unresolved.GetIncluded() {
		return nstc._unresolved.MarkStateRequested(id)
	}

	// Everything seems resolved, compile now
	if err := nstc.compile(); err != nil {
		panic(err)
	}

	return ""
}

func (nstc *NstCompiler) Dump() {
	spew.Dump(nstc.Tree())
}

// Tree returns completed tree
func (nstc *NstCompiler) Tree() *OTree {
	if len(nstc._unresolved.GetIncluded()) > 0 {
		panic("Calling for compiled tree when unresolved sources are still pending")
	}

	return nstc.tree
}

// Resolve includes
func (nstc *NstCompiler) resolveIncludes() {

}

func (nstc *NstCompiler) compileJob(intree []interface{}) []interface{} {
	return nil
}

func (nstc *NstCompiler) compileStateJobs(intree map[interface{}]interface{}) map[string]interface{} {
	return nil
}

// Compile branch of the state
func (nstc *NstCompiler) compileState(state *OTree) *OTree {
	tree := NewOTree()

	branch := state.GetBranch("state")
	for _, _blockdef := range branch.Keys() {
		blockdef := _blockdef.(string)
		if !nstc._functions.Condition(state.GetString("id"), blockdef) {
			// The whole thing should not be included
			continue
		}

		blocktype, err := nstc._functions.BlockType(state.GetString("id"), blockdef)
		if err != nil {
			panic(err.Error())
		}

		switch blocktype {
		case CDL_T_INCLUSION:
			//
			// Fetch that inclusion, compile it here
			tree.Set(blockdef+"__INCLUSION", branch.Get(_blockdef, nil))
		case CDL_T_REFERENCE:
			// Reference, compile it here
			tree.Set(blockdef+"__REFERENCE", branch.Get(_blockdef, nil))
		default:
			tree.Set(_blockdef, branch.Get(_blockdef, nil))
		}
	}
	return tree
}

// Compile the tree.
func (nstc *NstCompiler) compile() error {
	rootstate, found := nstc._states[nstc.rootStateId]
	if !found {
		panic(fmt.Errorf("Root state as '%s' was not found", nstc.rootStateId))
	}
	nstc.tree = NewOTree()

	// Header
	for _, id := range []string{"id", "description"} {
		nstc.tree.Set(id, rootstate.GetString(id))
	}

	nstc.tree.Set("state", nstc.compileState(rootstate))

	return nil
}
