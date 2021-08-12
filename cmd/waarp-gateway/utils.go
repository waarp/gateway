package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
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

func getPermTarget(c rune, perm *api.Perms) *string {
	switch c {
	case 'T':
		return &perm.Transfers
	case 'S':
		return &perm.Servers
	case 'P':
		return &perm.Partners
	case 'R':
		return &perm.Rules
	case 'U':
		return &perm.Users
	default:
		return nil
	}
}

func isPermOp(c rune) bool {
	return c == '=' || c == '+' || c == '-'
}

func isPermMode(c rune) bool {
	return c == 'r' || c == 'w' || c == 'd'
}

func parsePerms(str string) (*api.Perms, error) {
	var perms api.Perms

	groups := strings.Split(str, ",")

	for _, grp := range groups {
		if len(grp) == 0 {
			continue
		}
		if len(grp) == 1 {
			return nil, fmt.Errorf("misssing permission operator after '%s'", grp)
		}

		dest := getPermTarget(rune(grp[0]), &perms)
		if dest == nil {
			return nil, fmt.Errorf("invalid permission target '%c'", grp[0])
		}

		modes := grp[1:]
		for _, m := range modes {
			if !isPermOp(m) && !isPermMode(m) {
				return nil, fmt.Errorf("invalid permission mode '%s'", modes)
			}
		}
		*dest += modes
	}
	return &perms, nil
}
