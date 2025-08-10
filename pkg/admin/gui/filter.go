package gui

import "net/http"

type Filters struct {
	Offset            uint64
	Limit             uint64
	OrderAsc          bool
	Permissions       string
	PermissionsType   string
	PermissionsValue  string
	DisableNext       bool
	DisablePrevious   bool
	OrderBy           string
	Protocols         Protocols
	Status            Status
	FilterRuleSend    string
	FilterRuleReceive string
	DateStart         string
	DateEnd           string
	FilterFilePattern string
	FilterAgent       string
	FilterAccount     string
}

const DefaultLimitPagination = 30

func paginationPage(filter *Filters, lenList uint64, r *http.Request) {
	if r.URL.Query().Get("previous") == "true" && filter.Offset > 0 {
		filter.Offset--
	}

	if r.URL.Query().Get("next") == "true" {
		if filter.Limit*(filter.Offset+1) <= lenList {
			filter.Offset++
		}
	}

	if filter.Offset == 0 {
		filter.DisablePrevious = true
	}

	if (filter.Offset+1)*filter.Limit >= lenList {
		filter.DisableNext = true
	}
}
