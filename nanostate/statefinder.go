/*
Nanostate is loaded by Id or filename.
*/

package nanocms_state

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-yaml/yaml"
	"github.com/infra-whizz/wzcmslib/nanoutils"
	"github.com/sirupsen/logrus"
)

var logger *logrus.Logger

func init() {
	logger = nanoutils.GetTextLogger(logrus.DebugLevel, os.Stdout)
}

type NanoStateFunctionsMeta struct {
	Filename string
	Path     string
	Info     *os.FileInfo
}

type NanoStateMeta struct {
	Id        string
	Filename  string
	Path      string
	Info      *os.FileInfo
	Functions *NanoStateFunctionsMeta
}

type NanoStateIndex struct {
	stateRoots []string
	_id_index  map[string]int
	_fn_index  map[string]int
	_mt_index  map[int]NanoStateMeta
	_ct        int
}

func NewNanoStateIndex() *NanoStateIndex {
	nsf := new(NanoStateIndex)
	nsf.stateRoots = make([]string, 0)
	nsf._id_index = make(map[string]int)
	nsf._fn_index = make(map[string]int)
	nsf._mt_index = make(map[int]NanoStateMeta)

	return nsf
}

// AddStateRoot is used to chain-add another state root
func (nsf *NanoStateIndex) AddStateRoot(pth string) *NanoStateIndex {
	nsf.stateRoots = append(nsf.stateRoots, pth)
	return nsf
}

// GetStateRoots where states and collections are located
func (nsf *NanoStateIndex) GetStateRoots() []string {
	return nsf.stateRoots
}

// AddStateRoots is used to chain-add another state roots (array)
func (nsf *NanoStateIndex) AddStateRoots(pth ...string) *NanoStateIndex {
	for _, p := range pth {
		nsf.AddStateRoot(p)
	}
	return nsf
}

// Index all the files in the all roots
func (nsf *NanoStateIndex) Index() *NanoStateIndex {
	nsf._ct = len(nsf._mt_index)
	for _, root := range nsf.stateRoots {
		nsf.getPathFiles(root)
	}
	return nsf
}

// This only unmarshalls the state and fetches its ID
func (nsf *NanoStateIndex) getStateId(pth string) (string, error) {
	logger.Debugln("Loading state ID by path", pth)

	data, err := ioutil.ReadFile(pth)
	if err != nil {
		logger.Errorf("Error reading state file '%s': %s", pth, err.Error())
		return "", err
	}
	var state map[string]interface{}
	err = yaml.Unmarshal(data, &state)
	if err != nil {
		logger.Errorf("Error loading state '%s': %s", pth, err.Error())
		return "", err
	}
	stateId, ex := state["id"]
	if !ex {
		return "", fmt.Errorf("State %s has no id, skipping", pth)
	}
	return stateId.(string), nil
}

func (nsf *NanoStateIndex) getPathFiles(root string) {
	err := filepath.Walk(root,
		func(pth string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(info.Name(), ".st") { // Filter out only state files
				stateId, err := nsf.getStateId(pth)
				if err != nil {
					logger.Debugln("Skipping state", pth)
					return nil
				}
				// Load state
				nsm := &NanoStateMeta{
					Id:       stateId,
					Filename: path.Base(pth),
					Path:     pth,
					Info:     &info,
				}
				nsf._mt_index[nsf._ct] = *nsm
				nsf._fn_index[nsm.Filename] = nsf._ct
				nsf._id_index[nsm.Id] = nsf._ct
				nsf._ct++
			}
			return nil
		})
	if err != nil {
		panic(err)
	}
}

func (nsf *NanoStateIndex) GetStateById(id string) (*NanoStateMeta, error) {
	fp, ok := nsf._id_index[id]
	if !ok {
		return nil, fmt.Errorf("No state can be found by Id %s", id)
	}

	nstm := nsf._mt_index[fp]
	return &nstm, nil
}

func (nsf *NanoStateIndex) GetStateByFileName(name string) (*NanoStateMeta, error) {
	fp, ok := nsf._fn_index[name]
	if !ok {
		return nil, fmt.Errorf("No state corresponds to the filename %s", name)
	}

	nstm := nsf._mt_index[fp]
	return &nstm, nil
}
