package database

import "database/sql"

type deleteBean interface {
	table
	appellation
	deletionHook
}

type deleteQuery struct {
	bean deleteBean
}

// Delete creates a query to delete the given `bean` parameter in the database
// (using the struct fields as filters for the deletion).
//
//nolint:golint //This exported function returns an unexported type on purpose
//so that instances of deleteQuery cannot be created outside of this function
func Delete(bean deleteBean) *deleteQuery {
	return &deleteQuery{bean: bean}
}

func (d *deleteQuery) exec(ses *Session) (sql.Result, error) {
	if d.bean == nil {
		ses.logger.Error("'DELETE' called with a `nil` target")
		return nil, ErrNilRecord
	}

	if err := d.bean.BeforeDelete(ses); err != nil {
		return nil, err
	}

	n, err := ses.session.Table(d.bean.Table()).Delete(d.bean)
	if err != nil {
		ses.logger.Errorf("Failed to delete the %s entry: %s",
			d.bean.Appellation(), err)
		return nil, NewInternalError(err, "failed to delete the %s entry",
			d.bean.Appellation())
	}

	return &dbResult{affected: n}, nil
}
