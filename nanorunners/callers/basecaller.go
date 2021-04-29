package nanocms_callers

import (
	"os"

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
	chroot     string   // Run ansible chrooted, if it is different than "/"
	pce        *os.File
	pceModule  *os.File

	wzlib_logger.WzLogger
}
