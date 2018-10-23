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
func (s *WG) Start() error {
	fmt.Println("Server has started...")

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL, syscall.SIGINT)

mainloop:
	for {
		switch <-c {
		case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGKILL:
			break mainloop
		}
	}

	fmt.Println("Server is exiting...")
	return nil
}
