package gatewayd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"code.waarp.fr/waarp/gateway-ng/pkg/admin"
	"code.waarp.fr/waarp/gateway-ng/pkg/conf"
	"code.waarp.fr/waarp/gateway-ng/pkg/tk/service"
)

// WG is the top level service handler. It manages all other components.
type WG struct {
	*service.Environment
}

// NewWG creates a new application
func NewWG(config *conf.ServerConfig) *WG {
	return &WG{
		Environment: service.NewEnvironment(config),
	}
}

//
func (wg *WG) initServices() {
	wg.Services = make(map[string]service.Service)

	adminServer := &admin.Server{
		Environment: wg.Environment,
	}

	wg.Services[service.Admin] = adminServer
}

func (wg *WG) startServices() {
	for _, wgService := range wg.Services {
		_ = wgService.Start()
	}
}

func (wg *WG) stopServices() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	for _, wgService := range wg.Services {
		_ = wgService.Stop(ctx)
	}
}

// Start starts the main service of the Gateway
func (wg *WG) Start() error {
	if err := wg.SetOutput(wg.Conf.Log.LogTo, wg.Conf.Log.SyslogFacility); err != nil {
		return fmt.Errorf("log configuration failed: %s", err.Error())
	}
	if err := wg.SetLevel(wg.Conf.Log.Level); err != nil {
		return fmt.Errorf("log configuration failed: %s", err.Error())
	}
	wg.Info("Waarp Gateway NG is starting")
	wg.initServices()
	wg.startServices()
	wg.Infof("Waarp Gateway NG has started")

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)

mainloop:
	for {
		switch <-c {
		case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT:
			wg.stopServices()
			break mainloop
		}
	}

	wg.Info("Server is exiting...")
	return nil
}
