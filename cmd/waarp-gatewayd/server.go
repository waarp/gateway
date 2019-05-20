package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"code.waarp.fr/waarp/gateway-ng/pkg/conf"
	"code.waarp.fr/waarp/gateway-ng/pkg/log"
)

// WG is the top level service handler. It manages all other components.
type WG struct {
	*log.Logger
	Config *conf.ServerConfig
}

// NewWG creates a new application
func NewWG(config *conf.ServerConfig) *WG {
	return &WG{
		Config: config,
		Logger: log.NewLogger(),
	}
}

// Start starts the main service of the Gateway
func (wg *WG) Start() error {
	if err := wg.SetOutput(wg.Config.Log.LogTo, wg.Config.Log.SyslogFacility); err != nil {
		return fmt.Errorf("log configuration failed: %s", err.Error())
	}
	if err := wg.SetLevel(wg.Config.Log.Level); err != nil {
		return fmt.Errorf("log configuration failed: %s", err.Error())
	}
	wg.Info("Waarp Gateway NG is starting")
	wg.Infof("Waarp Gateway NG has started")

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)

mainloop:
	for {
		switch <-c {
		case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			break mainloop
		}
	}

	wg.Info("Server is exiting...")
	return nil
}
