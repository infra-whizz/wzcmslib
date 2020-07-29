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
	"path"
	"path/filepath"
	"strings"

	"github.com/karrick/godirwalk"

	wzlib_traits "github.com/infra-whizz/wzlib/traits"
	wzlib_traits_attributes "github.com/infra-whizz/wzlib/traits/attributes"
	wzlib_utils "github.com/infra-whizz/wzlib/utils"
)

func NewAnsibleLocalModuleCaller(modulename string) *AnsibleModule {
	am := new(AnsibleModule)
	am.modType = 0
	am.stateRoots = make([]string, 0)
	am.name = strings.ToLower(strings.TrimPrefix(modulename, "ansible."))
	am.args = map[string]interface{}{}
	am.pyexe = []string{"/usr/bin/python3"}

	return am
}

// SetStateRoots where module is going to be found.
// NOTE: if the same module is located in a various roots, then first win
func (am *AnsibleModule) SetStateRoots(roots ...string) *AnsibleModule {
	am.stateRoots = append(am.stateRoots, roots...)
	return am
}

// SetPyInterpreter path, such as "/usr/bin/python3" or "/usr/bin/env python" etc
func (am *AnsibleModule) SetPyInterpreter(pyexe string) *AnsibleModule {
	if pyexe == "" {
		// Skip, if empty shebang
		return am
	}

	am.pyexe = []string{}
	for _, cmd := range strings.Split(pyexe, " ") {
		if cmd != "" {
			am.pyexe = append(am.pyexe, cmd)
		}
	}
	return am
}

func (am *AnsibleModule) resolvePlatformPath() string {
	traits := wzlib_traits.NewWzTraitsContainer()
	wzlib_traits_attributes.NewSysInfo().Load(traits)

	sysName := traits.Get("os.sysname")
	if sysName == nil {
		return fmt.Sprintf("generic/%s", traits.Get("arch"))
	}

	return fmt.Sprintf("%s/%s", sysName, traits.Get("arch"))
}

// Resolve module path in 2.10+ collections style
func (am *AnsibleModule) resolveModulePath() (string, error) {
	modPath := ""
	platformPath := am.resolvePlatformPath()
	for _, stateRoot := range am.stateRoots {
		moduleRoot := filepath.Clean(path.Join(stateRoot, "modules"))
		suffBinPath := filepath.Clean(path.Join(moduleRoot, "bin", platformPath, strings.ReplaceAll(am.name, ".", "/")))
		suffPyPath := filepath.Clean(path.Join(moduleRoot, strings.ReplaceAll(am.name, ".", "/")+".py"))

		if err := godirwalk.Walk(moduleRoot, &godirwalk.Options{
			Unsorted:            true,
			FollowSymbolicLinks: true,
			Callback: func(pth string, info *godirwalk.Dirent) error {
				contentType, _ := wzlib_utils.FileContentTypeByPath(pth)
				switch contentType {
				case "application/octet-stream":
					if strings.HasSuffix(pth, suffBinPath) {
						modPath = pth
						am.modType = BINARY
						return fmt.Errorf("Binary module found")
					}
				case "text/plain":
					if strings.HasSuffix(pth, suffPyPath) {
						modPath = pth
						am.modType = SCRIPT
						return fmt.Errorf("Python module found")
					}
				}
				return nil
			},
			ErrorCallback: func(pth string, err error) godirwalk.ErrorAction { return godirwalk.SkipNode },
		}); err != nil {
			break
		}
	}

	return modPath, nil
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
	stdout, stderr, err := am.execModule()
	if stderr != "" {
		am.GetLogger().Errorf("Call error:\n%s", stderr)
	}
	if err != nil && stdout == "" && stderr == "" {
		return nil, err
	}

	err = json.Unmarshal([]byte(stdout), &ret)
	if err != nil {
		return nil, err
	}

	return ret, nil
}

func (am *AnsibleModule) execModule() (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	// Resolve module and set its type. Based on this, config file is made then
	exePath, err := am.resolveModulePath()
	if err != nil {
		return "", "", err
	}

	// Create configuration JSON file
	cfg, err := am.makeConfigFile()
	if err != nil {
		return "", "", err
	}

	defer os.Remove(cfg.Name())

	var sh *exec.Cmd
	if am.modType == BINARY {
		sh = exec.Command(exePath, cfg.Name())
	} else if am.modType == SCRIPT {
		cmd := append(am.pyexe, exePath, cfg.Name())
		sh = exec.Command(cmd[0], cmd[1:]...)
	} else {
		return "", "", fmt.Errorf("Module %s was not found", am.name)
	}
	sh.Stdout = &stdout
	sh.Stderr = &stderr

	err = sh.Run()

	if err != nil {
		am.GetLogger().Errorf("Module '%s' failed: %s", exePath, err.Error())
		am.GetLogger().Debugf("STDOUT:\n%s", stdout.String())
		am.GetLogger().Debugf("STDERR:\n%s", stderr.String())
	}

	return stdout.String(), stderr.String(), err
}

// Create a temporary config file and return a path to it.
func (am *AnsibleModule) makeConfigFile() (*os.File, error) {
	f, err := ioutil.TempFile("/tmp", "nst-ansible-")
	if err != nil {
		return nil, err
	}

	var data []byte
	if am.modType == BINARY {
		data, err = json.Marshal(am.args)
	} else if am.modType == SCRIPT {
		data, err = json.Marshal(map[string]interface{}{"ANSIBLE_MODULE_ARGS": am.args})
	} else {
		panic("An attempt to call an unresolved module type (binary or Ansible-native)")
	}
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
