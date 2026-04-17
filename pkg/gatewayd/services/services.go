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
	Clients = newServiceMap[Client]()
	Servers = newServiceMap[Server]()
)

type (
	Service = protocol.StartStopper
	Client  = protocol.Client
	Server  = protocol.Server
)

type serviceList []Service

func (s *serviceList) Add(service Service) { *s = append(*s, service) }

type serviceMap[T Service] struct {
	m *xsync.Map[int64, T]
}

func newServiceMap[T Service]() serviceMap[T] {
	return serviceMap[T]{m: xsync.NewMap[int64, T]()}
}

func (s serviceMap[T]) Get(obj database.Identifier) (T, bool) {
	return s.Load(obj)
}

func (s serviceMap[T]) Load(obj database.Identifier) (T, bool) {
	return s.m.Load(obj.GetID())
}

func (s serviceMap[T]) Add(obj database.Identifier, service T) {
	s.m.Store(obj.GetID(), service)
}

func (s serviceMap[T]) Exists(obj database.Identifier) bool {
	_, ok := s.m.Load(obj.GetID())

	return ok
}

func (s serviceMap[T]) Remove(obj database.Identifier) {
	s.m.Delete(obj.GetID())
}

func (s serviceMap[T]) Start(obj database.Identifier) (retErr error) {
	s.m.Compute(obj.GetID(), func(service T, loaded bool) (_ T, op xsync.ComputeOp) {
		if !loaded {
			return service, xsync.CancelOp
		} else if state, _ := service.State(); state == utils.StateRunning {
			retErr = fmt.Errorf("%w: %q", utils.ErrAlreadyRunning, service.Name())

			return service, xsync.CancelOp
		}

		if err := service.Start(); err != nil {
			retErr = fmt.Errorf("failed to start service: %w", err)
		}

		return service, xsync.UpdateOp
	})

	return retErr
}

func (serviceMap[T]) stop(ctx context.Context, service T, loaded, remove bool,
) (xsync.ComputeOp, error) {
	if !loaded {
		return xsync.CancelOp, nil
	} else if state, _ := service.State(); state != utils.StateRunning {
		if remove {
			return xsync.DeleteOp, nil
		}

		return xsync.CancelOp, nil
	}

	if err := service.Stop(ctx); err != nil {
		return xsync.UpdateOp, fmt.Errorf("failed to stop service: %w", err)
	}

	if remove {
		return xsync.DeleteOp, nil
	}

	return xsync.UpdateOp, nil
}

func (s serviceMap[T]) Stop(ctx context.Context, obj database.Identifier, remove bool) (retErr error) {
	s.m.Compute(obj.GetID(), func(service T, loaded bool) (_ T, op xsync.ComputeOp) {
		op, retErr = s.stop(ctx, service, loaded, remove)

		return service, op
	})

	return retErr
}

func (s serviceMap[T]) Restart(ctx context.Context, obj database.Identifier, newService T) (retErr error) {
	s.m.Compute(obj.GetID(), func(service T, loaded bool) (_ T, op xsync.ComputeOp) {
		if op, retErr = s.stop(ctx, service, loaded, false); retErr != nil {
			return service, op
		}

		if err := newService.Start(); err != nil {
			retErr = fmt.Errorf("failed to start service: %w", err)
		}

		return newService, xsync.UpdateOp
	})

	return retErr
}

func (s *serviceMap[T]) StopAll(ctx context.Context) error {
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

func (s *serviceMap[T]) Range(f func(int64, T) bool) { s.m.Range(f) }
