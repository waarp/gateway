package common

import (
	"encoding/json"
	"net/http"
)

func ReadBody[T any](r *http.Request) (T, error) {
	var body T
	decoder := json.NewDecoder(r.Body)

	if err := decoder.Decode(&body); err != nil {
		return body, NewErrorWith(http.StatusBadRequest, "failed to parse JSON", err)
	}

	return body, nil
}
