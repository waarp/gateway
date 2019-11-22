// Package sftp contains the functions necessary to execute a file transfer
// using the SFTP protocol. The package defines both a client and a server for
// SFTP.
package sftp

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/model"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// DoTransfer realise a sftp transfer according to the given Transfer
func DoTransfer(client *sftp.Client, t *model.Transfer, r *model.Rule) error {
	// Do the Transfer
	if r.IsGet {
		return getFile(client, t.SourcePath, t.DestPath)
	}
	return putFile(client, t.SourcePath, t.DestPath)
}

// Context is a struct containing both the SSHClient and the SftpClient
type Context struct {
	SSHClient  *ssh.Client
	SftpClient *sftp.Client
}

// Close the SftpContext by closing the SFTPClient then the SSHClient
func (c *Context) Close() {
	_ = c.SftpClient.Close()
	_ = c.SSHClient.Close()
}

// Connect opens and returns a sftp connection to the remote agent with the given Cert and RemoteAccount
func Connect(r *model.RemoteAgent, c *model.Cert, a *model.RemoteAccount) (*Context, error) {
	// Unmarshal Remote ProtoConfig
	var remoteConf map[string]interface{}
	if err := json.Unmarshal(r.ProtoConfig, &remoteConf); err != nil {
		return nil, err
	}

	// Build Remote address
	addr, err := getRemoteAddress(remoteConf)
	if err != nil {
		return nil, err
	}

	// Create SSH config
	sshConfig, err := getSSHConfig(c, a)
	if err != nil {
		return nil, err
	}

	// Open SSH connection
	conn, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, err
	}
	// Create SFTP Client using ssh connection
	client, err := sftp.NewClient(conn)

	return &Context{
		SSHClient:  conn,
		SftpClient: client,
	}, err
}

// getPartnerAddress return the remote address as "address:port"
func getRemoteAddress(conf map[string]interface{}) (string, error) {
	port, ok := conf["port"].(float64)
	if !ok {
		return "", fmt.Errorf("invalid value (%b) for port", conf["port"])
	}
	return fmt.Sprintf("%s:%d", conf["address"], int(port)), nil
}

// getSshConfig return a SSH ClientConfig using the given Cert and Account
func getSSHConfig(c *model.Cert, a *model.RemoteAccount) (*ssh.ClientConfig, error) {
	key, _, _, _, err := ssh.ParseAuthorizedKey(c.PublicKey) //nolint:dogsled
	if err != nil {
		return nil, err
	}
	pwd, err := model.DecryptPassword(a.Password)
	if err != nil {
		return nil, err
	}

	return &ssh.ClientConfig{
		User: a.Login,
		Auth: []ssh.AuthMethod{
			ssh.Password(string(pwd)),
		},
		HostKeyCallback: ssh.FixedHostKey(key),
	}, nil
}

// getFile retrieves the source File from the sftp client and write in in destination
func getFile(client *sftp.Client, source string, destination string) error {
	// Open remote source file
	remoteFile, err := client.Open(source)
	if err != nil {
		return err
	}
	// Create local destination file
	localFile, err := os.Create(filepath.FromSlash(destination))
	if err != nil {
		return err
	}
	defer func() { _ = localFile.Close() }()

	// Read remote file into local file
	if _, err = remoteFile.WriteTo(localFile); err != nil {
		_ = remoteFile.Close()
		return err
	}
	return remoteFile.Close()
}

// putFile write the source File in the destination on the sftp client
func putFile(client *sftp.Client, source string, destination string) error {
	// Open local source file
	localFile, err := os.Open(filepath.FromSlash(source))
	if err != nil {
		return err
	}
	defer func() { _ = localFile.Close() }()
	// Create remote destination file
	remoteFile, err := client.Create(destination)
	if err != nil {
		return err
	}
	// Read copy local file into remote file
	if _, err = io.Copy(remoteFile, localFile); err != nil {
		_ = remoteFile.Close()
		return err
	}
	return remoteFile.Close()
}
