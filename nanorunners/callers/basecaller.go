package nanocms_callers

type AnsibleModule struct {
	stateRoots []string
	name       string
	args       map[string]interface{}
}
