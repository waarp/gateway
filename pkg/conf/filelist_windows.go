//+build windows

package conf

import (
	"fmt"
	"os"
)

func getDefaultConfFiles() []string {
	rv := []string{
		"gatewayd.ini",
		"etc\\gatewayd.ini",
	}

	pd := os.Getenv("ProgramData")
	if pd != "" {
		fileList = append(rv, fmt.Sprintf("%s\\waarp-gateway\\gatewayd.ini", pd))
	}

	return rv
}
