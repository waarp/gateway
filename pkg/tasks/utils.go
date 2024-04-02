package tasks

import (
	"fmt"
	"strings"
)

func mapToStr(m map[string]string) string {
	args := make([]string, 0, len(m))
	for k, v := range m {
		args = append(args, fmt.Sprintf("%s=%s", k, v))
	}

	return "{" + strings.Join(args, ", ") + "}"
}
