/*
	SSH runner.
	Run states remotely without a client over SSH connection.
*/

package nanocms_runners

import (
	"encoding/json"
	"fmt"
	"os/user"
	"path"
	"strings"
)

type SSHRunner struct {
	BaseRunner
	_hosts       []string
	_rsapath     string
	_sshport     int
	_sshverify   bool
	_user        *user.User
	_remote_user string
}

func NewSSHRunner() *SSHRunner {
	shr := new(SSHRunner)
	shr.ref = shr
	shr._errcode = ERR_INIT
	shr._response = &RunnerResponse{}
	shr._hosts = make([]string, 0)
	shr._sshport = 22
	shr._sshverify = true
	shr.SetUserRSAKeys("")

	return shr
}

// AddHost appends another remote host
func (shr *SSHRunner) AddHost(fqdn string) *SSHRunner {
	shr._hosts = append(shr._hosts, fqdn)
	return shr
}

// SetRemoteUserame sets remote username.
func (shr *SSHRunner) SetRemoteUsername(username string) *SSHRunner {
	shr._remote_user = username
	return shr
}

// SetUserRSAKeys will set a root directory to the RSA keypair and "known_hosts" database file
// based on "$HOME/.ssh" from the given username. If an empty string passed, current user is chosen.
func (shr *SSHRunner) SetUserRSAKeys(username string) *SSHRunner {
	var err error
	if username == "" {
		shr._user, _ = user.Current()
	} else {
		shr._user, err = user.Lookup(username)
		if err != nil {
			panic("User not found")
		}
	}
	shr._rsapath = path.Join(shr._user.HomeDir, ".ssh")
	return shr
}

// SetRSAKeys will set a root directory to the RSA keypair and "known_hosts" database file.
// If an empty string is provided, "$HOME/.ssh" is used instead.
func (shr *SSHRunner) SetRSAKeys(rsapath string) *SSHRunner {
	if rsapath != "" {
		shr._rsapath = rsapath
	}
	return shr
}

// SetSSHHostVerification enables (true, default) or disables (false) the remote host verification,
// based on the "known_hosts" database.
func (shr *SSHRunner) SetSSHHostVerification(hvf bool) *SSHRunner {
	shr._sshverify = hvf
	return shr
}

// SetSSHPort sets an alternative SSH port if needed. Default is 22.
func (shr *SSHRunner) SetSSHPort(port int) *SSHRunner {
	shr._sshport = port
	return shr
}

// Run module with the parameters
func (shr *SSHRunner) callShell(args interface{}) ([]RunnerHostResult, error) {
	result := make([]RunnerHostResult, 0)
	for _, fqdn := range shr._hosts {
		ret := shr.callHost(fqdn, args)
		result = append(result, *ret)
	}
	return result, nil
}

// Run ansible module remotely, assuming Ansible is installed there.
// This runner does not copy anything between the machines, and the Ansible has to be pre-installed already.
// One way of doing it is to call "shell" command and add it there.
func (shr *SSHRunner) callAnsibleModule(name string, kwargs map[string]interface{}) ([]RunnerHostResult, error) {
	name = strings.Replace(name, "ansible.", "", 1)
	result := make([]RunnerHostResult, 0)

	callerJSON := map[string]interface{}{
		"ANSIBLE_MODULE_ARGS": kwargs,
	}
	data, _ := json.Marshal(callerJSON)
	for _, fqdn := range shr._hosts {
		fmt.Println(">>>>>", fqdn, name, string(data))
		ret := shr.callHost(fqdn, []interface{}{
			map[interface{}]interface{}{
				name: fmt.Sprintf("echo '%s' | python3 /usr/lib/python3.6/site-packages/ansible/modules/commands/%s.py", string(data), name),
			}})
		result = append(result, *ret)
	}
	return result, nil
}

// Call a single host with a series of serial, synchronous commands, ensuring their order.
func (shr *SSHRunner) callHost(fqdn string, args interface{}) *RunnerHostResult {
	response := make(map[string]RunnerStdResult)
	result := &RunnerHostResult{
		Host:     fqdn,
		Response: response,
	}
	for _, command := range args.([]interface{}) {
		for cid, cmd := range command.(map[interface{}]interface{}) {
			remote := NewSshShell(shr._rsapath).
				SetRemoteUsername(shr._remote_user).
				SetFQDN(fqdn).SetPort(shr._sshport).
				SetHostVerification(shr._sshverify).
				Connect()
			defer remote.Disconnect()
			session := remote.NewSession()
			_, err := session.Run(cmd.(string))
			out := &RunnerStdResult{
				Stdout: session.Outbuff.String(),
				Stderr: session.Errbuff.String(),
			}
			if err != nil {
				out.Errmsg = err.Error()
				out.Errcode = ERR_FAILED
			}
			response[cid.(string)] = *out
		}
	}
	return result
}
