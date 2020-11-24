package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"
)

func unmarshalBody(body io.Reader, object interface{}) error {
	b, err := ioutil.ReadAll(body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %s", err.Error())
	}

	if err := json.Unmarshal(b, object); err != nil {
		return fmt.Errorf("invalid JSON response object: %s", err.Error())
	}
	return nil
}

func getResponseMessage(resp *http.Response) error {
	body, _ := ioutil.ReadAll(resp.Body)
	return fmt.Errorf(strings.TrimSpace(string(body)))
}

func isNotUpdate(obj interface{}) bool {
	val := reflect.ValueOf(obj).Elem()
	for i := 0; i < val.NumField(); i++ {
		if !val.Field(i).IsZero() {
			return false
		}
	}
	return true
}
