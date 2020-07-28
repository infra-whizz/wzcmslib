package nanocms_callers

import (
	wzlib_logger "github.com/infra-whizz/wzlib/logger"
)

const (
	BINARY = iota + 1
	SCRIPT
)

type AnsibleModule struct {
	stateRoots []string
	name       string
	args       map[string]interface{}
	modType    int
	wzlib_logger.WzLogger
}
