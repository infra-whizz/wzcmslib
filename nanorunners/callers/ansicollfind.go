package nanocms_callers

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
}

// NewAnsibleCollectionResolver returns an instance of the resolver
// For performance reasons, it does not scans everything,
// but only returns a precise location of precice binary or plugin
func NewAnsibleCollectionResolver(paths ...string) *AnsibleCollectionResolver {
	acr := new(AnsibleCollectionResolver)
	acr.collections = make([]AnsibleCollection, 0)
	acr.collPaths = paths

	// XXX: Resolve current os/arch instead of hard-coding this
	acr.osname = "linux"
	acr.arch = "x86_64"
	return acr
}

// SetOsName (equivalent to GOOS variable)
func (acr *AnsibleCollectionResolver) SetOsName(osname string) *AnsibleCollectionResolver {
	acr.osname = osname
	return acr
}

// SetArch (equivalent to GOARCH variable)
func (acr *AnsibleCollectionResolver) SetArch(arch string) *AnsibleCollectionResolver {
	acr.arch = arch
	return acr
}

// Resolve a plugin by the given paths.
// If no paths given, they are resolved to the current Python
// installation where "ansible_collection" is located.
func (acr *AnsibleCollectionResolver) ResolvePlugin(module string) string {
	// Idea of resolving to do it fast:
	// - Do not preload everything every time
	// - Resolve only the precise module in searchable paths
	// - Module name e.g. "whizz.embedded.zypper" which resolves to a standard collection,
	//   so the resolver needs to look exactly in:
	//   - $PYTHON_SITE_PATH/ansible_collections/whizz/embedded/plugins/action/zypper.py
	//   - $PYTHON_SITE_PATH/ansible_collections/whizz/embedded/plugins/library/zypper_<target os>_<target arch>
	// - Pick the binary with already known target OS and arch
	// - Execute as further

	return ""
}
