package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// WG is the top level service handler. It manages all other components.
type WG struct {
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
