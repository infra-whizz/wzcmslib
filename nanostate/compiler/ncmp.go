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
	_unresolved *RefList
	tree        *OTree
	rootId      string
}

func NewNstCompiler() *NstCompiler {
	nstc := new(NstCompiler)
	nstc.tree = nil
	nstc._states = make(map[string]*OTree)
	nstc._unresolved = NewRefList()

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
				return nstc.LoadBytes(data)
			}
		}
	}

	return err
}

// LoadString loads a nanostate from a text YAML source)
func (nstc *NstCompiler) LoadString(src string) error {
	return nstc.LoadBytes([]byte(src))
}

// LoadString loads a nanostate from an array of bytes of a YAML source
func (nstc *NstCompiler) LoadBytes(src []byte) error {
	var data yaml.MapSlice
	var err error
	if err := yaml.Unmarshal(src, &data); err != nil {
		panic(err)
	}
	state := NewOTree().LoadMapSlice(data)

	if state.Exists("id") {
		if nstc.rootId == "" {
			nstc.rootId = state.GetString("id")
		}
		nstc._states[state.GetString("id")] = state
	} else {
		err = fmt.Errorf("%s", "State has no ID")
	}

	nstc._unresolved.FindRefs(state)

	return err
}

// Cycle compiles current state and returns a next state Id to be found and loaded, if any.
// If returns an empty string, then no more cycles are found and Tree is ready.
func (nstc *NstCompiler) Cycle() string {
	// Resolve includes
	for _, id := range nstc._unresolved.GetIncluded() {
		return nstc._unresolved.MarkVisited(id)
	}

	// Everything seems resolved, compile now
	if err := nstc.compile(); err != nil {
		panic(err)
	}

	return ""
}

func (nstc *NstCompiler) Dump() {
	spew.Dump(nstc.tree)
}

// Tree returns completed tree
func (nstc *NstCompiler) Tree() *OTree {
	if len(nstc._unresolved.GetIncluded()) > 0 {
		panic("Calling for compiled tree when unresolved sources are still pending")
	}

	spew.Dump(nstc._unresolved)

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

// Compiile the tree.
func (nstc *NstCompiler) compile() error {
	return nil
}
