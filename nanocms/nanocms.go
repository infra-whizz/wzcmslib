/*
NanoCMS is a micro-configuration management system, which is embedded
into the clbd. It runs nanostates that calls pre-installed Ansible modules on
the client machines.
*/

package nanocms_backend

import (
	"fmt"
	scp "github.com/bramvdbogaerde/go-scp"
	"github.com/bramvdbogaerde/go-scp/auth"
	"github.com/isbm/nano-cms/nanorunners"
	"github.com/isbm/nano-cms/nanostate"
	"github.com/isbm/nano-cms/nanostate/compiler"
	"github.com/isbm/nano-cms/nanoutils"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
	"os"
	"os/user"
	"path"
)

var logger *logrus.Logger

func init() {
	logger = nanoutils.GetTextLogger(logrus.DebugLevel, os.Stdout)
}

type NanoCms struct {
	stateindex      *nanocms_state.NanoStateIndex
	sshKeysDeployed bool
	staticdataRoot  string
}

func NewNanoCms() *NanoCms {
	cms := new(NanoCms)
	cms.stateindex = nanocms_state.NewNanoStateIndex()
	cms.sshKeysDeployed = false
	return cms
}

// GetStateIndex returns an instance of a state index.
// XXX: This is subject to change, because nanostate API
// are unstable, so is the indexer.
func (n *NanoCms) GetStateIndex() *nanocms_state.NanoStateIndex {
	return n.stateindex
}

// SetStaticDataRoot sets configuration of the static data
func (n *NanoCms) SetStaticDataRoot(root string) *NanoCms {
	n.staticdataRoot = root
	return n
}

// RunStateSSH is to run nanostate over SSH
func (n *NanoCms) RunStateSSH(state *nanocms_state.Nanostate, fqdns ...string) *nanocms_runners.RunnerResponse {
	logger.Debugf("Running state '%s' on %d machines", state.Id, len(fqdns))
	shr := nanocms_runners.NewSSHRunner().
		SetPermanentMode("/opt/nanocms").
		SetStaticDataRoot(n.staticdataRoot).
		SetRemoteUsername("root").
		SetSSHHostVerification(false)
	for _, fqdn := range fqdns {
		shr.AddHost(fqdn)
	}
	logger.Debugf("Run success: %t", shr.Run(state))

	return shr.Response()
}

// LoadNstFile loads a nanostate from a file path
func (n *NanoCms) LoadNstFile(statepath string) *nanocms_state.Nanostate {
	// Compile nanostate
	cpr := nanocms_compiler.NewNstCompiler()
	err := cpr.LoadFile(statepath)
	if err != nil {
		panic(err)
	}

	// Load compiled tree
	nst := nanocms_state.NewNanostate()
	err = nst.Load(cpr.Tree())
	if err != nil {
		panic(err)
	}

	return nst
}

// SshCopyId is copying SSH keys to the target machine, using username and password
func (n *NanoCms) SshCopyId(fqdn string, username string, password string) bool {
	var conf ssh.ClientConfig
	var cnt scp.Client
	var err error

	// Try if key is there already
	u, _ := user.Current()
	logger.Debugf("Probing SSH connection via keys to %s as %s", fqdn, username)
	conf, _ = auth.PrivateKey(username, path.Join(u.HomeDir, ".ssh", "id_rsa"), ssh.InsecureIgnoreHostKey())
	cnt = scp.NewClient(fqdn+":22", &conf)
	err = cnt.Connect()
	if err == nil {
		cnt.Close()
		logger.Debugf("SSH connection for user %s on %s is ready", username, fqdn)
		return true
	}
	logger.Debugf("No SSH key-based connection yet at %s for %s, deploying...", fqdn, username)

	// Pre-create .ssh directory, if none is there, otherwise key copy will fail and nothing will work
	// This SSH shell is only over password auth, so no RSA keypair is passed
	kcnt := nanocms_runners.NewSshShell("").
		SetFQDN(fqdn).
		SetHostVerification(false).
		SetRemoteUsername(username).
		SetRemotePassword(password).Connect()
	kss := kcnt.NewSession()
	_, kerr := kss.Run("mkdir -p .ssh")
	kcnt.Disconnect()
	if kerr != nil {
		logger.Errorln("Unable to pre-create .ssh directory, so the id_rsa.pub won't be copied properly. Exiting.")
		return false
	}

	// No key yet, copy & add
	conf, _ = auth.PasswordKey(username, password, ssh.InsecureIgnoreHostKey())
	cnt = scp.NewClient(fqdn+":22", &conf)
	err = cnt.Connect()
	if err != nil {
		logger.Errorln("Error connecting to the remote machine:", err.Error())
		return false
	}
	logger.Infof("Connected to the remote '%s' via SSH as user '%s'", fqdn, username)

	usr, _ := user.Current()
	fh, err := os.Open(path.Join(usr.HomeDir, ".ssh", "id_rsa.pub"))
	if err != nil {
		logger.Errorln("Error accessing id_rsa.pub key:", err.Error())
	}

	defer fh.Close()
	defer cnt.Close()

	tempkey := path.Join("/root", ".ssh", fqdn+".pub")
	err = cnt.CopyFile(fh, tempkey, "0600")
	if err != nil {
		logger.Errorln("Error copying pub", err.Error())
	}
	logger.Debugln("Copied", tempkey)

	rcnt, err := ssh.Dial("tcp", cnt.Host, cnt.ClientConfig)
	if err != nil {
		logger.Errorln("Error opening runner connection:", err.Error())
		return false
	}
	defer rcnt.Close()

	rssh, err := rcnt.NewSession()
	if err != nil {
		logger.Errorln("Error opening runner session:", err.Error())
		return false
	}
	defer rssh.Close()

	err = rssh.Run(fmt.Sprintf("cat %s >> /root/.ssh/authorized_keys;rm %s", tempkey, tempkey))
	if err != nil {
		logger.Errorln("Error running command:", err.Error())
		return false
	}
	logger.Debugln("Updated authorized_keys with", tempkey)

	return true
}

// Initialise cluster schema
func (n *NanoCms) StartUp() {}

// StageNode runs node stanging by calling the Node Controller
// to wipe it out and prepare for being added to the cluster.
//
// To stage any Uyuni Server, one needs to do the following:
//   - Run Uyuni Server in cluster node mode
//   - Configure and start Node Controller on it
//
// After stage node message is received on Node Controller,
// it should:
//   - reset the database to the initial state (remove all
//     systems, channels etc)
//   - Confirm this done and ack the response
//
// All this is achieved by installing a ncd binary and let it
// run a nanostate.
func (n *NanoCms) Bootstrap(fqdn string, stateid string, user string, password string) *nanocms_runners.RunnerResponse {
	logger.Debugf("Bootstrapping node '%s'...", fqdn)

	if !n.sshKeysDeployed {
		logger.Debugf("Copying SSH id as %s", user)
		n.sshKeysDeployed = n.SshCopyId(fqdn, user, password)
	}

	nstfile := n.GetStateIndex().GetStateById(stateid).Path
	logger.Debugf("Running NST file by Id '%s': %s", stateid, nstfile)

	return n.RunStateSSH(n.LoadNstFile(nstfile), fqdn)
}
