package wg

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/rest/api"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

var errBadPerm = errors.New("permissions are incorrect")

const (
	roleClient    = "client"
	roleServer    = "server"
	directionRecv = "receive"
	directionSend = "send"
	sizeUnknown   = "unknown"
)

func unmarshalBody(body io.Reader, object any) error {
	b, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if err := json.Unmarshal(b, object); err != nil {
		return fmt.Errorf("invalid JSON response object: %w", err)
	}

	return nil
}

func getResponseErrorMessage(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read the response body: %w", err)
	}

	return errors.New(strings.TrimSpace(string(body))) //nolint:goerr113 // too specific
}

func displayResponseMessage(w io.Writer, resp *http.Response) error {
	cont, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read the response body: %w", err)
	}

	if len(cont) > 0 {
		fmt.Fprintln(w, string(cont))
	}

	return nil
}

func isNotUpdate(obj any) bool {
	value := reflect.ValueOf(obj)

	switch value.Kind() {
	case reflect.Pointer:
		return isNotUpdate(value.Elem().Interface())
	case reflect.Map:
		return value.Len() == 0
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			if !value.Field(i).IsZero() {
				return false
			}
		}

		return true
	default:
		panic("JSON objects must be either a struct or a map (or a pointer to either)")
	}
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
	case 'A':
		return &perm.Administration
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
			return nil, fmt.Errorf("misssing permission operator after '%s': %w", grp, errBadPerm)
		}

		dest := getPermTarget(rune(grp[0]), &perms)
		if dest == nil {
			return nil, fmt.Errorf("invalid permission target '%c': %w", grp[0], errBadPerm)
		}

		modes := grp[1:]
		for _, m := range modes {
			if !isPermOp(m) && !isPermMode(m) {
				return nil, fmt.Errorf("invalid permission mode '%s': %w", modes, errBadPerm)
			}
		}

		*dest += modes
	}

	return &perms, nil
}

type ListOptions struct {
	Limit  uint `short:"l" long:"limit" description:"Max number of returned entries" default:"20"`
	Offset uint `short:"o" long:"offset" description:"Index of the first returned entry" default:"0"`
}

func agentListURL(path string, s *ListOptions, sort string, protos []string) {
	addr.Path = path
	query := url.Values{}
	query.Set("limit", utils.FormatUint(s.Limit))
	query.Set("offset", utils.FormatUint(s.Offset))
	query.Set("sort", sort)

	for _, proto := range protos {
		query.Add("protocol", proto)
	}

	addr.RawQuery = query.Encode()
}

func listURL(s *ListOptions, sort string) {
	query := url.Values{}
	query.Set("limit", utils.FormatUint(s.Limit))
	query.Set("offset", utils.FormatUint(s.Offset))
	query.Set("sort", sort)
	addr.RawQuery = query.Encode()
}
