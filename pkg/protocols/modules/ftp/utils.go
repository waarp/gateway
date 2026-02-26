package ftp

import (
	"context"
	"fmt"
	"math/rand/v2"
	"net"

	"code.waarp.fr/lib/goftp"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

func getPortInRange(addr string, minPort, maxPort uint16) (uint16, *pipeline.Error) {
	if minPort == maxPort {
		return minPort, nil
	} else if minPort > maxPort {
		return 0, pipeline.NewError(types.TeInternal, "invalid port range for active mode")
	}

	const (
		minNbTries = 10
		maxNbTries = 100
	)

	rangeSize := int(maxPort - minPort)
	nbTries := rangeSize
	nbTries = min(nbTries, maxNbTries)
	nbTries = max(nbTries, minNbTries)

	for i := range nbTries {
		candidate := minPort + uint16(rand.IntN(rangeSize))

		if list, err := net.Listen("tcp", fmt.Sprintf("%s:%d", addr, i)); err == nil {
			_ = list.Close() //nolint:errcheck //error is irrelevant

			return candidate, nil
		}
	}

	return 0, pipeline.NewError(types.TeInternal,
		"could not find an available port in the range for active mode")
}

func deleteRemoteCtx(ctx context.Context, client *goftp.Client, path string, recursive bool) error {
	result := make(chan error)
	go func() {
		defer close(result)
		result <- deleteRemote(client, path, recursive)
	}()

	select {
	case err := <-result:
		return err
	case <-ctx.Done():
		return fmt.Errorf("request canceled: %w", ctx.Err())
	}
}

func deleteRemote(client *goftp.Client, path string, recursive bool) error {
	if !recursive {
		if err := client.Delete(path); err != nil {
			return fmt.Errorf("failed to delete file %q: %w", path, err)
		}

		return nil
	}

	info, err := client.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file %q: %w", path, err)
	}

	return deleteRemoteRec(client, info)
}

func deleteRemoteRec(client *goftp.Client, info fs.FileInfo) error {
	path := info.Name()

	if info.IsDir() {
		children, rdErr := client.ReadDir(path)
		if rdErr != nil {
			return fmt.Errorf("failed to read directory %q: %w", path, rdErr)
		}

		for _, child := range children {
			if err := deleteRemoteRec(client, child); err != nil {
				return err
			}
		}
	}

	if err := client.Delete(path); err != nil {
		return fmt.Errorf("failed to delete file %q: %w", path, err)
	}

	return nil
}
