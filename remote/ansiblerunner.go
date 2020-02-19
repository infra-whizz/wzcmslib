package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

type AnsibleModule struct {
	Interpreter string
	Path        string
	Binary      bool
	Argv        map[string]interface{}
}

type AnsibleModRunner struct {
}

func NewAnsibleModRunner() *AnsibleModRunner {
	amr := new(AnsibleModRunner)
	return amr
}

func (amr *AnsibleModRunner) callShell(stdin []byte, command string, arg ...string) (string, string, error) {
	var outb bytes.Buffer
	var errb bytes.Buffer
	var err error
	cmd := exec.Command(command, arg...)
	cmd.Stdout = &outb
	cmd.Stderr = &errb
	if len(stdin) > 0 {
		pipe, _ := cmd.StdinPipe()
		err = cmd.Start()
		if err != nil {
			panic(err)
		}
		io.WriteString(pipe, string(stdin)+"\n")
		pipe.Close()
		cmd.Wait()

	} else {
		cmd.Run()
	}
	return strings.TrimSpace(outb.String()), strings.TrimSpace(errb.String()), err
}

// FindPython Module finds python module
func (amr *AnsibleModRunner) FindPythonModule(namespace string) *AnsibleModule {
	var root string
	mod := &AnsibleModule{Binary: false}
	for _, itp := range []string{"python", "python3"} {
		sto, _, _ := amr.callShell([]byte{}, itp, "-c", "import ansible;print(ansible.__file__)")
		if sto != "" && strings.Contains(sto, "/") {
			mod.Interpreter = itp
			mod.Binary = false
			root = path.Dir(sto)
			break
		}
	}
	err := filepath.Walk(root, func(p string, i os.FileInfo, err error) error {
		if mod.Path != "" {
			return nil
		}
		if err != nil {
			return err
		}
		if p == (path.Join(root, "modules", strings.Join(strings.Split(namespace, "."), "/")) + ".py") {
			mod.Path = p
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}
	return mod
}

// CallAnsibleModule calls Ansible Python module locally
func (amr *AnsibleModRunner) CallAnsibleModule(mod *AnsibleModule) (string, error) {
	params := map[string]interface{}{
		"ANSIBLE_MODULE_ARGS": mod.Argv,
	}
	data, err := json.Marshal(params)
	if err != nil {
		return "", err
	}
	out, _, err := amr.callShell(data, mod.Interpreter, mod.Path)
	return out, err
}

func main() {
	if len(os.Args) < 2 {
		panic("Arguments?")
	}
	modname := os.Args[1]
	amr := NewAnsibleModRunner()
	m := amr.FindPythonModule(modname)

	m.Argv = make(map[string]interface{})
	for _, arg := range os.Args[2:] {
		if strings.Contains(arg, "=") {
			_arg := strings.SplitN(arg, "=", 2)
			m.Argv[_arg[0]] = strings.Split(_arg[1], " ") // XXX: double spaces
		} else {
			fmt.Println("Wrong argument: ", arg)
		}
	}
	fmt.Println(amr.CallAnsibleModule(m))
}
