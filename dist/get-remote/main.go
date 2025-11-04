package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
)

const (
	exitLockFile    = 1
	exitNoEnv       = 2
	exitNoEnvVar    = 4
	exitBadListFile = 5
)

var errUnknownProtocol = errors.New("unknown protocol")

//nolint:forbidigo // main function must be able to print
func main() {
	gw_address := os.Getenv("WAARP_GATEWAY_ADDRESS")
	if gw_address == "" {
		fmt.Print("The environment variable WAARP_GATEWAY_ADDRESS must be defined.\n")
		os.Exit(exitNoEnvVar)
	}

	gw_insecure := os.Getenv("WAARP_GATEWAY_INSECURE") != ""

	// find out env
	files, err := getPaths()
	if err != nil {
		fmt.Println(err)
		os.Exit(exitNoEnv)
	}

	// lock/unlock
	l := lock{files.lockFile()}
	if l.isLocked() {
		fmt.Print("Another instance of get-remote is already running.\n")
		os.Exit(exitLockFile)
	}

	// parse file
	if !pathExists(files.listFile()) {
		fmt.Printf("No file get-files.list found.\n")

		return
	}

	checklist, err := parseListFile(files.listFile())
	if err != nil {
		fmt.Printf("Cannot parse list file: %v\n", err)
		os.Exit(exitBadListFile)
	}

	err = l.acquire()
	if err != nil {
		os.Exit(exitLockFile)
	}

	defer func() {
		if err2 := l.release(); err2 != nil {
			fmt.Printf("Cannot release lock: %v\n", err2)
			os.Exit(exitLockFile)
		}
	}()

	processChecks(checklist, gw_address, gw_insecure)
}

func processChecks(checklist []check, addr string, insecure bool) {
	for i := range checklist {
		c := &checklist[i]

		fmt.Printf("Checking files on %q for flow %q.\n",
			c.hostid, c.flowID)

		files, listErr := listFiles(c, addr, insecure)
		if listErr != nil {
			fmt.Printf("Cannot check files on %s for flow %s: %v\n",
				c.hostid, c.flowID, listErr)

			continue
		}

		// create dl
		for _, file := range files {
			fmt.Printf("Add transfer for file %q.\n", file)

			transfer := api.InTransfer{
				Rule:    c.rule,
				Partner: c.hostid,
				Account: c.user,
				File:    file,
				IsSend:  api.Nullable[bool]{Valid: true, Value: false},
			}

			if err := addTransfer(&transfer, addr, insecure); err != nil {
				fmt.Printf("Cannot add transfer for file %q: %v\n", file, err)

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
	content, err := os.ReadFile(filepath.Clean(p))
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

func listFiles(c *check, addr string, insecure bool) ([]string, error) {
	partner, err := getPartner(c.hostid, addr, insecure)
	if err != nil {
		return nil, fmt.Errorf("could not find partner %s: %w", c.hostid, err)
	}

	account, err := getAccount(partner.Name, c.user, addr, insecure)
	if err != nil {
		return nil, fmt.Errorf("could not find account %s: %w", c.user, err)
	}

	rule, err := getRule(c.rule, addr, insecure)
	if err != nil {
		return nil, fmt.Errorf("could not find rule %s: %w", c.rule, err)
	}

	client, err := newClient(partner.Protocol)
	if err != nil {
		return nil, fmt.Errorf("could not initialize client for %s: %w", c.hostid, err)
	}

	if netErr := client.Connect(partner, account, addr, insecure); netErr != nil {
		return nil, fmt.Errorf("could not connect to partner %s: %w", c.hostid, netErr)
	}

	defer client.Close()

	files, err := client.List(rule, c.pattern)
	if err != nil {
		return nil, fmt.Errorf("could not list file from partner %s: %w", c.hostid, err)
	}

	return files, nil
}

func getPartner(partner, addr string, insecure bool) (*api.OutPartner, error) {
	restPath, urlErr := url.JoinPath(addr, "/api/partners", partner)
	if urlErr != nil {
		return nil, fmt.Errorf("failed to build URL: %w", urlErr)
	}
	apiPartner := &api.OutPartner{}

	if err := get(apiPartner, restPath, insecure); err != nil {
		return nil, err
	}

	return apiPartner, nil
}

func getAccount(partner, account, addr string, insecure bool) (*api.OutRemoteAccount, error) {
	restPath, urlErr := url.JoinPath(addr, "/api/partners", partner, "accounts", account)
	if urlErr != nil {
		return nil, fmt.Errorf("failed to build URL: %w", urlErr)
	}
	apiAccount := &api.OutRemoteAccount{}

	if err := get(apiAccount, restPath, insecure); err != nil {
		return nil, err
	}

	return apiAccount, nil
}

func getRule(rule, addr string, insecure bool) (*api.OutRule, error) {
	restPath, urlErr := url.JoinPath(addr, "/api/rules", rule, "receive")
	if urlErr != nil {
		return nil, fmt.Errorf("failed to build URL: %w", urlErr)
	}
	apiRule := &api.OutRule{}

	if err := get(apiRule, restPath, insecure); err != nil {
		return nil, err
	}

	return apiRule, nil
}

func addTransfer(inTransfer *api.InTransfer, addr string, insecure bool) error {
	restPath, urlErr := url.JoinPath(addr, "/api/transfers")
	if urlErr != nil {
		return fmt.Errorf("failed to build URL: %w", urlErr)
	}

	if err := add(inTransfer, restPath, insecure); err != nil {
		return fmt.Errorf("failed to add transfer for file %s: %w", inTransfer.File, err)
	}

	return nil
}
