package gui

import (
	"net/http"

	"github.com/puzpuzpuz/xsync/v4"

	"code.waarp.fr/apps/gateway/gateway/pkg/admin/gui/v2/backend/common"
)

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

const (
	DefaultLimitPagination = 30
	True                   = "true"
	False                  = "false"
)

type filtersCookieMap map[string]*Filters

//nolint:gochecknoglobals //filters map
var userFiltersStore = xsync.NewMap[int64, filtersCookieMap]()

func GetPageFilters(r *http.Request, page string) (*Filters, bool) {
	user := common.GetUser(r)

	if cookieMap, ok := userFiltersStore.Load(user.ID); ok {
		if filter, ok2 := cookieMap[page]; ok2 {
			return filter, true
		}
	}

	return nil, false
}

func PersistPageFilters(r *http.Request, page string, filter *Filters) {
	user := common.GetUser(r)

	if cookieMap, ok := userFiltersStore.Load(user.ID); ok {
		cookieMap[page] = filter
		userFiltersStore.Store(user.ID, cookieMap)
	} else {
		userFiltersStore.Store(user.ID, filtersCookieMap{page: filter})
	}
}

func ClearPageFilters(r *http.Request, page string) {
	user := common.GetUser(r)

	if cookieMap, ok := userFiltersStore.Load(user.ID); ok {
		delete(cookieMap, page)
		userFiltersStore.Store(user.ID, cookieMap)
	}
}

func paginationPage(filter *Filters, lenList uint64, r *http.Request) {
	if r.URL.Query().Get("previous") == True && filter.Offset > 0 {
		filter.Offset--
	}

	if r.URL.Query().Get("next") == True {
		if filter.Limit*(filter.Offset+1) <= lenList {
			filter.Offset++
		}
	}

	filter.DisablePrevious = filter.Offset == 0
	filter.DisableNext = (filter.Offset+1)*filter.Limit >= lenList
}
