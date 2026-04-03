package model

import (
	"fmt"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

type EbicsServerContractItem struct {
	ID    int64  `xorm:"<- id AUTOINCR"`
	Owner string `xorm:"owner"`

	ServerContractSetID int64 `xorm:"server_contract_set_id"`

	ItemType           string `xorm:"item_type"`
	ItemKey            string `xorm:"item_key"`
	OrderType          string `xorm:"order_type"`
	ServiceName        string `xorm:"service_name"`
	ServiceOption      string `xorm:"service_option"`
	Scope              string `xorm:"scope"`
	MsgName            string `xorm:"msg_name"`
	ContainerType      string `xorm:"container_type"`
	AdminOrderType     string `xorm:"admin_order_type"`
	AuthorisationLevel string `xorm:"authorisation_level"`
	AccountID          string `xorm:"account_id"`
	MaxAmountValue     string `xorm:"max_amount_value"`
	MaxAmountCurrency  string `xorm:"max_amount_currency"`
	IsEnabled          bool   `xorm:"is_enabled"`
	Payload            string `xorm:"payload"`

	CreatedAt time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}

func (*EbicsServerContractItem) TableName() string   { return TableEbicsServerContractItems }
func (*EbicsServerContractItem) Appellation() string { return NameEbicsServerContractItem }
func (i *EbicsServerContractItem) GetID() int64      { return i.ID }

func (i *EbicsServerContractItem) BeforeWrite(db database.Access) error {
	i.Owner = conf.GlobalConfig.GatewayName
	i.ItemType = strings.ToUpper(strings.TrimSpace(i.ItemType))
	i.ItemKey = strings.TrimSpace(i.ItemKey)
	i.OrderType = NormalizeEbicsPayloadOrderType(i.OrderType)
	i.ServiceName = strings.TrimSpace(i.ServiceName)
	i.ServiceOption = strings.TrimSpace(i.ServiceOption)
	i.Scope = strings.TrimSpace(i.Scope)
	i.MsgName = strings.TrimSpace(i.MsgName)
	i.ContainerType = strings.TrimSpace(i.ContainerType)
	i.AdminOrderType = strings.ToUpper(strings.TrimSpace(i.AdminOrderType))
	i.AuthorisationLevel = strings.TrimSpace(i.AuthorisationLevel)
	i.AccountID = strings.TrimSpace(i.AccountID)
	i.MaxAmountValue = strings.TrimSpace(i.MaxAmountValue)
	i.MaxAmountCurrency = strings.ToUpper(strings.TrimSpace(i.MaxAmountCurrency))
	i.Payload = strings.TrimSpace(i.Payload)

	if i.ServerContractSetID == 0 {
		return database.NewValidationError("the EBICS server contract set reference is missing")
	}
	if i.ItemKey == "" {
		return database.NewValidationError("the EBICS server contract item key is missing")
	}
	if err := validateEbicsContractItemType(i.ItemType); err != nil {
		return err
	}

	shadow := &EbicsContractViewItem{
		ContractViewID:     1,
		ItemType:           i.ItemType,
		ItemKey:            i.ItemKey,
		OrderType:          i.OrderType,
		ServiceName:        i.ServiceName,
		ServiceOption:      i.ServiceOption,
		Scope:              i.Scope,
		MsgName:            i.MsgName,
		ContainerType:      i.ContainerType,
		AdminOrderType:     i.AdminOrderType,
		AuthorisationLevel: i.AuthorisationLevel,
		AccountID:          i.AccountID,
		MaxAmountValue:     i.MaxAmountValue,
		MaxAmountCurrency:  i.MaxAmountCurrency,
		IsEnabled:          i.IsEnabled,
		Payload:            i.Payload,
	}
	if err := validateEbicsContractItemCoherence(shadow); err != nil {
		return err
	}

	var set EbicsServerContractSet
	if err := db.Get(&set, "id=?", i.ServerContractSetID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf(
				"the EBICS server contract set %d does not exist", i.ServerContractSetID)
		}

		return fmt.Errorf("failed to retrieve EBICS server contract set: %w", err)
	}

	return nil
}
