package model

import (
	"fmt"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

const (
	defaultEbicsContractRefreshInterval = 24 * time.Hour

	DefaultEbicsContractRefreshIntervalSeconds int64 = int64(defaultEbicsContractRefreshInterval / time.Second)

	ebicsContractRefreshPolicyStatusReady    = "READY"
	ebicsContractRefreshPolicyStatusRunning  = "RUNNING"
	ebicsContractRefreshPolicyStatusError    = "ERROR"
	ebicsContractRefreshPolicyStatusDisabled = "DISABLED"
)

// EbicsContractRefreshPolicy stores one administrable scheduled contract refresh target.
type EbicsContractRefreshPolicy struct {
	ID    int64  `xorm:"<- id AUTOINCR"`
	Owner string `xorm:"owner"`
	Name  string `xorm:"name"`

	Enabled           bool      `xorm:"enabled"`
	ClientID          int64     `xorm:"client_id"`
	EbicsSubscriberID int64     `xorm:"ebics_subscriber_id"`
	IncludeHEV        bool      `xorm:"include_hev"`
	IntervalSeconds   int64     `xorm:"interval_seconds"`
	Status            string    `xorm:"status"`
	NextRunAt         time.Time `xorm:"next_run_at DATETIME(6) UTC"`
	LastAttemptAt     time.Time `xorm:"last_attempt_at DATETIME(6) UTC"`
	LastSuccessAt     time.Time `xorm:"last_success_at DATETIME(6) UTC"`
	LastError         string    `xorm:"last_error"`
	CreatedAt         time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt         time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}

func (*EbicsContractRefreshPolicy) TableName() string   { return TableEbicsContractRefreshPolicies }
func (*EbicsContractRefreshPolicy) Appellation() string { return NameEbicsContractRefreshPolicy }
func (p *EbicsContractRefreshPolicy) GetID() int64      { return p.ID }

func (p *EbicsContractRefreshPolicy) BeforeWrite(db database.Access) error {
	p.Owner = conf.GlobalConfig.GatewayName
	p.Name = strings.TrimSpace(p.Name)
	p.Status = strings.ToUpper(strings.TrimSpace(p.Status))
	p.LastError = strings.TrimSpace(p.LastError)

	if p.Name == "" {
		return database.NewValidationError("the EBICS contract refresh policy name is missing")
	}

	if p.ClientID == 0 {
		return database.NewValidationError("the EBICS contract refresh policy client reference is missing")
	}

	if p.EbicsSubscriberID == 0 {
		return database.NewValidationError("the EBICS contract refresh policy subscriber reference is missing")
	}

	if p.IntervalSeconds == 0 {
		p.IntervalSeconds = DefaultEbicsContractRefreshIntervalSeconds
	}

	if p.IntervalSeconds < 0 {
		return database.NewValidationError("the EBICS contract refresh interval cannot be negative")
	}

	if p.IntervalSeconds == 0 {
		return database.NewValidationError("the EBICS contract refresh interval is missing")
	}

	if p.Enabled {
		if p.Status == "" || p.Status == ebicsContractRefreshPolicyStatusDisabled {
			p.Status = ebicsContractRefreshPolicyStatusReady
		}
		if p.NextRunAt.IsZero() {
			p.NextRunAt = time.Now().UTC()
		}
	} else if p.Status == "" {
		p.Status = ebicsContractRefreshPolicyStatusDisabled
	}

	if err := validateEbicsContractRefreshPolicyStatus(p.Status); err != nil {
		return err
	}

	if err := validateEbicsContractRefreshPolicyRefs(db, p); err != nil {
		return err
	}

	n, err := db.Count(p).Where("id<>? AND owner=? AND name=?", p.ID, p.Owner, p.Name).Run()
	if err != nil {
		return fmt.Errorf("failed to check duplicate EBICS contract refresh policies: %w", err)
	}
	if n != 0 {
		return database.NewValidationErrorf("an EBICS contract refresh policy named %q already exists", p.Name)
	}

	return nil
}

func validateEbicsContractRefreshPolicyStatus(status string) error {
	switch status {
	case ebicsContractRefreshPolicyStatusReady,
		ebicsContractRefreshPolicyStatusRunning,
		ebicsContractRefreshPolicyStatusError,
		ebicsContractRefreshPolicyStatusDisabled:
		return nil
	case "":
		return database.NewValidationError("the EBICS contract refresh policy status is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS contract refresh policy status", status)
	}
}

func validateEbicsContractRefreshPolicyRefs(db database.Access, policy *EbicsContractRefreshPolicy) error {
	var client Client
	if err := db.Get(&client, "id=?", policy.ClientID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf("the EBICS client %d does not exist", policy.ClientID)
		}

		return fmt.Errorf("failed to retrieve EBICS client for contract refresh policy: %w", err)
	}

	if client.Protocol != "ebics" {
		return database.NewValidationErrorf(
			"the contract refresh policy client %d does not use the EBICS protocol", policy.ClientID)
	}

	if client.Disabled {
		return database.NewValidationErrorf("the contract refresh policy client %d is disabled", policy.ClientID)
	}

	var subscriber EbicsSubscriber
	if err := db.Get(&subscriber, "id=?", policy.EbicsSubscriberID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf(
				"the EBICS subscriber %d does not exist", policy.EbicsSubscriberID)
		}

		return fmt.Errorf("failed to retrieve EBICS subscriber for contract refresh policy: %w", err)
	}

	if !subscriber.Enabled {
		return database.NewValidationErrorf("the EBICS subscriber %d is disabled", policy.EbicsSubscriberID)
	}

	if !subscriber.RemoteAccountID.Valid || subscriber.AccountRole == ebicsSubscriberAccountRoleServer {
		return database.NewValidationError(
			"the EBICS contract refresh policy subscriber must reference a client-side remote account")
	}

	return nil
}

func (p *EbicsContractRefreshPolicy) Interval() time.Duration {
	return time.Duration(p.IntervalSeconds) * time.Second
}
