package backup

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	_ "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
)

var discard *log.Logger

func init() {
	_ = log.InitBackend("CRITICAL", "/dev/null", "")
	discard = log.NewLogger("discard")
}
