package database

import (
	"fmt"
)

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

	all bool
}

func (d *DeleteQuery) All() *DeleteQuery {
	d.all = true
	return d
}

func (d *DeleteQuery) run(s *Session) error {
	logger := s.getLogger()
	engine := s.getUnderlying()

	addOwnerCond(engine, d.all, d.bean, s.getOwner())

	if hook, ok := d.bean.(DeletionHook); ok {
		if err := hook.BeforeDelete(s); err != nil {
			logger.Errorf("%s deletion hook failed: %v", d.bean.Appellation(), err)

			return fmt.Errorf("%s deletion hook failed: %w", d.bean.Appellation(), err)
		}
	}

	query := engine.Table(d.bean.TableName()).Where("id=?", d.bean.GetID())
	if err := query.Delete(d.bean).Error; err != nil {
		logger.Errorf("Failed to delete the %s entry: %v", d.bean.Appellation(), err)

		return NewInternalError(err)
	}

	if hook, ok := d.bean.(DeletionCallback); ok {
		if err := hook.AfterDelete(s); err != nil {
			logger.Errorf("%s deletion callback failed: %v", d.bean.Appellation(), err)

			return fmt.Errorf("%s deletion callback failed: %w", d.bean.Appellation(), err)
		}
	}

	return nil
}

// Run executes the 'DELETE' query.
func (d *DeleteQuery) Run() error {
	if err := checkExists(d.db, d.bean); err != nil {
		return err
	}

	return d.db.Transaction(d.run)
}
