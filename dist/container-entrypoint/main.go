// Package main defines an entrypoint forto be used in lcontainers.
//
// It wraps gatewayd executable and sets up its configuration according to the given environment variables.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const (
	loggerName        = "entrypoint"
	defaultConfigFile = "./etc/gatewayd.ini"
	gatewaydBin       = "bin/waarp-gatewayd"

	ExitBadConfFile       = 1
	ExitCodeBadConfValue  = 2
	ExitManagerConfError  = 3
	ExitCannotCreateCerts = 4
	ExitGatewayError      = 5
	exitLogBackendError   = 7
	// ExitDBMigrateError    = 6.

	decimal = 10
	bits16  = 16
	bits64  = 64
)

var (
	ErrMissingUsernameOrPassword = errors.New("the URL to Waarp Manager does not contain the username or the password")
	ErrNoConfFound               = errors.New("no configuration found in the configuration package")
)

func main() {
	logger := getLogger()

	// handleConfigFile exits in case of error
	serverConf := handleConfigFile()

	// TODO: handle database migrations
	// cmd := exec.Command(gatewaydBin, "migrate", "--config", defaultConfigFile, "latest")

	// out, err := cmd.CombinedOutput()
	// if err != nil {
	// 	logger.Critical("Cannot run database migrations: %v. Command output: %s", err, out)
	// 	os.Exit(ExitDBMigrateError)
	// }

	// get conf from manager
	managerURL := os.Getenv("WAARP_GATEWAY_MANAGER_URL")

	if managerURL == "" {
		// start server
		startGatewayServer()

		return
	}
	if err := verifyCertificates(serverConf); err != nil {
		logger.Criticalf("there is an issue with the certificates: %v", err)
		os.Exit(ExitCannotCreateCerts)
	}

	var shouldInitialize bool

	if err := importConfFromManager(serverConf, managerURL); err != nil {
		if errors.Is(err, errConfURLNotFound) {
			logger.Info("This Gateway has not been found in Manager. We will try to register it")
			shouldInitialize = true
		} else {
			logger.Criticalf("Cannot synchronize the gateway with Manager: %v", err)
			os.Exit(ExitManagerConfError)
		}
	}

	if shouldInitialize {
		err2 := initializeGatewayInManager(serverConf, managerURL)
		if err2 != nil {
			logger.Criticalf("cannot register this Gateway in manager: %v", err2)
			os.Exit(ExitManagerConfError)
		}

		logger.Info("The Gateway has been added to Manager. Trying to download conf again")

		if err := importConfFromManager(serverConf, managerURL); err != nil {
			logger.Criticalf("Cannot synchronize the gateway with Manager: %v", err)
			os.Exit(ExitManagerConfError)
		}
	}

	// start server
	startGatewayServer()
}

func startGatewayServer() {
	logger := getLogger()

	if err := startGatewayProccess(); err != nil {
		logger.Critical(err.Error())
		os.Exit(ExitGatewayError)
	}
}

func startGatewayProccess() error {
	logger := getLogger()

	cmdArgs := []string{"server", "--config", defaultConfigFile}
	if nodeID := os.Getenv("WAARP_GATEWAY_NODE_ID"); nodeID != "" {
		cmdArgs = append(cmdArgs, "--instance", nodeID)
	}

	logger.Info("Starting Waarp Gateway...")
	logger.Debugf("Command used to start Waarp Gateway: %s %s",
		gatewaydBin, strings.Join(cmdArgs, " "))

	ctx := context.Background()
	path := os.Getenv("PATH")
	path = "/app:/app/bin:/app/share:" + path

	cmd := exec.CommandContext(ctx, gatewaydBin, cmdArgs...)
	cmd.Env = append(cmd.Environ(), "PATH="+path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("waarp Gateway exited abnormally: %w", err)
	}

	return nil
}
