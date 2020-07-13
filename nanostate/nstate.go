/*
NanoNanostate implementation.
Nanostate example:

----------------------------------------------
id: some-Nanostate
descr: This describes what this Nanostate is for.

state:
  maintain-database:
	- sshrunner:
	  - stop-db: systemctl stop postgresql.service
	  - backup-db: pg-backup /var/lib/pgsql/data /opt/backups/
	  - start-db: systemctl start postgresql.service
	- somemodule:
	  keyparam: valueparam
	  keyparam2: valueparam2
	  keyparam3: valueparam3

  some-other-group:
	- sshrunner:
	  - uptime: uptime
	  - id : cat /etc/machine-id
----------------------------------------------

In the example above, three fields are required:

- id
  This an ID of the Nanostate. It is used for the
  reporting at the end.

- descr:
  Description of the Nanostate. Also reporting.

- state:
  This is the entire tree of the Nanostate structure.
  It has twofold tree: a group IDs with a list of
  modules and the params or commands below.

Currently only one module is implemented: sshrunner,
which performs a series of synchronous commands in
the order they were placed for orchestration purposes.

In the nanoNanostate groups are asynchronous, but the
commands inside the groups are synchronous.
*/
package nanocms_state

import (
	"errors"
	"reflect"

	nanocms_compiler "github.com/infra-whizz/wzcmslib/nanostate/compiler"
)

type StateModule struct {
	Module       string
	Instructions []interface{}          // For modules that might be called multiple times. Usually a shell command.
	Args         map[string]interface{} // Modules, that are called only once.
}

type StateGroup struct {
	Id    string
	Group []*StateModule
}

type Nanostate struct {
	Id         string
	Descr      string
	Groups     []*StateGroup
	GroupIndex []string
}

func NewNanostate() *Nanostate {
	pb := new(Nanostate)
	return pb
}

func (pb *Nanostate) OrderedGroups() []*StateGroup {
	orderedGroups := make([]*StateGroup, 0)
	for _, groupId := range pb.GroupIndex {
		for _, group := range pb.Groups {
			if group.Id == groupId {
				orderedGroups = append(orderedGroups, group)
			}
		}
	}
	return orderedGroups
}

// Load Nanostate tree, which is already compiled statically and vaildated.
func (pb *Nanostate) Load(tree *nanocms_compiler.OTree) error {
	pb.Groups = make([]*StateGroup, 0)
	pb.GroupIndex = make([]string, 0)
	sTree := tree.Serialise()

	for _, groupId := range tree.GetBranch("state").Keys() {
		pb.GroupIndex = append(pb.GroupIndex, groupId.(string))
	}

	for _, rootKey := range []string{"id", "description", "state"} {
		if val, ex := sTree[rootKey]; ex {
			switch rootKey {
			case "id":
				pb.Id = val.(string)
			case "description":
				pb.Descr = val.(string)
			case "state":
				pb.loadState(val)
			}
		}
	}

	pb.Groups = pb.OrderedGroups()

	var err error
	if pb.Id == "" || pb.Descr == "" || pb.Groups == nil {
		err = errors.New("Broken state: id or description or state itself is missing")
	}
	return err
}

// Load the state, splitting groups and modules
func (pb *Nanostate) loadState(state interface{}) {
	// Load groups
	for gname, gobj := range state.(map[string]interface{}) {
		pb.Groups = append(pb.Groups, pb.loadGroup(gname, gobj))
	}
}

// Load a group
func (pb *Nanostate) loadGroup(name string, gobj interface{}) *StateGroup {
	group := &StateGroup{
		Id:    name,
		Group: make([]*StateModule, 0),
	}

	for _, mobj := range gobj.([]interface{}) {
		instr := pb.loadModuleInstructions(mobj)
		if instr != nil {
			group.Group = append(group.Group, instr)
		}
	}
	return group
}

// Load an arbitrary module instructions (parameters)
func (pb *Nanostate) loadModuleInstructions(mobj interface{}) *StateModule {
	// Note: always length of 1
	var module *StateModule
	for mname, minstr := range mobj.(map[string]interface{}) {
		module = &StateModule{
			Instructions: make([]interface{}, 0),
		}
		module.Module = mname
		tMinstr := reflect.ValueOf(minstr).Kind()
		if tMinstr == reflect.Slice {
			module.Instructions = append(module.Instructions, minstr.([]interface{})...)
		} else if tMinstr == reflect.Map {
			module.Args = make(map[string]interface{})
			for argname, argval := range minstr.(map[string]interface{}) {
				module.Args[argname] = argval
			}
		}
	}
	return module
}
