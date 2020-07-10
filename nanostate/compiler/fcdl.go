package nanocms_compiler

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

const (
	CDL_T_INCLUSION = iota
	CDL_T_DEPENDENCY
	CDL_T_LOOP
)

type CDLLoop struct {
	StateId string
	Module  string
	Params  []map[interface{}]interface{}
}

type CDLInclusion struct {
	Stateid string
	Blocks  []string
}

type CDLDependency struct {
	Stateid     string
	AnchorBlock string
	Blocks      []string
}

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
		state, ex := cdl.threads[stateid]
		if !ex {
			panic(fmt.Errorf("State '%s.st' does not have assotiated Python "+
				"file '%s.fn' where should be a function '%s()'. To resolve this, "+
				"create a file '%s.fn' in the same directory where the state is, "+
				"and define that function there.", stateid, stateid, fn, stateid))
		}
		res, err := state.Call(fn, nil, nil)
		if err != nil {
			panic(fmt.Errorf("Error calling state '%s': %s", stateid, err.Error()))
		}
		if res.Truth() {
			return true
		}
	}
	return len(conditions) == 0
}

/*
	Loop returns an array of key/value maps as keyword arguments and applies
	given function or module on each.

	Example:

		my-job:
			- my_module []my_function

	The "my_function" is expected to return an array with keywords map inside, e.g.:

		def my_function():
			return [
				{
					"name": "value",
					"other": "value",
				},
				{
					"name": "something",
					"other": "something-else".
				}
			]

	In this case the example above will be compiled to the following:

		my-job:
			- my_module:
				name: value
				other: value
			- my_module:
				name: something
				other: something-else
*/
func (cdl *CDLFunc) Loop(stateid string, line string) (*CDLLoop, error) {
	line = regexp.MustCompile(`\s+`).ReplaceAllString(line, " ")
	tokens := strings.Split(line, " ")
	if len(tokens) != 2 || !strings.HasPrefix(tokens[1], "[]") {
		return nil, fmt.Errorf("Loop directive '%s' has invalid syntax at '%s'", line, stateid)
	}
	fn := tokens[1][2:]
	res, err := cdl.threads[stateid].Call(fn, nil, nil)
	if err != nil {
		panic(fmt.Errorf("Error calling state '%s': %s", stateid, err.Error()))
	}

	res_type := res.Type()
	if res_type != "list" {
		panic(fmt.Errorf("Function '%s' returns '%s', but is expected to return a list of dicts.", fn, res_type))
	}

	params := make([]map[interface{}]interface{}, 0)
	for _, pset := range NewStarType(res).Interface().([]interface{}) {
		if reflect.TypeOf(pset).Kind() != reflect.Map {
			panic(fmt.Errorf("Function '%s' is expected to return a list of dicts.", fn))
		}
		params = append(params, pset.(map[interface{}]interface{}))
	}

	return &CDLLoop{StateId: stateid, Params: params, Module: tokens[0]}, nil
}

/*
	Inclusion can have the entire state included or only specific blocks from it.
	The format is the following:

		~<STATE-ID>/[BLOCK-A]:[BLOCK-B]:...

	Blocks are delimited with colon ":" symbol. For example, to add the entire
	state:

		~my-state

	To add only one block from that state:

		~my-state/my-block

	To add few blocks from that state:

		~my-state/my-block:my-other-block

	All jobs from that block will be included.
*/
func (cdl *CDLFunc) GetInclusion(stateid string, line string) (*CDLInclusion, error) {
	// XXX: Should check if there is only one inclusion
	incl := &CDLInclusion{
		Blocks: make([]string, 0),
	}
	if strings.Contains(line, "~") {
		for _, token := range strings.Split(line, " ") {
			if token == "" {
				continue
			}
			token = strings.TrimSpace(strings.TrimSuffix(token, "/"))

			if strings.HasPrefix(token, "~") {
				inclPath := append(strings.Split(strings.ReplaceAll(token, "~", ""), "/"), "")[:2]
				incl.Stateid = inclPath[0]
				if inclPath[1] != "" {
					incl.Blocks = strings.Split(inclPath[1], ":")
				}
			}
		}
	}

	return incl, nil
}

/*
	Dependency can include specific list of blocks from a state in the order they are defined.
	It does not allow to include the entire state with all the blocks:

		<BLOCK-ID> &<STATE-ID>/<BLOCK-ID>:[BLOCK-ID]:...

	Blocks are delimited with colon ":" symbol. For example, to add only one block from some state:

		do-something &my-state/my-block

	To add few blocks from that state:

		do-something &my-state/my-block:my-other-block
*/
func (cdl *CDLFunc) GetDependency(stateid, line string) (*CDLDependency, error) {
	var err error
	expr := strings.Split(strings.ReplaceAll(regexp.MustCompile(`\s+`).ReplaceAllString(line, " "), "&", ""), " ")
	if len(expr) != 2 {
		return nil, fmt.Errorf("Dependency directive '%s' has invalid syntax at '%s'", line, stateid)
	}
	refJobs := strings.Split(expr[1], "/")
	if len(refJobs) < 2 {
		return nil, fmt.Errorf("Dependency directive '%s' has invalid syntax at '%s'", line, stateid)
	}

	dep := &CDLDependency{
		Stateid:     refJobs[0],
		AnchorBlock: expr[0],
		Blocks:      strings.Split(strings.Trim(refJobs[1], ":"), ":"),
	}

	if len(dep.Blocks) == 0 {
		return nil, fmt.Errorf("Dependency directive '%s' should not include the entire state at '%s'.", line, stateid)
	}

	return dep, err
}

// BlockType returns a type of a block: reference or inclusion
func (cdl *CDLFunc) BlockType(stateid string, line string) (int, error) {
	// XXX: The implementation is very basic and dirty. This needs to be a better updated.
	var err error
	if strings.Contains(line, "~") && strings.Contains(line, "&") {
		return -1, fmt.Errorf("Line '%s' in '%s' cannot be both inclusion and reference", stateid, line)
	}
	if strings.Contains(line, "~") {
		return CDL_T_INCLUSION, err
	} else if strings.Contains(line, "&") {
		return CDL_T_DEPENDENCY, err
	} else if strings.Contains(line, "[]") {
		if strings.Contains(line, "~") || strings.Contains(line, "&") {
			return -1, fmt.Errorf("Loop at '%s' in '%s' cannot be inclusion or a reference", stateid, line)
		}
		return CDL_T_LOOP, err
	}

	return -1, err
}

// ToCDLKey removes all controlling macros, leaving only ready to final use key.
func (cdl *CDLFunc) ToCDLKey(stateid string, line string) string {
	for _, token := range strings.Split(line, " ") {
		if len(token) > 0 && !strings.HasPrefix(token, "~") && !strings.HasPrefix(token, "&") {
			return token
		}
	}
	return ""
}
