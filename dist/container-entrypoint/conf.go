package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"code.bcarlin.xyz/go/logging"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/config"
)

func updateConfig(serverConf *conf.ServerConfig) {
	setStringFromEnv("NAME", &serverConf.GatewayName)
	setStringFromEnv("HOME", &serverConf.Paths.GatewayHome)
	setStringFromEnv("IN_DIR", &serverConf.Paths.DefaultInDir)
	setStringFromEnv("OUT_DIR", &serverConf.Paths.DefaultOutDir)
	setStringFromEnv("TMP_DIR", &serverConf.Paths.DefaultTmpDir)
	setStringFromEnv("LOG_LEVEL", &serverConf.Log.Level)
	setStringFromEnv("LOG_TO", &serverConf.Log.LogTo)
	setStringFromEnv("SYSLOG_FACILITY", &serverConf.Log.SyslogFacility)
	setStringFromEnvDefault("ADMIN_ADDRESS", &serverConf.Admin.Host, "0.0.0.0")

	if err := setPortFromEnv("ADMIN_PORT", &serverConf.Admin.Port); err != nil {
		log.Printf("Configuration error: %v", err)
		os.Exit(ExitCodeBadConfValue)
	}

	if os.Getenv("WAARP_GATEWAY_MANAGER_URL") != "" {
		setStringFromEnvDefault("ADMIN_TLS_CERT", &serverConf.Admin.TLSCert, "/app/etc/admin.crt")
		setStringFromEnvDefault("ADMIN_TLS_KEY", &serverConf.Admin.TLSKey, "/app/etc/admin.key")
	} else {
		setStringFromEnv("ADMIN_TLS_CERT", &serverConf.Admin.TLSCert)
		setStringFromEnv("ADMIN_TLS_KEY", &serverConf.Admin.TLSKey)
	}

	setStringFromEnv("DB_TYPE", &serverConf.Database.Type)
	setStringFromEnv("DB_ADDRESS", &serverConf.Database.Address)
	setStringFromEnv("DB_NAME", &serverConf.Database.Name)
	setStringFromEnv("DB_USER", &serverConf.Database.User)
	setStringFromEnv("DB_PASSWORD", &serverConf.Database.Password)
	setStringFromEnv("DB_TLS_CERT", &serverConf.Database.TLSCert)
	setStringFromEnv("DB_TLS_KEY", &serverConf.Database.TLSKey)
	setStringFromEnvDefault("DB_AES_PASSPHRASE", &serverConf.Database.AESPassphrase, "etc/passphrase.aes")

	if err := setUint64FromEnv("MAX_IN", &serverConf.Controller.MaxTransfersIn); err != nil {
		log.Printf("Configuration error: %v", err)
		os.Exit(ExitCodeBadConfValue)
	}

	if err := setUint64FromEnv("MAX_OUT", &serverConf.Controller.MaxTransfersOut); err != nil {
		log.Printf("Configuration error: %v", err)
		os.Exit(ExitCodeBadConfValue)
	}
}

func setStringFromEnv(envKey string, target *string) {
	val := os.Getenv("WAARP_GATEWAY_" + envKey)
	if val != "" {
		*target = val
	}
}

func setStringFromEnvDefault(envKey string, target *string, defaultValue string) {
	val := os.Getenv("WAARP_GATEWAY_" + envKey)
	if val != "" {
		*target = val

		return
	}

	*target = defaultValue
}

func setPortFromEnv(envKey string, target *uint16) error {
	val := os.Getenv("WAARP_GATEWAY_" + envKey)
	if val != "" {
		p, err := strconv.ParseUint(val, decimal, bits16)
		if err != nil {
			return fmt.Errorf("invalid port from environment: %w", err)
		}

		*target = uint16(p)
	}

	return nil
}

func setUint64FromEnv(envKey string, target *uint64) error {
	val := os.Getenv("WAARP_GATEWAY_" + envKey)
	if val != "" {
		p, err := strconv.ParseUint(val, decimal, bits64)
		if err != nil {
			return fmt.Errorf("invalid value from environment: %w", err)
		}

		*target = p
	}

	return nil
}

func setIntFromEnv(envKey string, target *int) error {
	val := os.Getenv("WAARP_GATEWAY_" + envKey)
	if val != "" {
		p, err := strconv.ParseInt(val, decimal, bits64)
		if err != nil {
			return fmt.Errorf("invalid value from environment: %w", err)
		}

		*target = int(p)
	}

	return nil
}

func handleConfigFile() *conf.ServerConfig {
	logger := logging.GetLogger("entrypoint")
	serverConf := &conf.ServerConfig{}

	confFile := defaultConfigFile
	setStringFromEnv("CONFIG", &confFile)

	parser, err := config.NewParser(serverConf)
	if err != nil {
		logger.Criticalf("Cannot initialize the configuration parser: %v\n", err)
		os.Exit(ExitBadConfFile)
	}

	if pathExists(confFile) {
		logger.Infof("Reading configuration file %q", confFile)

		// load configuration
		err = parser.ParseFile(confFile)
		if err != nil {
			logger.Criticalf("Cannot parse the configuration file: %v\n", err)
			os.Exit(ExitBadConfFile)
		}
	}

	// set up with environment
	logger.Info("Updating configuration file with the environment")
	updateConfig(serverConf)

	// write config
	logger.Infof("Writing configuration file %q", confFile)

	err = parser.WriteFile(confFile)
	if err != nil {
		logger.Criticalf("Cannot write the configuration file: %v", err)
		os.Exit(ExitBadConfFile)
	}

	return serverConf
}
