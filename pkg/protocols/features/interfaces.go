package features

import "context"

// DeletionInterface provides a method for deleting a single file. It should be
// implemented by the protocol's TransferClient.
type DeletionInterface interface {
	Delete(ctx context.Context, path string) error
}

// RecursiveDeletionInterface provides a method for recursively deleting all files
// within a directory. It should be implemented by the protocol's TransferClient.
type RecursiveDeletionInterface interface {
	DeleteAll(ctx context.Context, path string) error
}
