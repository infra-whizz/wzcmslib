package nanocms_builtins

import (
	"go.starlark.net/starlark"
)

// Example
func Stk_Uptime(thread *starlark.Thread, b *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	loadAverage := starlark.NewDict(3)
	loadAverage.SetKey(starlark.String("min"), starlark.String("0.93"))
	loadAverage.SetKey(starlark.String("max"), starlark.String("1.07"))
	loadAverage.SetKey(starlark.String("avg"), starlark.String("0.81"))
	return loadAverage, nil
}
