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
	"code.waarp.fr/apps/gateway/gateway/pkg/tk/utils"
)

var errBadPerm = errors.New("permissions are insorrect")

const (
	roleClient    = "client"
	roleServer    = "server"
	directionRecv = "receive"
	directionSend = "send"
	sizeUnknown   = "unknown"
)

func unmarshalBody(body io.Reader, object interface{}) error {
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

func displayResponseMessage(resp *http.Response) error {
	cont, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read the response body: %w", err)
	}

	if len(cont) > 0 {
		fmt.Fprintln(getColorable(), string(cont))
	}

	return nil
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

func dirToBoolPtr(dir string) *bool {
	switch dir {
	case directionSend:
		return utils.TruePtr
	case directionRecv:
		return utils.FalsePtr
	default:
		return nil
	}
}

type ListOptions struct {
	Limit  int `short:"l" long:"limit" description:"Max number of returned entries" default:"20"`
	Offset int `short:"o" long:"offset" description:"Index of the first returned entry" default:"0"`
}

func agentListURL(path string, s *ListOptions, sort string, protos []string) {
	addr.Path = path
	query := url.Values{}
	query.Set("limit", fmt.Sprint(s.Limit))
	query.Set("offset", fmt.Sprint(s.Offset))
	query.Set("sort", sort)

	for _, proto := range protos {
		query.Add("protocol", proto)
	}

	addr.RawQuery = query.Encode()
}

func listURL(s *ListOptions, sort string) {
	query := url.Values{}
	query.Set("limit", fmt.Sprint(s.Limit))
	query.Set("offset", fmt.Sprint(s.Offset))
	query.Set("sort", sort)
	addr.RawQuery = query.Encode()
}

func stringMapToAnyMap(input map[string]string) (map[string]any, error) {
	output := map[string]any{}

	for key, strVal := range input {
		if !json.Valid([]byte(strVal)) {
			strVal = fmt.Sprintf(`"%s"`, strVal)
		}

		var val any

		if err := json.Unmarshal([]byte(strVal), &val); err != nil {
			return nil, fmt.Errorf("cannot parse value '%s': %w", strVal, err)
		}

		output[key] = val
	}

	return output, nil
}
