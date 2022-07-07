// Package main defines an entrypoint forto be used in lcontainers.
//
// It wraps gatewayd executable and sets up its configuration according to the given environment variables.
package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
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
	// ExitDBMigrateError    = 6.

	decimal                  = 10
	bits16                   = 16
	bits64                   = 64
	sleepDurationBeforeRetry = 5 * time.Second
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
	retryConfDownload := false
	managerURL := os.Getenv("WAARP_GATEWAY_MANAGER_URL")

	if managerURL != "" {
		if err := verifyCertificates(serverConf); err != nil {
			logger.Critical("there is an issue with the certificates: %v", err)
			os.Exit(ExitCannotCreateCerts)
		}

		if err := importConfFromManager(serverConf, managerURL); err != nil {
			if errors.Is(err, errConfURLNotFound) {
				retryConfDownload = true

				logger.Info("This Gateway has not been found in Manager. We will try to register it")
			} else {
				logger.Critical("Cannot synchronize the gateway with Manager: %v", err)
				os.Exit(ExitManagerConfError)
			}
		}
	}

	// start server
	startGatewayServer(serverConf, managerURL, retryConfDownload)
}

func startGatewayServer(serverConf *conf.ServerConfig, managerURL string, retryConfDownload bool) {
	logger := getLogger()

	var wg sync.WaitGroup

	restart := make(chan struct{})

	wg.Add(1)

	go func(restart <-chan struct{}) {
		if err := startGatewayProccess(restart); err != nil {
			logger.Critical(err.Error())
			os.Exit(ExitGatewayError)
		}

		wg.Done()
	}(restart)

	if retryConfDownload {
		time.Sleep(sleepDurationBeforeRetry)

		err2 := initializeGatewayInManager(serverConf, managerURL)
		if err2 != nil {
			logger.Critical("cannot register this Gateway in manager: %v", err2)
			os.Exit(ExitManagerConfError)
		}

		logger.Info("The Gateway has been added to Manager. Trying to download conf again")

		if err := importConfFromManager(serverConf, managerURL); err != nil {
			logger.Critical("Cannot synchronize the gateway with Manager: %v", err)
			os.Exit(ExitManagerConfError)
		}

		restart <- struct{}{}
	}

	wg.Wait()
}

func startGatewayProccess(restart <-chan struct{}) error {
	logger := getLogger()

	cmdArgs := []string{"server", "--config", defaultConfigFile}
	if nodeID := os.Getenv("WAARP_GATEWAY_NODE_ID"); nodeID != "" {
		cmdArgs = append(cmdArgs, "--instance", nodeID)
	}

	logger.Info("Starting Waarp Gateway...")
	logger.Debug("Command used to start Waarp Gateway: %s %s",
		gatewaydBin, strings.Join(cmdArgs, " "))

	for {
		ctx, cancel := context.WithCancel(context.Background())

		cmd := exec.CommandContext(ctx, gatewaydBin, cmdArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		go func() {
			if err := cmd.Run(); err != nil {
				logger.Error("Waarp Gateway exited abnormally: %v", err)
			}
		}()

		<-restart
		cancel()
		logger.Info("Restarting Waarp Gateway...")
	}
}
