package tasks

import (
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"code.waarp.fr/apps/gateway/gateway/pkg/fs"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/modeltest"
)

const testProtocol = "test_proto"

//nolint:gochecknoinits // init is used to ease the tests
func init() {
	modeltest.AddDummyProtoConfig(testProtocol)

	fs.FilePerms = 0o644
	fs.DirPerms = 0o755
}

func isWindowsRuntime() bool {
	return runtime.GOOS == "windows"
}

func shellCmd() string {
	if isWindowsRuntime() {
		return "cmd.exe"
	}

	return "sh"
}

func shellCmdArgs(args string) string {
	if isWindowsRuntime() {
		args = strings.ReplaceAll(args, ";", "&")

		return "/C " + args
	}

	return "-c " + args
}

func endl() string {
	if isWindowsRuntime() {
		return "\r\n"
	}

	return "\n"
}

func TestISOTSFormat(t *testing.T) {
	ts := time.Now()
	str := formatTime(isoTSFormat, ts)

	assert.Equal(t, ts.Format(time.RFC3339), str)
}
