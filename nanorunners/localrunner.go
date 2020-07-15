package nanocms_runners

import (
	"bytes"
	"os/exec"
	"strings"

	nanocms_callers "github.com/infra-whizz/wzcmslib/nanorunners/callers"
)

type LocalRunner struct {
	stateRoots []string
	BaseRunner
}

func NewLocalRunner() *LocalRunner {
	lr := new(LocalRunner)
	lr.ref = lr
	lr.stateRoots = make([]string, 0)
	lr._errcode = ERR_INIT
	lr._response = &RunnerResponse{}
	return lr
}

// Set state roots
func (lr *LocalRunner) setStateRoots(roots ...string) {
	lr.stateRoots = append(lr.stateRoots, roots...)
}

// Call module commands
func (lr *LocalRunner) callShell(args interface{}) ([]RunnerHostResult, error) {
	result := make([]RunnerHostResult, 0)
	for _, argset := range args.([]interface{}) {
		result = append(result, *lr.runCommand(argset))
	}
	return result, nil
}

func (lr *LocalRunner) callAnsibleModule(name string, kwargs map[string]interface{}) ([]RunnerHostResult, error) {
	lr.GetLogger().Debugf("Calling external module '%s': %v", name, kwargs)
	caller := nanocms_callers.NewAnsibleLocalModuleCaller(name).SetStateRoots(lr.stateRoots...)
	ret, err := caller.SetArgs(kwargs).Call()

	var errmsg string
	errcode := ERR_OK
	if err != nil {
		errmsg = err.Error()
		errcode = ERR_FAILED
	}

	response := map[string]RunnerStdResult{
		name: RunnerStdResult{
			Json:    ret,
			Errmsg:  errmsg,
			Errcode: errcode,
		},
	}

	rhr := &RunnerHostResult{
		Host:     "localhost",
		Response: response,
	}

	return []RunnerHostResult{*rhr}, nil
}

// Run a local command
func (br *LocalRunner) runCommand(argset interface{}) *RunnerHostResult {
	response := make(map[string]RunnerStdResult)
	result := &RunnerHostResult{
		Host:     "localhost",
		Response: response,
	}

	for icid, icmd := range argset.(map[string]interface{}) {
		cmd := icmd.(string)
		args := make([]string, 0)
		for idx, token := range strings.Split(strings.TrimSpace(cmd), " ") {
			if idx == 0 {
				cmd = token
			} else {
				if token != "" {
					args = append(args, token)
				}
			}
		}
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		br.GetLogger().Debugf("Running command '%s' args: '%v'", cmd, args)

		sh := exec.Command(cmd, args...)
		sh.Stdout = &stdout
		sh.Stderr = &stderr

		err := sh.Run()

		out := &RunnerStdResult{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}

		if err != nil {
			out.Errmsg = err.Error()
			out.Errcode = ERR_FAILED
		}
		response[icid] = *out
	}

	return result
}
