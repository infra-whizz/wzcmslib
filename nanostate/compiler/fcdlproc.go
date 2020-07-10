package nanocms_compiler

import (
	"fmt"

	nanocms_builtins "github.com/infra-whizz/wzcmslib/nanostate/builtins"
	"go.starlark.net/repl"
	"go.starlark.net/resolve"
	"go.starlark.net/starlark"
)

func init() {
	resolve.AllowFloat = true
	resolve.AllowNestedDef = true
	resolve.AllowLambda = true
	resolve.AllowSet = true
	resolve.AllowRecursion = true // while statements and recursive functions
}

type StarlarkProcess struct {
	thread   *starlark.Thread
	globals  starlark.StringDict
	builtins starlark.StringDict
}

func NewStarlarkProcess() *StarlarkProcess {
	sp := new(StarlarkProcess)
	sp.builtins = nanocms_builtins.BuiltinMap
	return sp
}

func (sp *StarlarkProcess) LoadFile(src string) error {
	var err error
	sp.thread = &starlark.Thread{Load: repl.MakeLoad()}
	sp.globals, err = starlark.ExecFile(sp.thread, src, nil, sp.builtins)

	return err
}

func (sp *StarlarkProcess) Call(fn string, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	if !sp.globals.Has(fn) {
		return nil, fmt.Errorf("No such function: %s", fn)
	}
	return starlark.Call(sp.thread, sp.globals[fn], args, kwargs)
}
