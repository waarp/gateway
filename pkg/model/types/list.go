package types

import (
	"strings"
)

type List []string

func (a *List) IsSet() bool               { return len(*a) > 0 }
func (a *List) FromDB(bytes []byte) error { return a.Set(string(bytes)) }
func (a *List) ToDB() ([]byte, error)     { return []byte(a.String()), nil }

func (a *List) String() string {
	if !a.IsSet() {
		return ""
	}

	return strings.Join(*a, ", ")
}

func (a *List) Set(list string) error {
	if list == "" {
		return nil
	}

	*a = strings.Split(list, ",")
	for i, elem := range *a {
		(*a)[i] = strings.TrimSpace(elem)
	}

	return nil
}
