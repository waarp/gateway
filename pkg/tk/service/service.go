package service

import "context"

type Service interface {
	Start() error
	Stop(ctx context.Context) error
}
