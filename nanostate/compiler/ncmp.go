/*
Nanostate compiler.

Currently just a static YAML instructions loader according to the Nanostate specs.
*/

package nanocms_compiler

import (
	"errors"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-yaml/yaml"
	"io/ioutil"
	"os"
	"strings"
)

type NstCompiler struct {
	tree map[string]interface{}
}

func NewNstCompiler() *NstCompiler {
	nstc := new(NstCompiler)
	nstc.tree = make(map[string]interface{})

	return nstc
}

// LoadFile loads a nanostate from the YAML file
func (nstc *NstCompiler) LoadFile(nstpath string) error {
	var err error
	if !strings.HasSuffix(nstpath, ".nst") { // This is not a storage file from IBM's Lotus Domino :-)
		err = errors.New("State file should have suffix \".nst\"")
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
	return nstc.compile(src)
}

func (nstc *NstCompiler) Dump() {
	spew.Dump(nstc.tree)
}

// Tree returns
func (nstc *NstCompiler) Tree() map[string]interface{} {
	return nstc.tree
}

// Compiile the source. At this point, if there are several files
// with "include" statement, they already should be properly merged.
//
// Compilation here processing internal variables, functions and placeholders
// into a real values, rendering at the end a static hash tree, ready to process
// with an appropriate runner over SSH remotrely or locally (used by a client).
func (nstc *NstCompiler) compile(src []byte) error {
	return yaml.Unmarshal(src, &nstc.tree)
}
