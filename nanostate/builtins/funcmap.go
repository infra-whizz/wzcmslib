package nanocms_builtins

import (
	"go.starlark.net/starlark"
)

/*
Skylark builtins
*/

var BuiltinMap starlark.StringDict

func init() {
	// An example what should be coming out of traits generator
	traits := starlark.NewDict(3)
	traits.SetKey(starlark.String("kernel"), starlark.String("Linux"))
	traits.SetKey(starlark.String("kernelrelease"), starlark.String("4.4.0-109-generic"))
	traits.SetKey(starlark.String("kernelversion"), starlark.String("#132-Ubuntu SMP Tue Jan 9 19:52:39 UTC 2018"))

	BuiltinMap = starlark.StringDict{
		"traits": traits,
		"uptime": starlark.NewBuiltin("uptime", Stk_Uptime),
	}
}
