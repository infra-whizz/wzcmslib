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
	pyexe      []string // Interpreter shebang, e.g. "/usr/bin/python3" or "/usr/bin/env python" etc.
	wzlib_logger.WzLogger
}
