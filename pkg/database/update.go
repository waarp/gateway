package database

import (
	"database/sql"
)

type updateBean interface {
	table
	appellation
	identifier
	writeHook
}

type updateQuery struct {
	bean updateBean
}

// Update creates a query to update the given `bean` parameter in the database.
// The entry's ID will be used as condition for the update.
//
//nolint:golint //This exported function returns an unexported type on purpose
//so that instances of updateQuery cannot be created outside of this function
func Update(bean updateBean) *updateQuery {
	return &updateQuery{bean: bean}
}

func (u *updateQuery) exec(ses *Session) (sql.Result, error) {
	if u.bean == nil {
		ses.logger.Error("'UPDATE' called with a `nil` target")
		return nil, ErrNilRecord
	}

	res, err := ses.Count(Select(u.bean).Where(Equal("id", u.bean.GetID())))
	if err != nil {
		ses.logger.Errorf("Failed to check if the %s to update exists: %s",
			u.bean.Appellation(), err)
		return nil, NewInternalError(err, "failed to check if the %s to update exists",
			u.bean.Appellation())
	}
	if res < 1 {
		ses.logger.Errorf("No %s found with ID %d", u.bean.Appellation(), u.bean.GetID())
		return nil, newNotFoundError(u.bean.Appellation())
	}

	if err := u.bean.BeforeWrite(ses); err != nil {
		ses.logger.Errorf("%s entry UPDATE validation failed: %s",
			u.bean.Appellation(), err)
		return nil, err
	}

	n, err := ses.session.Table(u.bean.Table()).ID(u.bean.GetID()).AllCols().Update(u.bean)
	if err != nil {
		ses.logger.Errorf("Failed to update the %s entry: %s",
			u.bean.Appellation(), err)
		return nil, NewInternalError(err, "failed to update the %s entry",
			u.bean.Appellation())
	}

	return &dbResult{affected: n}, nil
}
