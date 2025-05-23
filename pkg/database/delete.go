package database

import "fmt"

// DeleteBean is the interface that a model must implement in order to be
// deletable via the Delete query builder.
type DeleteBean interface {
	Table
	Identifier
}

// DeleteQuery is the type representing a SQL DELETE statement with an ID
// condition (so for a single entry). The ID is taken from the given model.
type DeleteQuery struct {
	db   Access
	bean DeleteBean
}

func (d *DeleteQuery) run(s *Session) error {
	if hook, ok := d.bean.(DeletionHook); ok {
		if err := hook.BeforeDelete(s); err != nil {
			s.logger.Error("%s deletion hook failed: %s", d.bean.Appellation(), err)

			return fmt.Errorf("%s deletion hook failed: %w", d.bean.Appellation(), err)
		}
	}

	query := s.session.NoAutoCondition().Table(d.bean.TableName()).
		Where("id=?", d.bean.GetID())

	if _, err := query.Delete(d.bean); err != nil {
		s.logger.Error("Failed to delete the %s entry: %s", d.bean.Appellation(), err)

		return NewInternalError(err)
	}

	return nil
}

// Run executes the 'DELETE' query.
func (d *DeleteQuery) Run() error {
	if err := checkExists(d.db, d.bean); err != nil {
		return err
	}

	switch db := d.db.(type) {
	case *Standalone:
		return db.Transaction(d.run)
	case *Session:
		return d.run(db)
	default:
		panic("unknown database accessor type")
	}
}
