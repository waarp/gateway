package backup

import (
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"
	"code.waarp.fr/waarp-gateway/waarp-gateway/pkg/log"
	_ "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/tasks"
)

var discard *log.Logger

func init() {
	logConf := conf.LogConfig{LogTo: "/dev/null"}
	_ = log.InitBackend(logConf)
	discard = log.NewLogger("discard")
}
