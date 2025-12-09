package ftp

import (
	"errors"
	"fmt"
	"net"
	"regexp"

	"code.waarp.fr/lib/goftp"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/pipeline"
)

//nolint:mnd //magic numbers are needed here for FTP return codes
func toPipelineError(ftpErr error, context string) *pipeline.Error {
	var pipErr *pipeline.Error
	if errors.As(ftpErr, &pipErr) && pipErr != nil {
		return pipErr
	}

	var netErr *net.OpError
	if errors.As(ftpErr, &netErr) {
		switch netErr.Op {
		case "dial":
			return pipeline.NewErrorWith(types.TeConnection,
				"could not connect to FTP server", netErr.Err)
		case "read", "write":
			return pipeline.NewErrorf(types.TeConnectionReset,
				"connection closed unexpectedly")
		default:
			return pipeline.NewErrorf(types.TeConnection,
				"%s: %s", context, netErr.Err)
		}
	}

	var goftpErr goftp.Error
	if !errors.As(ftpErr, &goftpErr) {
		return pipeline.NewErrorWith(types.TeConnection, "FTP error", ftpErr)
	}

	if errors.Is(goftpErr, goftp.ErrInvalidFileSize) {
		return pipeline.NewError(types.TeBadSize,
			"destination file size does not match the source file size")
	}

	reg := regexp.MustCompile(`TransferError\((?P<Code>Te\w+)\): (?P<Details>.*)$`)
	matches := reg.FindStringSubmatch(goftpErr.Message())

	//nolint:mnd //too specific
	if len(matches) == 3 {
		code := types.TecFromString(matches[1])
		details := matches[2]

		return pipeline.NewErrorf(code, "Error on remote partner: %s", details)
	}

	detail := goftpErr.Message()
	if detail == "" {
		detail = goftpErr.Error()
	}

	msg := fmt.Sprintf("%s: %s", context, detail)

	// see https://en.wikipedia.org/wiki/List_of_FTP_server_return_codes for a
	// list of FTP return codes
	//nolint:mnd //too specific
	switch goftpErr.Code() {
	case 331, 332, 430, 530, 532:
		return pipeline.NewError(types.TeBadAuthentication, msg)
	case 421:
		return pipeline.NewError(types.TeShuttingDown, msg)
	case 425:
		return pipeline.NewError(types.TeConnection, msg)
	case 426:
		return pipeline.NewError(types.TeConnectionReset, msg)
	case 450:
		return pipeline.NewError(types.TeFileNotFound, msg)
	case 452:
		return pipeline.NewError(types.TeBadSize, msg)
	case 502, 504:
		return pipeline.NewError(types.TeUnimplemented, msg)
	case 553:
		return pipeline.NewError(types.TeForbidden, msg)
	default:
		return pipeline.NewError(types.TeUnknown, msg)
	}
}
