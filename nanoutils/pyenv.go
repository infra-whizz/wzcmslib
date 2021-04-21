package nanoutils

import (
	"encoding/json"

	wzlib_subprocess "github.com/infra-whizz/wzlib/subprocess"
)

type PythonEnvironment struct {
	PyExe string
}

func NewPythonEnvironment() *PythonEnvironment {
	pyenv := new(PythonEnvironment)
	pyenv.PyExe = "/usr/bin/python3"

	return pyenv
}

// SetPyExe sets other Python executable than default (/usr/bin/python3)
func (pe *PythonEnvironment) SetPyExe(pth string) *PythonEnvironment {
	pe.PyExe = pth
	return pe
}

// GetPureLibPath returns distributed site-packages root
func (pe PythonEnvironment) GetPureLibPath() (string, error) {
	plp, err := wzlib_subprocess.Output(wzlib_subprocess.ExecCommand(pe.PyExe, "-c", "import sysconfig; print(sysconfig.get_paths()[\"purelib\"])"))
	if err != nil {
		return "", err
	}

	return plp, nil
}

// GetSitePackagesPath returns python's dist-packages paths. NOTE: that won't work under virtenv!
func (pe PythonEnvironment) GetSitePackagesPath() ([]string, error) {
	spaths := []string{}
	sitepath, err := wzlib_subprocess.Output(wzlib_subprocess.ExecCommand(pe.PyExe, "-c", "import site, json; print(json.dumps(site.getsitepackages()))"))
	if err != nil {
		return spaths, err
	}
	var buff interface{}
	if err := json.Unmarshal([]byte(sitepath), &buff); err != nil {
		return spaths, err
	}

	if buff != nil {
		for _, pth := range buff.([]interface{}) {
			spaths = append(spaths, pth.(string))
		}
	}

	return spaths, nil
}
