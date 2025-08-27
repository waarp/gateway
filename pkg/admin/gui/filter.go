package gui

import (
	"errors"
	"net/http"
)

//nolint:gochecknoglobals //global needed
var DocLink = "https://doc.waarp.org/waarp-gateway/latest/fr/"

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

type filtersCookieMap map[string]Filters

var (
	ErrNoToken       = errors.New("no token found")
	ErrLoadSession   = errors.New("failed to load session")
	ErrAccessSession = errors.New("failed access session")
)

func sessionFilter(r *http.Request) (*Session, string, error) {
	cookie, err := r.Cookie("token")
	if err != nil || cookie.Value == "" {
		return nil, "", ErrNoToken
	}

	value, ok := sessionStore.Load(cookie.Value)
	if !ok {
		return nil, "", ErrLoadSession
	}

	session, ok := value.(Session)
	if !ok {
		return nil, "", ErrAccessSession
	}

	return &session, cookie.Value, nil
}

func GetPageFilters(r *http.Request, page string) (Filters, bool) {
	if session, _, err := sessionFilter(r); err == nil && session.Filters != nil {
		if filter, ok := session.Filters[page]; ok {
			return filter, true
		}

		if value, ok := userFiltersStore.Load(session.UserID); ok {
			if cookieMap, hasValue := value.(filtersCookieMap); hasValue {
				if filter, ok2 := cookieMap[page]; ok2 {
					return filter, true
				}
			}
		}
	}

	return Filters{}, false
}

func PersistPageFilters(r *http.Request, page string, filter *Filters) {
	if session, token, err := sessionFilter(r); err == nil {
		if session.Filters == nil {
			session.Filters = make(filtersCookieMap)
		}
		session.Filters[page] = *filter
		sessionStore.Store(token, *session)

		if value, ok := userFiltersStore.Load(session.UserID); ok {
			if cookieMap, hasValue := value.(filtersCookieMap); hasValue {
				cookieMap[page] = *filter
				userFiltersStore.Store(session.UserID, cookieMap)
			} else {
				userFiltersStore.Store(session.UserID, filtersCookieMap{page: *filter})
			}
		} else {
			userFiltersStore.Store(session.UserID, filtersCookieMap{page: *filter})
		}
	}
}

func ClearPageFilters(r *http.Request, page string) {
	if session, token, err := sessionFilter(r); err == nil {
		if session.Filters != nil {
			delete(session.Filters, page)
		}

		sessionStore.Store(token, *session)

		if value, ok := userFiltersStore.Load(session.UserID); ok {
			if cookieMap, hasValue := value.(filtersCookieMap); hasValue {
				delete(cookieMap, page)
				userFiltersStore.Store(session.UserID, cookieMap)
			}
		}
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

	if filter.Offset == 0 {
		filter.DisablePrevious = true
	}

	if (filter.Offset+1)*filter.Limit >= lenList {
		filter.DisableNext = true
	}
}
