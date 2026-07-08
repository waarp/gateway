// Package main defines an entrypoint forto be used in lcontainers.
//
// It wraps gatewayd executable and sets up its configuration according to the
// given environment variables.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
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
	ExitDBMigrateError    = 6
	exitLogBackendError   = 7

	defaultDBMigrationTimeout = 30
	dbRetryInterval           = 2

	decimal = 10
	bits16  = 16
	bits64  = 64
)

var (
	ErrMissingUsernameOrPassword = errors.New("the URL to Waarp Manager does not contain the username or the password")
	ErrNoConfFound               = errors.New("no configuration found in the configuration package")
)

// runMigrations executes database migrations with retry logic for database
// availability.
// It will retry for defaultDBMigrationTimeout seconds (configurable via
// WAARP_GATEWAY_DB_MIGRATION_TIMEOUT) before failing with ExitDBMigrateError.
func runMigrations(serverConf *conf.ServerConfig) {
	logger := getLogger()

	// Skip if no database configured
	if serverConf.Database.Type == "" {
		logger.Info("No database configured, skipping migrations")
		return
	}

	logger.Info("Running database migrations...")

	// Get timeout from environment or use default
	timeout := defaultDBMigrationTimeout
	if timeoutStr := os.Getenv("WAARP_GATEWAY_DB_MIGRATION_TIMEOUT"); timeoutStr != "" {
		if t, err := strconv.Atoi(timeoutStr); err == nil && t > 0 {
			timeout = t
		} else {
			logger.Warningf("Invalid WAARP_GATEWAY_DB_MIGRATION_TIMEOUT value '%s', using default %d seconds",
				timeoutStr, defaultDBMigrationTimeout)
		}
	}

	deadline := time.Now().Add(time.Duration(timeout) * time.Second)

	for {
		cmd := exec.Command(gatewaydBin, "migrate", "--config", defaultConfigFile)
		out, err := cmd.CombinedOutput()

		for line := range strings.SplitSeq(string(out), "\n") {
			if line != "" {
				logger.Info(line)
			}
		}

		if err == nil {
			logger.Info("Database migrations completed successfully")
			return
		}

		// Check if deadline exceeded
		if time.Now().After(deadline) {
			logger.Criticalf("Database migrations failed after timeout: %v", err)
			os.Exit(ExitDBMigrateError)
		}

		// Log and retry
		logger.Warningf("Database migration attempt failed, retrying in %d seconds... Error: %v",
			dbRetryInterval, err)
		time.Sleep(time.Duration(dbRetryInterval) * time.Second)
	}
}

func main() {
	logger := getLogger()

	// handleConfigFile exits in case of error
	serverConf := handleConfigFile()

	// get conf from manager
	managerURL := os.Getenv("WAARP_GATEWAY_MANAGER_URL")

	if managerURL == "" {
		// Run migrations before starting server
		runMigrations(serverConf)

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

	// Run migrations after configuration is synchronized from Manager
	runMigrations(serverConf)

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
