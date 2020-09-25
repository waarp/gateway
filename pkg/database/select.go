package database

import (
	"fmt"
	"reflect"
)

type selectBean interface {
	table
	appellation
}

type selectQuery struct {
	bean     selectBean
	lim, off int
	cond     Condition
	order    string
	desc     bool
}

// Select creates a query to retrieve a set of the given `bean` parameter from
// the database according to the conditions appended to the query.
//
//nolint:golint //This exported function returns an unexported type on purpose
//so that instances of selectQuery cannot be created outside of this function
func Select(bean selectBean) *selectQuery {
	return &selectQuery{bean: bean}
}

func (s *selectQuery) Where(cond Condition) *selectQuery {
	s.cond = cond
	return s
}

func (s *selectQuery) OrderBy(order string, desc bool) *selectQuery {
	s.order = order
	s.desc = desc
	return s
}

func (s *selectQuery) Limit(lim, off int) *selectQuery {
	s.lim = lim
	s.off = off
	return s
}

func (s *selectQuery) query(ses *Session) (Iterator, error) {
	if s.bean == nil {
		ses.logger.Error("'SELECT' called with a `nil` target")
		return nil, ErrNilRecord
	}
	if s.cond != nil {
		ses.session.Where(s.cond.convert())
	}
	if s.lim != 0 || s.off != 0 {
		ses.session.Limit(s.lim, s.off)
	}
	if s.order != "" {
		ses.session.OrderBy(fmt.Sprintf("%s ASC", s.order))
		if s.desc {
			ses.session.OrderBy(fmt.Sprintf("%s DESC", s.order))
		}
	}

	bean := reflect.New(reflect.TypeOf(s.bean).Elem()).Interface().(selectBean)
	ses.logger.Criticalf("BEAN: %#v", bean)
	rows, err := ses.session.Rows(bean)
	if err != nil {
		ses.logger.Errorf("Failed to retrieve the %s entries: %s",
			s.bean.Appellation(), err)
		return nil, NewInternalError(err, "failed to retrieve the %s entries",
			s.bean.Appellation())
	}

	return rows, nil
}
