package nanocms_runners

import (
	"bytes"
	"github.com/google/uuid"
	"golang.org/x/crypto/ssh"
	"log"
)

type SSHSession struct {
	Session *ssh.Session
	Id      string
	Outbuff bytes.Buffer
	Errbuff bytes.Buffer
}

func NewSSHSession(ctx *ssh.Client) *SSHSession {
	ss := new(SSHSession)

	uuid, _ := uuid.NewUUID()
	ss.Id = uuid.String()

	var err error
	ss.Session, err = ctx.NewSession()
	if err != nil {
		log.Fatalln("unable to create SSH session:", err.Error())
	}

	ss.Session.Stdout = &ss.Outbuff
	ss.Session.Stderr = &ss.Errbuff

	return ss
}

// Run a command
func (ss *SSHSession) Run(cmd string) (string, error) {
	defer ss.Session.Close()
	err := ss.Session.Run(cmd)
	return ss.Outbuff.String(), err
}
