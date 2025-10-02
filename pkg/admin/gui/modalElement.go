package gui

import (
	"net/http"
	"strings"
)

func getFormValues(r *http.Request) map[string]any {
	if err := r.ParseForm(); err != nil {
		return nil
	}
	values := make(map[string]any)

	for key, val := range r.Form {
		if strings.HasSuffix(key, "[]") {
			values[key] = val
		} else if len(val) > 0 {
			values[key] = val[0]
		}
	}

	return values
}
