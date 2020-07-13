package nanocms_state

import (
	nanocms_compiler "github.com/infra-whizz/wzcmslib/nanostate/compiler"
	wzlib_utils "github.com/infra-whizz/wzlib/utils"
)

/*
	StateCompiler factory, which returns ready to use Nano CMS compiler,
	providing wrapped classes for whatever kind of runners (local runner, remote etc).
*/

type StateCompiler struct {
	compiler   *nanocms_compiler.NstCompiler
	stateIndex *NanoStateIndex
	state      *Nanostate
}

func NewStateCompiler() *StateCompiler {
	cmp := new(StateCompiler)
	cmp.compiler = nanocms_compiler.NewNstCompiler()
	cmp.stateIndex = NewNanoStateIndex()
	cmp.state = NewNanostate()

	return cmp
}

// Index state roots
func (nst *StateCompiler) Index(roots ...string) *StateCompiler {
	nst.GetStateIndex().AddStateRoots(roots...).Index()
	return nst
}

// Compile state tree starting from the entry state as a resolvable path.
func (nst *StateCompiler) Compile(indexPath string) (int, error) {
	if err := nst.compiler.LoadFile(indexPath); err != nil {
		return wzlib_utils.EX_GENERIC, err
	}
	// Load the entire chain of the local caller
	for {
		nextId := nst.compiler.Cycle()
		cMeta, x := nst.stateIndex.GetStateById(nextId)
		if x != nil && nextId != "" {
			nst.compiler.SquashState(nextId) // XXX: This still is not sure if state is optional!
			continue
		}
		if cMeta != nil {
			if err := nst.compiler.LoadFile(cMeta.Path); err != nil {
				return wzlib_utils.EX_GENERIC, err
			}
		} else {
			break
		}
	}

	if err := nst.state.Load(nst.compiler.Tree()); err != nil {
		return wzlib_utils.EX_GENERIC, err
	}

	return wzlib_utils.EX_OK, nil
}

// GetStateIndex returns an instance of the state index.
func (nst *StateCompiler) GetStateIndex() *NanoStateIndex {
	return nst.stateIndex
}

// GetState returns an instance of the compiled state
func (nst *StateCompiler) GetState() *Nanostate {
	return nst.state
}
