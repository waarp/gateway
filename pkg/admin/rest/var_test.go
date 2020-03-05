package rest

import "code.waarp.fr/waarp-gateway/waarp-gateway/pkg/conf"

var logConf = conf.LogConfig{
	Level: "DEBUG",
	LogTo: "stdout",
}
