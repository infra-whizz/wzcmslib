package nanocms_runners

import (
	"strings"

	nanocms_state "github.com/infra-whizz/wzcmslib/nanostate"
	wzlib_logger "github.com/infra-whizz/wzlib/logger"
)

type IBaseRunner interface {
	callShell(args interface{}) ([]RunnerHostResult, error)
	callAnsibleModule(name string, kwargs map[string]interface{}) ([]RunnerHostResult, error)
	setStateRoots(roots ...string)
}

type BaseRunner struct {
	ref       IBaseRunner
	_response *RunnerResponse
	_errcode  int

	pyexe      string // Python shebang for Ansible modules
	chrootPath string
	wzlib_logger.WzLogger
}

// AddStateRoots of the collections
func (br *BaseRunner) AddStateRoots(roots ...string) *BaseRunner {
	br.ref.setStateRoots(roots...)
	return br
}

// Run the compiled and loaded nanostate
func (br *BaseRunner) Run(state *nanocms_state.Nanostate) bool {
	errors := 0
	br._response.Id = state.Id
	br._response.Description = state.Descr
	groups := make([]RunnerResponseGroup, 0)

	for _, group := range state.OrderedGroups() { // At this point groups are anyway already ordered at .Groups
		resp := &RunnerResponseGroup{
			GroupId: group.Id,
			Errcode: -1,
		}
		br.GetLogger().Debugf("Processing group '%s'", group.Id)
		response, err := br.runGroup(group.Group)
		if err != nil {
			resp.Errmsg = err.Error()
			resp.Errcode = ERR_FAILED
			errors++
		} else {
			resp.Response = response
		}
		groups = append(groups, *resp)
	}
	br._response.Groups = groups

	switch errors {
	case 0:
		br._errcode = ERR_OK
	default:
		br._errcode = ERR_FAILED
	}

	br.GetLogger().Debugf("Cycle is finished")
	return errors == 0
}

func (br *BaseRunner) setGroupResponse(cycle *RunnerResponseModule, response []RunnerHostResult, err error) {
	if err != nil {
		cycle.Errcode = ERR_FAILED
		cycle.Errmsg = err.Error()
	} else {
		cycle.Errcode = ERR_OK
		cycle.Response = response
	}
}

// Run group of modules
func (br *BaseRunner) runGroup(group []*nanocms_state.StateModule) ([]RunnerResponseModule, error) {
	resp := make([]RunnerResponseModule, 0)
	for _, smod := range group {
		cycle := &RunnerResponseModule{
			Module: smod.Module,
		}
		if cycle.Module == "shell" {
			response, err := br.ref.callShell(smod.Instructions)
			br.setGroupResponse(cycle, response, err)
			resp = append(resp, *cycle)
		} else if strings.HasPrefix(cycle.Module, "ansible.") {
			response, err := br.ref.callAnsibleModule(cycle.Module, smod.Args)
			br.setGroupResponse(cycle, response, err)
			resp = append(resp, *cycle)
		} else {
			br.GetLogger().Errorf("Module %s is not supported", cycle.Module)
		}
	}
	return resp, nil
}

// Calls shell commands (both remotely or locally)
func (br *BaseRunner) callShell(args interface{}) ([]RunnerHostResult, error) {
	panic("Abstract method call")
}

// Runs Ansible module (both remotely or locally)
func (br *BaseRunner) callAnsibleModule(name string, kwargs map[string]interface{}) ([]RunnerHostResult, error) {
	panic("Abstract method call")
}

// Response returns a map of string/any structure for further processing
func (br *BaseRunner) Response() *RunnerResponse {
	return br._response
}

// Errcode returns an error code of the runner
func (br *BaseRunner) Errcode() int {
	return br._errcode
}
