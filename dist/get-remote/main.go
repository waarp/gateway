package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	exitLockFile    = 1
	exitNoEnv       = 2
	exitNoLogger    = 3
	exitNoEnvVar    = 4
	exitBadListFile = 5
)

var errUnknownProtocol = errors.New("unknown protocol")

//nolint:forbidigo // main function must be able to print
func main() {
	if os.Getenv("WAARP_GATEWAY_ADDRESS") == "" {
		fmt.Println("The environment variable WAARP_GATEWAY_ADDRESS must be defined")
		os.Exit(exitNoEnvVar)
	}

	// find out env
	files, err := getPaths()
	if err != nil {
		fmt.Println(err)
		os.Exit(exitNoEnv)
	}

	// setup logger
	log, err := newLogger(files.logFile())
	if err != nil {
		fmt.Println(err)
		os.Exit(exitNoLogger)
	}

	// lock/unlock
	l := lock{files.lockFile()}
	if l.isLocked() {
		log.Print("Another instance of get-remote is already running.")
		os.Exit(exitLockFile)
	}

	// parse file
	if !pathExists(files.listFile()) {
		log.Printf("No file get-files.list found")

		return
	}

	checklist, err := parseListFile(files.listFile())
	if err != nil {
		log.Printf("Cannot parse list file: %v", err)
		os.Exit(exitBadListFile)
	}

	err = l.acquire()
	if err != nil {
		os.Exit(exitLockFile)
	}

	defer func() {
		if err2 := l.release(); err2 != nil {
			log.Printf("Cannot release lock: %v\n", err2)
			os.Exit(exitLockFile)
		}
	}()

	processChecks(log, files, checklist)
}

func processChecks(log *logger, p paths, checklist []check) {
	for i := range checklist {
		c := &checklist[i]

		log.Printf("Checking files on %q for flow %q",
			c.remoteHost, c.flowID)

		files, err := listFiles(c)
		if err != nil {
			log.Printf("Cannot check files on %s for flow %s: %v",
				c.remoteHost, c.flowID, err)

			continue
		}

		// create dl
		for _, file := range files {
			log.Printf("Add transfer for file %q", file)

			if err := addTransfer(c, p, file); err != nil {
				log.Printf("Cannot add transfer for file %q: %v", file, err)

				continue
			}
		}
	}
}

type check struct {
	flowID      string
	rule        string
	proto       string
	hostid      string
	remoteHost  string
	remotePort  string
	authentMode string
	user        string
	password    string
	pattern     string
}

func parseListFile(p string) ([]check, error) {
	content, err := ioutil.ReadFile(filepath.Clean(p))
	if err != nil {
		return nil, fmt.Errorf("cannot read file %q: %w", p, err)
	}

	lines := strings.Split(string(content), "\n")

	var checks []check

	for i := range lines {
		if lines[i] == "" {
			continue
		}

		checks = append(checks, parseLine(lines[i]))
	}

	return checks, nil
}

func parseLine(line string) check {
	parts := strings.Split(line, ",")

	return check{
		flowID:      parts[0],
		rule:        parts[1],
		proto:       parts[2],
		hostid:      parts[3],
		remoteHost:  parts[4],
		remotePort:  parts[5],
		authentMode: parts[6],
		user:        parts[7],
		password:    parts[8],
		pattern:     parts[9],
	}
}

func listFiles(c *check) ([]string, error) {
	switch c.proto {
	case "sftp":
		return listFilesSftp(c)
	default:
		return nil, fmt.Errorf("unsupported protocol %q: %w", c.proto, errUnknownProtocol)
	}
}

func listFilesSftp(c *check) ([]string, error) {
	sshconfig := &ssh.ClientConfig{
		User: c.user,
		Auth: []ssh.AuthMethod{
			ssh.Password(c.password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // ignore for now
	}

	conn, err := ssh.Dial("tcp", c.remoteHost+":"+c.remotePort, sshconfig)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to %q: %w", c.remoteHost, err)
	}
	defer conn.Close() //nolint:errcheck // nothing to handle the error

	// create new SFTP client
	client, err := sftp.NewClient(conn)
	if err != nil {
		return nil, fmt.Errorf("cannot create SFTP session for %q: %w", c.remoteHost, err)
	}
	defer client.Close() //nolint:errcheck // nothing to handle the error

	dir := c.pattern
	if strings.Contains(c.pattern, "*") {
		dir = path.Dir(c.pattern)
		c.pattern = path.Base(c.pattern)
	}

	fileinfoList, err := client.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot list remote files: %w", err)
	}

	fileinfoList, err = filterFileInfoList(fileinfoList, c.pattern)
	if err != nil {
		return nil, fmt.Errorf("cannot filter remote files: %w", err)
	}

	filelist := make([]string, len(fileinfoList))
	for i := range fileinfoList {
		filelist[i] = path.Join(dir, fileinfoList[i].Name())
	}

	return filelist, nil
}

func filterFileInfoList(fil []os.FileInfo, pattern string) ([]os.FileInfo, error) {
	rv := []os.FileInfo{}

	for _, fi := range fil {
		if fi.IsDir() || strings.HasPrefix(fi.Name(), ".") {
			continue
		}

		matches, err := path.Match(pattern, fi.Name())
		if err != nil {
			return rv, fmt.Errorf("an error occurred while testing path %q: %w", fi.Name(), err)
		}

		if !matches {
			continue
		}

		rv = append(rv, fi)
	}

	return rv, nil
}

func addTransfer(c *check, p paths, file string) error {
	//nolint: gosec // ignore for now
	cmd := exec.Command(filepath.Join(p.binDir, "waarp-gateway"),
		"transfer", "add",
		"--file", path.Base(file),
		"--partner", c.hostid,
		"--login", c.user,
		"--rule", c.rule,
		"--way", "receive")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed with output %q and error: %w",
			string(out), err)
	}

	return nil
}
