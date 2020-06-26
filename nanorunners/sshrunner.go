/*
	SSH runner.
	Run states remotely without a client over SSH connection.
*/

package nanocms_runners

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
	"reflect"
	"strings"

	"github.com/bramvdbogaerde/go-scp"
	"github.com/bramvdbogaerde/go-scp/auth"
	"golang.org/x/crypto/ssh"
)

type SSHRunner struct {
	BaseRunner
	_hosts       []string
	stateRoots   []string
	_rsapath     string
	_sshport     int
	_sshverify   bool
	_user        *user.User
	_remote_user string
	_perma_dir   string
	_static_data string // This is a directory root for runners installation.
}

func NewSSHRunner() *SSHRunner {
	shr := new(SSHRunner)
	shr.ref = shr
	shr._errcode = ERR_INIT
	shr._response = &RunnerResponse{}
	shr._hosts = make([]string, 0)
	shr.stateRoots = make([]string, 0)
	shr._sshport = 22
	shr._sshverify = true
	shr.SetUserRSAKeys("")

	return shr
}

// Set state roots
func (shr *SSHRunner) setStateRoots(roots ...string) {
	shr.stateRoots = append(shr.stateRoots, roots...)
}

// AddHost appends another remote host
func (shr *SSHRunner) AddHost(fqdn string) *SSHRunner {
	shr._hosts = append(shr._hosts, fqdn)
	return shr
}

/*
SetPermanentMode takes a root path where it will will create
the following structure:

  $ROOT/bin
	   /etc
	   /modules

In "bin" directory local runners are stored; "etc" contains
possible configurations, if any; "modules" will stockpile on demand
remote modules, if they are downloaded.
*/
func (shr *SSHRunner) SetPermanentMode(root string) *SSHRunner {
	shr._perma_dir = root
	return shr
}

// SetStaticDataRoot is a directory where runners and other client-related
// data is located. Static directory should have a specified structure.
// For example, for runners it should be "runners/<ARCH>/",
// e.g. "runners/x86_64/", "runners/arm/" etc.
func (shr *SSHRunner) SetStaticDataRoot(root string) *SSHRunner {
	shr._static_data = root
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
		ret := shr.callHost(fqdn, args, false)
		result = append(result, *ret)
	}
	return result, nil
}

// Converts kwargs to a command line
func (shr *SSHRunner) kwargsToCli(kwargs map[string]interface{}) string {
	var buff bytes.Buffer
	for k, v := range kwargs {
		var pv string
		if reflect.ValueOf(v).Kind() == reflect.Array {
			for _, elem := range v.([]interface{}) {
				pv += elem.(string) + " "
			}
			pv = strings.TrimSpace(pv)
		} else {
			pv = v.(string)
		}
		buff.WriteString(fmt.Sprintf("%s='%s' ", k, pv))
	}
	return strings.TrimSpace(buff.String())
}

// Run ansible module remotely, assuming Ansible is installed there.
// This runner does not copy anything between the machines, and the Ansible has to be pre-installed already.
// One way of doing it is to call "shell" command and add it there.
func (shr *SSHRunner) callAnsibleModule(name string, kwargs map[string]interface{}) ([]RunnerHostResult, error) {
	name = strings.Replace(name, "ansible.", "", 1)
	result := make([]RunnerHostResult, 0)

	for _, fqdn := range shr._hosts {
		ret := shr.callHost(fqdn, []interface{}{
			map[interface{}]interface{}{
				name: fmt.Sprintf("%s %s %s", "/opt/nanocms/bin/ansiblerunner", name, shr.kwargsToCli(kwargs)),
			}}, true)
		result = append(result, *ret)
	}
	return result, nil
}

// Installs permanent client
func (shr *SSHRunner) installPermanentClient(shell *SshShell) {
	if shr._perma_dir == "" {
		log.Println("Attempt to install permanent client, but no permanent directory has been given")
		return
	}

	if shr._static_data == "" {
		log.Println("Attempt to install permanent client, but no static data storage directory has been given")
		return
	}

	// Create directories
	for _, dirname := range []string{"", "bin", "etc", "modules"} {
		session := shell.NewSession()
		target := path.Join(shr._perma_dir, dirname)
		_, err := session.Run(fmt.Sprintf("mkdir %s", target))
		if err != nil {
			fmt.Println("Errored:", target, err.Error())
			return
		}
	}

	// Get remote architecture
	arch, _ := shell.NewSession().Run("uname -i")
	arch = strings.TrimSpace(arch)

	runners, err := ioutil.ReadDir(path.Join(shr._static_data, "runners", arch))
	if err != nil {
		panic(err)
	}

	// Upload runners
	conf, _ := auth.PrivateKey(shell._user, path.Join(shr._rsapath, "id_rsa"), ssh.InsecureIgnoreHostKey())
	cnt := scp.NewClient(shell.GetFQDN()+":22", &conf)
	err = cnt.Connect()
	if err != nil {
		panic(err)
	}
	defer cnt.Close()

	for _, runnerFile := range runners {
		src := path.Join(shr._static_data, "runners", arch, runnerFile.Name())
		dst := path.Join(shr._perma_dir, "bin", path.Base(src))
		nfo, err := os.Stat(src)
		if !os.IsNotExist(err) && !nfo.IsDir() {
			fh, err := os.Open(src)
			if err != nil {
				fmt.Println("Error accessing runner:", err.Error())
			}
			err = cnt.CopyFile(fh, dst, "0755")
			if err != nil {
				fmt.Println("Error copying runner to the remote:", err.Error())
			}
			fh.Close()
		}
	}
}

// Call a single host with a series of serial, synchronous commands, ensuring their order.
func (shr *SSHRunner) callHost(fqdn string, args interface{}, jsonout bool) *RunnerHostResult {
	response := make(map[string]RunnerStdResult)
	result := &RunnerHostResult{
		Host:     fqdn,
		Response: response,
	}
	remote := NewSshShell(shr._rsapath).SetRemoteUsername(shr._remote_user).SetFQDN(fqdn).
		SetPort(shr._sshport).SetHostVerification(shr._sshverify).Connect()
	defer remote.Disconnect()

	for _, command := range args.([]interface{}) {
		for cid, cmd := range command.(map[interface{}]interface{}) {
			log.Println("Calling", cmd)
			session := remote.NewSession()
			_, err := session.Run(cmd.(string))

			if err != nil && shr._perma_dir != "" {
				log.Println("First run errored, attempt to install permanent client:", err.Error())
				shr.installPermanentClient(remote)

				session = remote.NewSession()
				_, err = session.Run(cmd.(string)) // Second attempt
			}

			out := &RunnerStdResult{}
			if !jsonout {
				out.Stdout = session.Outbuff.String()
			} else {
				if err := json.Unmarshal(session.Outbuff.Bytes(), &out.Json); err != nil {
					log.Println("Erroneous JSON:", err.Error())
				}
			}
			if err != nil {
				out.Errmsg = err.Error()
				out.Errcode = ERR_FAILED
				out.Stderr = session.Errbuff.String()
			}
			response[cid.(string)] = *out
		}
	}
	return result
}
