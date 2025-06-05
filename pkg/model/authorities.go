package model

import (
	"fmt"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/authentication"
)

// Authority is the model representing an authentication authority. An authentication
// authority is a trusted third party used to authenticate transfer partners.
type Authority struct {
	ID             int64    `xorm:"<- id AUTOINCR"`
	Name           string   `xorm:"name"`
	Type           string   `xorm:"type"`
	PublicIdentity string   `xorm:"public_identity"`
	ValidHosts     []string `xorm:"-"`
}

func (*Authority) TableName() string   { return TableAuthorities }
func (*Authority) Appellation() string { return NameAuthority }
func (a *Authority) GetID() int64      { return a.ID }

func (a *Authority) BeforeWrite(db database.Access) error {
	if strings.TrimSpace(a.Name) == "" {
		return database.NewValidationError("the authority is missing a name")
	}

	if strings.TrimSpace(a.Type) == "" {
		return database.NewValidationError("the authority is missing a type")
	}

	if strings.TrimSpace(a.PublicIdentity) == "" {
		return database.NewValidationError("the authority is missing a public identity value")
	}

	validator := authentication.GetAuthorityHandler(a.Type)
	if validator == nil {
		return database.NewValidationErrorf("%q is not a valid authority type", a.Type)
	}

	if err := validator.Validate(a.PublicIdentity); err != nil {
		return database.NewValidationErrorf(
			"could not validate the authority's public identity value: %w", err)
	}

	if n, err := db.Count(a).Where("id<>? AND name=?", a.ID, a.Name).Run(); err != nil {
		return fmt.Errorf("failed to check for duplicate authorities: %w", err)
	} else if n != 0 {
		return database.NewValidationErrorf("an %s named %q already exists",
			a.Appellation(), a.Name)
	}

	return nil
}

func (a *Authority) AfterUpdate(db database.Access) error {
	if err := db.Exec(fmt.Sprintf("DELETE FROM %s WHERE authority_id=?",
		TableAuthHosts), a.ID); err != nil {
		return fmt.Errorf("failed to delete the authority's valid hosts: %w", err)
	}

	return a.AfterInsert(db)
}

func (a *Authority) AfterInsert(db database.Access) error {
	for _, host := range a.ValidHosts {
		if host = strings.TrimSpace(host); host == "" {
			continue
		}

		if err := db.Insert(&Host{AuthorityID: a.ID, Host: host}).
			Run(); err != nil {
			return fmt.Errorf("failed to insert the authority's valid host %q: %w", host, err)
		}
	}

	return nil
}

func (a *Authority) AfterRead(db database.ReadAccess) error {
	var hosts Hosts
	if err := db.Select(&hosts).Where("authority_id=?", a.ID).
		OrderBy("host", true).Run(); err != nil {
		return fmt.Errorf("failed to retrieve the authority's valid hosts: %w", err)
	}

	a.ValidHosts = make([]string, 0, len(hosts))

	for i := range hosts {
		a.ValidHosts = append(a.ValidHosts, hosts[i].Host)
	}

	return nil
}

type Host struct {
	AuthorityID int64  `xorm:"authority_id"`
	Host        string `xorm:"host"`
}

func (*Host) TableName() string   { return TableAuthHosts }
func (*Host) Appellation() string { return NameAuthHost }
