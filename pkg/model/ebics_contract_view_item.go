package model

import (
	"fmt"
	"strings"
	"time"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

const (
	ebicsContractItemTypeOrder   = "ORDER_TYPE"
	ebicsContractItemTypeBTF     = "BTF"
	ebicsContractItemTypeAdmin   = "ADMIN_ORDER"
	ebicsContractItemTypePerm    = "PERMISSION"
	ebicsContractItemTypeAccount = "ACCOUNT_PERMISSION"
	ebicsContractItemTypeCap     = "CAPABILITY"
)

type EbicsContractViewItem struct {
	ID    int64  `xorm:"<- id AUTOINCR"`
	Owner string `xorm:"owner"`

	ContractViewID int64 `xorm:"contract_view_id"`

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

func (*EbicsContractViewItem) TableName() string   { return TableEbicsContractViewItems }
func (*EbicsContractViewItem) Appellation() string { return NameEbicsContractViewItem }
func (i *EbicsContractViewItem) GetID() int64      { return i.ID }

func (i *EbicsContractViewItem) BeforeWrite(db database.Access) error {
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

	if i.ContractViewID == 0 {
		return database.NewValidationError("the EBICS contract view reference is missing")
	}

	if i.ItemKey == "" {
		return database.NewValidationError("the EBICS contract item key is missing")
	}

	if err := validateEbicsContractItemType(i.ItemType); err != nil {
		return err
	}

	if err := validateEbicsContractItemCoherence(i); err != nil {
		return err
	}

	var view EbicsContractView
	if err := db.Get(&view, "id=?", i.ContractViewID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf(
				"the EBICS contract view %d does not exist", i.ContractViewID)
		}

		return fmt.Errorf("failed to retrieve EBICS contract view: %w", err)
	}

	return nil
}

func validateEbicsContractItemType(itemType string) error {
	switch itemType {
	case ebicsContractItemTypeOrder, ebicsContractItemTypeBTF, ebicsContractItemTypeAdmin,
		ebicsContractItemTypePerm, ebicsContractItemTypeAccount, ebicsContractItemTypeCap:
		return nil
	case "":
		return database.NewValidationError("the EBICS contract item type is missing")
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS contract item type", itemType)
	}
}

func validateEbicsContractItemCoherence(item *EbicsContractViewItem) error {
	switch item.ItemType {
	case ebicsContractItemTypeOrder, ebicsContractItemTypeBTF:
		if err := validateEbicsPayloadOrderType(item.OrderType); err != nil {
			return err
		}
	case ebicsContractItemTypeAdmin:
		if item.AdminOrderType == "" {
			return database.NewValidationError(
				"the EBICS contract admin order item requires an admin order type")
		}
	case ebicsContractItemTypeAccount:
		if item.AccountID == "" {
			return database.NewValidationError(
				"the EBICS contract account permission item requires an account ID")
		}
	}

	if item.MaxAmountCurrency != "" && item.MaxAmountValue == "" {
		return database.NewValidationError("the EBICS contract max amount currency requires a value")
	}

	if item.MaxAmountValue != "" && item.MaxAmountCurrency == "" {
		return database.NewValidationError("the EBICS contract max amount value requires a currency")
	}

	return nil
}
