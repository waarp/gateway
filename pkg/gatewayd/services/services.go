// Package services provides lists of all the gateway's internal services.
// This includes core services, clients and servers.
package services

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/puzpuzpuz/xsync/v4"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//nolint:gochecknoglobals //global vars are required here
var (
	Core    = serviceList{}
	Clients = NewServiceMap[*model.Client, Client]()
	Servers = NewServiceMap[*model.LocalAgent, Server]()
)

type (
	Service = protocol.StartStopper
	Client  = protocol.Client
	Server  = protocol.Server
)

type ServiceNotFoundError int64

func (e ServiceNotFoundError) Error() string {
	return fmt.Sprintf("service %d not found", e)
}

type serviceList []Service

func (s *serviceList) Add(service Service) { *s = append(*s, service) }

type ServiceMap[O database.Identifier, S Service] struct {
	m *xsync.Map[int64, S]
}

func NewServiceMap[O database.Identifier, T Service]() ServiceMap[O, T] {
	return ServiceMap[O, T]{m: xsync.NewMap[int64, T]()}
}

func (s ServiceMap[O, S]) Get(obj O) (S, bool) {
	return s.m.Load(obj.GetID())
}

func (s ServiceMap[O, S]) Add(obj O, service S) {
	s.m.Store(obj.GetID(), service)
}

func (s ServiceMap[O, S]) Exists(obj O) bool {
	_, ok := s.Get(obj)

	return ok
}

func (s ServiceMap[O, S]) Remove(ctx context.Context, obj O) (retErr error) {
	s.m.Compute(obj.GetID(), func(service S, loaded bool) (S, xsync.ComputeOp) {
		if !loaded {
			return service, xsync.CancelOp
		}

		if state, _ := service.State(); state == utils.StateRunning {
			retErr = service.Stop(ctx)
		}

		return service, xsync.DeleteOp
	})

	return retErr //nolint:wrapcheck //no need to wrap here
}

func (s ServiceMap[O, S]) Start(obj O) (started bool, retErr error) {
	s.m.Compute(obj.GetID(), func(service S, loaded bool) (_ S, op xsync.ComputeOp) {
		if !loaded {
			retErr = ServiceNotFoundError(obj.GetID())

			return service, xsync.CancelOp
		}

		if err := service.Start(); err != nil {
			if errors.Is(err, utils.ErrAlreadyRunning) {
				return service, xsync.CancelOp
			}

			retErr = fmt.Errorf("failed to start service: %w", err)

			return service, xsync.CancelOp
		}

		started = true

		return service, xsync.UpdateOp
	})

	return started, retErr
}

func (s ServiceMap[O, S]) Stop(ctx context.Context, obj O) (stopped bool, retErr error) {
	s.m.Compute(obj.GetID(), func(service S, loaded bool) (S, xsync.ComputeOp) {
		if !loaded {
			retErr = ServiceNotFoundError(obj.GetID())

			return service, xsync.CancelOp
		}

		if err := service.Stop(ctx); err != nil {
			if errors.Is(err, utils.ErrNotRunning) {
				return service, xsync.CancelOp
			}

			retErr = fmt.Errorf("failed to start service: %w", err)

			return service, xsync.CancelOp
		}

		stopped = true

		return service, xsync.UpdateOp
	})

	return stopped, retErr
}

func (s ServiceMap[O, S]) Restart(ctx context.Context, obj O) (retErr error) {
	s.m.Compute(obj.GetID(), func(service S, loaded bool) (S, xsync.ComputeOp) {
		if !loaded {
			retErr = ServiceNotFoundError(obj.GetID())

			return service, xsync.CancelOp
		}

		if state, _ := service.State(); state == utils.StateRunning {
			if err := service.Stop(ctx); err != nil {
				retErr = fmt.Errorf("failed to stop service: %w", err)

				return service, xsync.CancelOp
			}
		}

		if err := service.Start(); err != nil {
			retErr = fmt.Errorf("failed to start service: %w", err)

			return service, xsync.CancelOp
		}

		return service, xsync.UpdateOp
	})

	return retErr
}

func (s *ServiceMap[O, S]) StopAll(ctx context.Context) error {
	errChan := make(chan error, s.m.Size())

	wg := sync.WaitGroup{}
	s.m.Range(func(_ int64, service S) bool {
		if state, _ := service.State(); state == utils.StateRunning {
			wg.Go(func() {
				errChan <- service.Stop(ctx)
			})
		}

		return true
	})

	wg.Wait()
	close(errChan)

	return errors.Join(utils.Collect(errChan)...)
}

func (s *ServiceMap[O, S]) Range(f func(int64, S) bool) { s.m.Range(f) }
