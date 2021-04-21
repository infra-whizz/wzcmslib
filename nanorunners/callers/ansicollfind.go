package nanocms_callers

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/infra-whizz/wzcmslib/nanoutils"
	wzlib_logger "github.com/infra-whizz/wzlib/logger"
	wzlib_traits "github.com/infra-whizz/wzlib/traits"
	wzlib_traits_attributes "github.com/infra-whizz/wzlib/traits/attributes"
)

// AnsibleCollection is a collection map for Ansible
// Binary mode supported only via proxy plugins and pre-compiled binaries
type AnsibleCollection struct {
	binary  bool // Binary = true does not mean that the modules actually are binaries
	path    string
	plugins []string
}

// Binary or not (type)
func (ac AnsibleCollection) Binary() bool {
	return ac.binary
}

// Path to the collection
func (ac AnsibleCollection) Path() string {
	return ac.path
}

// Plugins of the collection (their full path to each)
func (ac AnsibleCollection) Plugins() []string {
	return ac.plugins
}

// AnsibleCollectionResolver object
type AnsibleCollectionResolver struct {
	collections []AnsibleCollection
	collPaths   []string
	osname      string
	arch        string

	wzlib_logger.WzLogger
}

// NewAnsibleCollectionResolver returns an instance of the resolver
// For performance reasons, it does not scans everything,
// but only returns a precise location of precice binary or plugin
func NewAnsibleCollectionResolver(paths ...string) *AnsibleCollectionResolver {
	acr := new(AnsibleCollectionResolver)
	acr.collections = make([]AnsibleCollection, 0)
	acr.collPaths = paths

	traits := wzlib_traits.NewWzTraitsContainer()
	wzlib_traits_attributes.NewSysInfo().Load(traits)

	osname := traits.Get("os.sysname")
	if osname == nil {
		acr.GetLogger().Error("Unable to obtain details about this platform")
		return acr
	}

	arch := traits.Get("arch")
	if arch == nil {
		acr.GetLogger().Error("Unable to obtain architecture of this platform")
		return acr
	}

	acr.osname = osname.(string)
	acr.arch = arch.(string)

	return acr
}

// SetOsName (equivalent to GOOS variable)
func (acr *AnsibleCollectionResolver) SetOsName(osname string) *AnsibleCollectionResolver {
	acr.osname = strings.ToLower(osname)
	return acr
}

// SetArch (equivalent to GOARCH variable)
func (acr *AnsibleCollectionResolver) SetArch(arch string) *AnsibleCollectionResolver {
	acr.arch = strings.ToLower(arch)
	return acr
}

// ResolveCorePlugin of the Ansible, that is not a part of any collection
// but is shipped together with the Ansible distribution.
func (acr *AnsibleCollectionResolver) ResolveCorePlugin(module string) (string, error) {
	return "", nil
}

// ResolveCollectionPlugin returns a plugin by the given paths that is formatted as a collection.
// If no paths given, they are resolved to the current Python
// installation where "ansible_collection" is located.
// If plugin is binary (i.e. "library" directory is present and pattern matches there) then the matched binary returned directly.
func (acr *AnsibleCollectionResolver) ResolveCollectionPlugin(module string) (string, error) {
	moduleNamespace := strings.Split(module, ".")
	if len(moduleNamespace) != 3 {
		return "", fmt.Errorf("Module is expected to be in the collection, therefore format should be specified as 'collection.namespace.module' instead")
	}

	pyenv := nanoutils.NewPythonEnvironment()
	plp, err := pyenv.GetPureLibPath()
	if err != nil {
		return "", err
	}

	pluginRoot := path.Join(plp, "ansible_collections", moduleNamespace[0], moduleNamespace[1])
	binModPath := path.Join(pluginRoot, "library", fmt.Sprintf("%s-%s-%s", moduleNamespace[2], acr.osname, acr.arch))
	binModWrapper := path.Join(pluginRoot, "plugins", "action", fmt.Sprintf("%s.py", moduleNamespace[2]))
	pyModPath := path.Join(pluginRoot, "plugins", "modules", fmt.Sprintf("%s.py", moduleNamespace[2]))

	if _, err := os.Stat(binModPath); err == nil {
		if _, err := os.Stat(binModWrapper); err == nil {
			// is binary and compliant
			return binModPath, nil
		} else if os.IsNotExist(err) {
			// is not compliant
			return "", fmt.Errorf("Module %s seems binary, but the collection is not compliant.", module)
		}
	} else if os.IsNotExist(err) {
		// Is not binary
		if _, err := os.Stat(pyModPath); err == nil {
			// is Python and compliant
			return pyModPath, nil
		} else if os.IsNotExist(err) {
			return "", err
		}
	} else {
		return "", err
	}

	return "", nil
}
