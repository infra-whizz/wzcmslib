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
	"github.com/thoas/go-funk"

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
	am.chroot = "/"

	return am
}

// SetChroot sets another root where an Ansible module should be ran.
// Chrooted Ansible module is called via PCE (Python Chroot Executor) wrapper,
// which is embedded inside the binary of this wzd via wzcmslib.
// See wzcmslib/nanorunners/wrappers/pce.py for more details.
//
// Default "/". In this case PCE is not used.
func (am *AnsibleModule) SetChroot(root string) *AnsibleModule {
	am.chroot = root
	return am
}

// SetStateRoots where module is going to be found.
// NOTE: if the same module is located in a various roots, then first win
func (am *AnsibleModule) SetStateRoots(roots ...string) *AnsibleModule {
	am.stateRoots = append(am.stateRoots, roots...)
	return am
}

// Prepare PCE
func (am *AnsibleModule) preparePCE() error {
	if am.chroot == "/" {
		return nil
	}

	var err error
	am.pce, err = ioutil.TempFile("/tmp", ".waka-pce-")

	if err != nil {
		return err
	}

	return ioutil.WriteFile(am.pce.Name(), NewWzPyPce().Get("pce"), 0644)
}

func (am *AnsibleModule) removePCE() error {
	if am.pce == nil {
		return nil
	}

	if err := os.Remove(am.pce.Name()); err != nil {
		return err
	}
	am.pce.Close()
	am.pce = nil
	return nil
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

	modPath = wzlib_utils.RemovePrefix(modPath, am.chroot)
	am.GetLogger().Debugf("Ansible module path: %s", modPath)

	return modPath, nil
}

// SetArgs sets the key/value arguments
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

// Execute module.
// There are certain rules how that works:
//   - if everything is chrooted, everything runs chrooted.
//   - but if "root" is specified, module is NOT ran as chrooted, but parameter just passed to the module
//   - If everything is chrooted and "root" specified, its value is overwritten with the current chroot
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
	var chrootExit func() error

	if am.modType == BINARY {
		// should we run module chrooted?
		var cnt bool
		if am.chroot != "/" {
			cnt = !funk.Contains(am.args, "root")
		} else {
			cnt = false
		}

		if cnt {
			// Run chrooted
			conf := new(wzlib_utils.WzContainerParam)
			conf.Root = am.chroot
			conf.Command = exePath
			conf.Args = []string{wzlib_utils.RemovePrefix(cfg.Name(), am.chroot)}

			return wzlib_utils.NewWzContainer(conf).Run()
		} else {
			// Run directly
			sh := exec.Command(path.Join(am.chroot, exePath), cfg.Name())

			// TODO: Move to a function (this is a code repeat!)
			sh.Stdout = &stdout
			sh.Stderr = &stderr

			err = sh.Run()

			if err != nil {
				am.GetLogger().Errorf("Module '%s' failed: %s", exePath, err.Error())
				am.GetLogger().Debugf("STDOUT:\n%s", stdout.String())
				am.GetLogger().Debugf("STDERR:\n%s", stderr.String())
			}
			am.GetLogger().Debugf("STDOUT:\n%s", stdout.String())
			am.GetLogger().Debugf("STDERR:\n%s", stderr.String())

			if chrootExit != nil {
				err = chrootExit()
			}

			return stdout.String(), stderr.String(), err
		}

		// TODO: Split to two methods!
	} else if am.modType == SCRIPT {
		// Python module
		if err := am.preparePCE(); err != nil {
			return "", "", err
		}

		defer am.removePCE()

		var cmd []string
		if am.pce != nil {
			cmd = append(am.pyexe, am.pce.Name(), "-r", am.chroot, "-c", exePath,
				"-j", wzlib_utils.RemovePrefix(cfg.Name(), am.chroot))
		} else {
			cmd = append(am.pyexe, exePath, cfg.Name())
		}

		debugMessage := ""
		if am.chroot != "/" {
			debugMessage = fmt.Sprintf(" (inside: %s)", am.chroot)
		}
		am.GetLogger().Debugf("Calling Ansible module%s:\n'%s'", debugMessage, strings.Join(cmd, " "))
		sh := exec.Command(cmd[0], cmd[1:]...)
		sh.Stdout = &stdout
		sh.Stderr = &stderr

		err = sh.Run()

		if err != nil {
			am.GetLogger().Errorf("Module '%s' failed: %s", exePath, err.Error())
			am.GetLogger().Debugf("STDOUT:\n%s", stdout.String())
			am.GetLogger().Debugf("STDERR:\n%s", stderr.String())
		}
		am.GetLogger().Debugf("STDOUT:\n%s", stdout.String())
		am.GetLogger().Debugf("STDERR:\n%s", stderr.String())

		if chrootExit != nil {
			err = chrootExit()
		}

		return stdout.String(), stderr.String(), err
	}

	return "", "", fmt.Errorf("Module %s was not found", am.name)
}

// Create a temporary config file and return a path to it.
func (am *AnsibleModule) makeConfigFile() (*os.File, error) {
	var prefix string
	if am.chroot != "/" {
		prefix = am.chroot
		if funk.Contains(am.args, "root") {
			am.args["root"] = am.chroot // asking for root
		}
	}
	f, err := ioutil.TempFile(prefix+"/tmp", "nst-ansible-")
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
