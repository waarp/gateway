package database

import "database/sql"

type insertBean interface {
	table
	appellation
	writeHook
}

type insertQuery struct {
	bean insertBean
}

// Insert creates a query to insert the given `bean` parameter in the database.
//
//nolint:golint //This exported function returns an unexported type on purpose
//so that instances of insertQuery cannot be created outside of this function
func Insert(bean insertBean) *insertQuery {
	return &insertQuery{bean: bean}
}

func (i *insertQuery) exec(ses *Session) (sql.Result, error) {
	if i.bean == nil {
		ses.logger.Error("'INSERT' called with a `nil` target")
		return nil, ErrNilRecord
	}
	if err := i.bean.BeforeWrite(ses); err != nil {
		ses.logger.Errorf("%s entry INSERT validation failed: %s",
			i.bean.Appellation(), err)
		return nil, err
	}

	n, err := ses.session.Table(i.bean.Table()).InsertOne(i.bean)
	if err != nil {
		ses.logger.Errorf("Failed to insert the new %s entry: %s",
			i.bean.Appellation(), err)
		return nil, NewInternalError(err, "failed to insert the new %s entry",
			i.bean.Appellation())
	}

	return &dbResult{affected: n}, nil
}
