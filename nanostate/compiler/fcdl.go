package nanocms_compiler

import (
	"fmt"
	"strings"
)

const (
	CDL_T_INCLUSION = iota
	CDL_T_REFERENCE
)

type CDLFunc struct {
	threads map[string]*StarlarkProcess
}

func NewCDLFunc() *CDLFunc {
	cdl := new(CDLFunc)
	cdl.threads = make(map[string]*StarlarkProcess)
	return cdl
}

// ImportSource of Starlark script and evaluate it into a running thread.
// StarlarkProcess has extra-check for the source contains only functions.
func (cdl *CDLFunc) ImportSource(id string, srcpath string) {
	sp := NewStarlarkProcess()
	err := sp.LoadFile(srcpath)
	if err != nil {
		panic(fmt.Errorf("Unable to import '%s' for id %s: %s", srcpath, id, err.Error()))
	}
	cdl.threads[id] = sp
}

// Parse incoming line and extract conditions
func (cdl *CDLFunc) getConditionsFromLine(line string) []string {
	conditions := make([]string, 0)
	for _, token := range strings.Split(line, " ") {
		if strings.HasPrefix(token, "?") {
			conditions = append(conditions, token[1:])
		}
	}
	return conditions
}

// Condition evaluates all possible conditions.
//
// Although Wiz CDL supports multiple conditions in one line,
// yet they are not encouraged and are evaluated with "OR" statement (any).
//
// Example:
//
//   something ?one ?two
//
// If "one" or "two" results to "true", then "something" will happen.
// For complex conditions they needs to be combined in Skylark function and
// expressed in Wiz CDL as a single condition.
//
// Example:
//
//   something ?onetwo
//
// def onetwo():
//     return one() and two()
//
func (cdl *CDLFunc) Condition(stateid string, line string) bool {
	conditions := cdl.getConditionsFromLine(line)
	for _, fn := range conditions {
		res, err := cdl.threads[stateid].Call(fn, nil, nil)
		if err != nil {
			panic(fmt.Errorf("Error calling state '%s': %s", stateid, err.Error()))
		}
		if res.Truth() {
			return true
		}
	}
	return len(conditions) == 0
}

// BlockType returns a type of a block: reference or inclusion
func (cdl *CDLFunc) BlockType(stateid string, line string) (int, error) {
	var err error
	if strings.Contains(line, "~") && strings.Contains(line, "&") {
		return -1, fmt.Errorf("Line '%s' in '%s' cannot be both inclusion and reference", stateid, line)
	}
	if strings.Contains(line, "~") {
		return CDL_T_INCLUSION, err
	} else if strings.Contains(line, "&") {
		return CDL_T_REFERENCE, err
	}

	return -1, err
}
