package database

import vers "code.waarp.fr/apps/gateway/gateway/pkg/version"

//nolint:gochecknoinits // init is used by design
func init() {
	AddInit(&version{})
}

type version struct {
	Current string `xorm:"notnull text 'current'"`
}

// TableName returns the name of the version table.
func (v *version) TableName() string { return "version" }

// Appellation returns the name of an element of the version table (not really
// relevant since the version table will always have only 1 row).
func (v *version) Appellation() string { return "version" }

// Init initializes the version table with the current program version.
func (v *version) Init(db Access) Error {
	if n, err := db.Count(&version{}).Run(); err != nil {
		return err
	} else if n != 0 {
		return nil
	}

	return db.Insert(&version{vers.Num}).Run()
}
