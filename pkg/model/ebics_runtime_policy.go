package model

import (
	"fmt"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

const (
	defaultEbicsMaintenanceInterval  = 6 * time.Hour
	defaultEbicsTransactionRetention = 7 * 24 * time.Hour
	defaultEbicsRTNEventRetention    = 30 * 24 * time.Hour

	DefaultEbicsMaintenanceIntervalSeconds  int64 = int64(defaultEbicsMaintenanceInterval / time.Second)
	DefaultEbicsTransactionRetentionSeconds int64 = int64(defaultEbicsTransactionRetention / time.Second)
	DefaultEbicsRTNEventRetentionSeconds    int64 = int64(defaultEbicsRTNEventRetention / time.Second)
)

// EbicsRuntimePolicy stores the administrable runtime policy for EBICS maintenance.
type EbicsRuntimePolicy struct {
	ID    int64  `xorm:"<- id AUTOINCR"`
	Owner string `xorm:"owner"`
	Name  string `xorm:"name"`

	Enabled                     bool      `xorm:"enabled"`
	MaintenanceIntervalSeconds  int64     `xorm:"maintenance_interval_seconds"`
	TransactionRetentionSeconds int64     `xorm:"transaction_retention_seconds"`
	RTNEventRetentionSeconds    int64     `xorm:"rtn_event_retention_seconds"`
	CreatedAt                   time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt                   time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}

func (*EbicsRuntimePolicy) TableName() string   { return TableEbicsRuntimePolicies }
func (*EbicsRuntimePolicy) Appellation() string { return NameEbicsRuntimePolicy }
func (p *EbicsRuntimePolicy) GetID() int64      { return p.ID }

func (p *EbicsRuntimePolicy) BeforeWrite(db database.Access) error {
	p.Owner = conf.GlobalConfig.GatewayName
	p.Name = strings.TrimSpace(p.Name)

	if p.Name == "" {
		p.Name = "default"
	}

	if p.MaintenanceIntervalSeconds == 0 {
		p.MaintenanceIntervalSeconds = DefaultEbicsMaintenanceIntervalSeconds
	}

	if p.TransactionRetentionSeconds == 0 {
		p.TransactionRetentionSeconds = DefaultEbicsTransactionRetentionSeconds
	}

	if p.RTNEventRetentionSeconds == 0 {
		p.RTNEventRetentionSeconds = DefaultEbicsRTNEventRetentionSeconds
	}

	if p.MaintenanceIntervalSeconds < 0 {
		return database.NewValidationError("the EBICS maintenance interval cannot be negative")
	}

	if p.TransactionRetentionSeconds < 0 {
		return database.NewValidationError("the EBICS transaction retention cannot be negative")
	}

	if p.RTNEventRetentionSeconds < 0 {
		return database.NewValidationError("the EBICS RTN event retention cannot be negative")
	}

	if n, err := db.Count(p).Where("id<>? AND owner=? AND name=?", p.ID, p.Owner, p.Name).Run(); err != nil {
		return fmt.Errorf("failed to check duplicate EBICS runtime policies: %w", err)
	} else if n != 0 {
		return database.NewValidationErrorf("an EBICS runtime policy named %q already exists", p.Name)
	}

	return nil
}

func (p *EbicsRuntimePolicy) MaintenanceInterval() time.Duration {
	return time.Duration(p.MaintenanceIntervalSeconds) * time.Second
}

func (p *EbicsRuntimePolicy) TransactionRetention() time.Duration {
	return time.Duration(p.TransactionRetentionSeconds) * time.Second
}

func (p *EbicsRuntimePolicy) RTNEventRetention() time.Duration {
	return time.Duration(p.RTNEventRetentionSeconds) * time.Second
}

// DefaultEbicsRuntimePolicy returns the default policy values for the current gateway instance.
func DefaultEbicsRuntimePolicy() *EbicsRuntimePolicy {
	return &EbicsRuntimePolicy{
		Name:                        "default",
		Enabled:                     true,
		MaintenanceIntervalSeconds:  DefaultEbicsMaintenanceIntervalSeconds,
		TransactionRetentionSeconds: DefaultEbicsTransactionRetentionSeconds,
		RTNEventRetentionSeconds:    DefaultEbicsRTNEventRetentionSeconds,
	}
}

// EnsureDefaultEbicsRuntimePolicy returns the default policy row, creating it if necessary.
func EnsureDefaultEbicsRuntimePolicy(db database.Access) (*EbicsRuntimePolicy, error) {
	var policy EbicsRuntimePolicy
	if err := db.Get(&policy, "owner=? AND name=?", conf.GlobalConfig.GatewayName, "default").Run(); err == nil {
		return &policy, nil
	} else if !database.IsNotFound(err) {
		return nil, fmt.Errorf("failed to retrieve the EBICS runtime policy: %w", err)
	}

	policy = *DefaultEbicsRuntimePolicy()
	if err := db.Insert(&policy).Run(); err != nil {
		if getErr := db.Get(&policy, "owner=? AND name=?", conf.GlobalConfig.GatewayName, "default").Run(); getErr == nil {
			return &policy, nil
		}

		return nil, fmt.Errorf("failed to create the default EBICS runtime policy: %w", err)
	}

	return &policy, nil
}
