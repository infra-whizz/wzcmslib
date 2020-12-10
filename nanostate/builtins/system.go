package nanocms_builtins

import (
	"fmt"
	"os"
	"strings"

	"go.starlark.net/starlark"
)

// Stk_OsEnviron returns operating system environment, like "os.environ" in Python
// Adding specific keys as args will search only for them (Python example):
//
//   value = os_environ("FOO").get("FOO")

func Stk_OsEnviron(thread *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	specific := make([]string, 0)
	for i := 0; i < args.Len(); i++ {
		if args.Index(i).Type() == starlark.String("").Type() {
			specific = append(specific, args.Index(i).(starlark.String).GoString())
		}
	}

	envVariables := os.Environ()
	environ := starlark.NewDict(len(envVariables))

	var key interface{}
	var value interface{}
	for _, keypair := range envVariables {
		kval := strings.SplitN(keypair, "=", 2)
		if len(kval) == 2 {
			if len(specific) > 0 {
				for _, skey := range specific {
					if skey == kval[0] {
						key, value = kval[0], kval[1]
					}
				}
			} else {
				key, value = kval[0], kval[1]
			}
			if key != nil {
				environ.SetKey(starlark.String(key.(string)), starlark.String(value.(string)))
				key, value = nil, nil
			}
		}
	}

	return environ, nil
}

// Stk_OsEnvironKey searches for a specific key and returns a string value or None
// Usage:
//
//   value = os_get_environ("FOO")
//
func Stk_OsEnvironKey(thread *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	switch len := args.Len(); {
	case len > 1:
		return starlark.None, fmt.Errorf("Function 'os_get_environ' expects only one argument")
	case len == 1:
		env, err := Stk_OsEnviron(thread, builtin, args, kwargs)
		if err != nil {
			return starlark.None, err
		}
		value, ex, err := env.(*starlark.Dict).Get(args.Index(0))
		if !ex {
			return starlark.None, err
		}
		return value, err
	default:
		return starlark.None, nil
	}
}

// Stk_OsEnvironRoot appends
func Stk_OsEnvironRoot(thread *starlark.Thread, builtin *starlark.Builtin, args starlark.Tuple, kwargs []starlark.Tuple) (starlark.Value, error) {
	//args.Index(0)
	envArgs := starlark.Tuple
	val, err := Stk_OsEnvironKey(thread, builtin, args, kwargs)
	return starlark.None, nil
}
