package nanocms_callers

const (
	BINARY = iota + 1
	SCRIPT
)

type AnsibleModule struct {
	stateRoots []string
	name       string
	args       map[string]interface{}
	modType    int
}
