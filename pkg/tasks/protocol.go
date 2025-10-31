package tasks

import "context"

type RemoteDeleter interface {
	Delete(ctx context.Context, path string, recursive bool) error
}
