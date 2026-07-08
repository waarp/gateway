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
	"code.waarp.fr/apps/gateway/gateway/pkg/protocols/protocol"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

//nolint:gochecknoglobals //global vars are required here
var (
	Core    = serviceList{}
	Clients = NewServiceMap[Client]()
	Servers = NewServiceMap[Server]()
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

type ServiceMap[T Service] struct {
	m *xsync.Map[int64, T]
}

func NewServiceMap[T Service]() ServiceMap[T] {
	return ServiceMap[T]{m: xsync.NewMap[int64, T]()}
}

func (s ServiceMap[T]) Get(obj database.Identifier) (T, bool) {
	return s.m.Load(obj.GetID())
}

func (s ServiceMap[T]) Add(obj database.Identifier, service T) {
	s.m.Store(obj.GetID(), service)
}

func (s ServiceMap[T]) Exists(obj database.Identifier) bool {
	_, ok := s.Get(obj)

	return ok
}

func (s ServiceMap[T]) Remove(ctx context.Context, obj database.Identifier) (retErr error) {
	s.m.Compute(obj.GetID(), func(service T, loaded bool) (T, xsync.ComputeOp) {
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

func (s ServiceMap[T]) Start(obj database.Identifier) (started bool, retErr error) {
	s.m.Compute(obj.GetID(), func(service T, loaded bool) (_ T, op xsync.ComputeOp) {
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

func (s ServiceMap[T]) Stop(ctx context.Context, obj database.Identifier) (stopped bool, retErr error) {
	s.m.Compute(obj.GetID(), func(service T, loaded bool) (T, xsync.ComputeOp) {
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

func (s ServiceMap[T]) Restart(ctx context.Context, obj database.Identifier) (retErr error) {
	s.m.Compute(obj.GetID(), func(service T, loaded bool) (T, xsync.ComputeOp) {
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

func (s *ServiceMap[T]) StopAll(ctx context.Context) error {
	errChan := make(chan error, s.m.Size())

	wg := sync.WaitGroup{}
	s.m.Range(func(_ int64, service T) bool {
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

func (s *ServiceMap[T]) Range(f func(int64, T) bool) { s.m.Range(f) }
