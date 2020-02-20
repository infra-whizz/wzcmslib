/*
	Local caller to call any Ansible module on the current machine.
	Used by a client.
*/

package nanocms_callers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

func NewAnsibleLocalModuleCaller(modulename string) *AnsibleModule {
	am := new(AnsibleModule)
	am.name = strings.ToLower(strings.TrimPrefix(modulename, "ansible."))
	am.args = map[string]interface{}{
		"new": true,
	}

	return am
}

// SetKwargs sets the key/value arguments
func (am *AnsibleModule) SetArgs(kwargs map[string]interface{}) *AnsibleModule {
	for k, v := range kwargs {
		am.AddArg(k, v)
	}
	return am
}

// AddArg adds an argument with key/value
func (am *AnsibleModule) AddArg(key string, value interface{}) *AnsibleModule {
	am.args[key] = value
	return am
}

// Call Ansible module
func (am *AnsibleModule) Call() (map[string]interface{}, error) {
	var ret map[string]interface{}
	cfg, err := am.makeConfigFile()
	if err != nil {
		return nil, err
	} else {
		defer os.Remove(cfg.Name())
		stdout, stderr, err := am.execModule(cfg.Name())
		if stderr != "" {
			fmt.Println(stderr)
		}
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal([]byte(stdout), &ret)
		if err != nil {
			return nil, err
		}
	}

	return ret, nil
}

func (am *AnsibleModule) execModule(cfgpath string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	sh := exec.Command(am.name, cfgpath)
	sh.Stdout = &stdout
	sh.Stderr = &stderr

	err := sh.Run()
	return stdout.String(), stderr.String(), err
}

// Create a temporary config file and return a path to it.
func (am *AnsibleModule) makeConfigFile() (*os.File, error) {
	f, err := ioutil.TempFile("/tmp", "nst-ansible-")
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(am.args)
	if err != nil {
		return nil, err
	}

	_, err = f.WriteString(string(data))
	f.Close()
	if err != nil {
		os.Remove(f.Name())
	}

	return f, err
}
