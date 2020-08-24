//+build windows

package conf

import (
	"fmt"
	"os"
)

var fileList = []string{
	"gatewayd.ini",
	"etc\\gatewayd.ini",
}

func init() {
	pd := os.Getenv("ProgramData")
	if pd != "" {
		fileList = append(fileList, fmt.Sprintf("%s\\waarp-gateway\\gatewayd.ini", pd))
	}
}
