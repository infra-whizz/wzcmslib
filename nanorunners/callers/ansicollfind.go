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

type OSArch struct {
	definition string
}

var ARM32 *OSArch = &OSArch{definition: "arm"}
var ARM64 *OSArch = &OSArch{definition: "arm64"}
var INTEL64 *OSArch = &OSArch{definition: "x86_64"}

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
	plp         string
	isbinary    bool

	wzlib_logger.WzLogger
}

// NewAnsibleCollectionResolver returns an instance of the resolver
// For performance reasons, it does not scans everything,
// but only returns a precise location of precice binary or plugin
func NewAnsibleCollectionResolver(paths ...string) *AnsibleCollectionResolver {
	acr := new(AnsibleCollectionResolver)
	acr.collections = make([]AnsibleCollection, 0)
	acr.collPaths = paths
	acr.isbinary = false

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

	var err error
	acr.plp, err = nanoutils.NewPythonEnvironment().GetPureLibPath()
	if err != nil {
		acr.GetLogger().Error(err)
	}

	return acr
}

// SetOsName (equivalent to GOOS variable)
func (acr *AnsibleCollectionResolver) SetOsName(osname string) *AnsibleCollectionResolver {
	acr.osname = strings.ToLower(osname)
	return acr
}

// SetArch (equivalent to GOARCH variable)
func (acr *AnsibleCollectionResolver) SetArch(arch *OSArch) *AnsibleCollectionResolver {
	acr.arch = arch.definition
	return acr
}

// IsBinary module or not
func (arc AnsibleCollectionResolver) IsBinary() bool {
	return arc.isbinary
}

// ResolveModuleByURI of the Ansible: collection or core plugin.
func (acr *AnsibleCollectionResolver) ResolveModuleByURI(module string) (string, error) {
	moduleNamespace := strings.Split(module, ".")

	if len(moduleNamespace) > 4 || len(moduleNamespace) < 3 {
		return "", fmt.Errorf("Unknown module URI. Should be as 'ansible.namespace.plugin' for core plugin and 'ansible.collection.namespace.plugin' for collection plugin")
	}
	if moduleNamespace[0] != "ansible" {
		return "", fmt.Errorf("Unknown module URI: should start from 'ansible'")
	}

	// Todo: iterate over collPaths. Include acr.plp in the collPaths by default as first iteration
	if len(moduleNamespace) == 3 {
		return acr.resolveCorePlugin(moduleNamespace[1], moduleNamespace[2])
	} else {
		return acr.resolveCollectionPlugin(moduleNamespace[1], moduleNamespace[2], moduleNamespace[3])
	}
}

// ResolveCorePlugin of the Ansible, that is not a part of any collection
// but is shipped together with the Ansible distribution. Core modules are only in Python.
// URL: "ansible.namespace.plugin", e.g.: 'ansible.system.ping'
func (acr *AnsibleCollectionResolver) resolveCorePlugin(namespace, plugin string) (string, error) {
	pyModPath := path.Join(acr.plp, "ansible", "modules", namespace, fmt.Sprintf("%s.py", plugin))
	if _, err := os.Stat(pyModPath); err == nil {
		return pyModPath, nil
	} else {
		return "", err
	}
}

// ResolveCollectionPlugin returns a plugin by the given paths that is formatted as a collection.
// If no paths given, they are resolved to the current Python
// installation where "ansible_collection" is located.
// If plugin is binary (i.e. "library" directory is present and pattern matches there) then the matched binary returned directly.
// URI: "ansible.collection.namespace.plugin", e.g.: 'ansible.whizz.embedded.zypper'.
func (acr *AnsibleCollectionResolver) resolveCollectionPlugin(collection, namespace, plugin string) (string, error) {
	pluginRoot := path.Join(acr.plp, "ansible_collections", collection, namespace)
	binModPath := path.Join(pluginRoot, "library", fmt.Sprintf("%s-%s-%s", plugin, acr.osname, acr.arch))
	binModWrapper := path.Join(pluginRoot, "plugins", "action", fmt.Sprintf("%s.py", plugin))
	pyModPath := path.Join(pluginRoot, "plugins", "modules", fmt.Sprintf("%s.py", plugin))

	if _, err := os.Stat(binModPath); err == nil {
		if _, err := os.Stat(binModWrapper); err == nil {
			// is binary and compliant
			acr.isbinary = true
			return binModPath, nil
		} else if os.IsNotExist(err) {
			// is not compliant
			return "", fmt.Errorf("Module %s.%s.%s seems binary, but the collection is not compliant.", collection, namespace, plugin)
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
