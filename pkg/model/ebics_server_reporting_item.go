package model

import (
	"fmt"
	"strings"
	"time"

	ebicsxml "code.waarp.fr/lib/ebics/ebics/xml"

	"code.waarp.fr/apps/gateway/gateway/pkg/conf"
	"code.waarp.fr/apps/gateway/gateway/pkg/database"
)

type EbicsServerReportingItem struct {
	ID    int64  `xorm:"<- id AUTOINCR"`
	Owner string `xorm:"owner"`

	ServerReportingSetID int64 `xorm:"server_reporting_set_id"`

	ItemKey       string `xorm:"item_key"`
	OrderID       string `xorm:"order_id"`
	ServiceName   string `xorm:"service_name"`
	ServiceOption string `xorm:"service_option"`
	Scope         string `xorm:"scope"`
	MsgName       string `xorm:"msg_name"`
	ContainerType string `xorm:"container_type"`
	IsEnabled     bool   `xorm:"is_enabled"`

	ResponsePayload []byte `xorm:"response_payload"`
	OriginalPayload []byte `xorm:"original_payload"`

	CreatedAt time.Time `xorm:"created_at CREATED DATETIME(6) UTC"`
	UpdatedAt time.Time `xorm:"updated_at UPDATED DATETIME(6) UTC"`
}

func (*EbicsServerReportingItem) TableName() string   { return TableEbicsServerReportingItems }
func (*EbicsServerReportingItem) Appellation() string { return NameEbicsServerReportingItem }
func (i *EbicsServerReportingItem) GetID() int64      { return i.ID }

func (i *EbicsServerReportingItem) BeforeWrite(db database.Access) error {
	i.Owner = conf.GlobalConfig.GatewayName
	i.ItemKey = strings.TrimSpace(i.ItemKey)
	i.OrderID = strings.TrimSpace(i.OrderID)
	i.ServiceName = strings.TrimSpace(i.ServiceName)
	i.ServiceOption = strings.TrimSpace(i.ServiceOption)
	i.Scope = strings.TrimSpace(i.Scope)
	i.MsgName = strings.TrimSpace(i.MsgName)
	i.ContainerType = strings.TrimSpace(i.ContainerType)

	if i.ServerReportingSetID == 0 {
		return database.NewValidationError("the EBICS server reporting set reference is missing")
	}
	if i.ItemKey == "" {
		return database.NewValidationError("the EBICS server reporting item key is missing")
	}

	var set EbicsServerReportingSet
	if err := db.Get(&set, "id=?", i.ServerReportingSetID).Run(); err != nil {
		if database.IsNotFound(err) {
			return database.NewValidationErrorf(
				"the EBICS server reporting set %d does not exist", i.ServerReportingSetID)
		}

		return fmt.Errorf("failed to retrieve EBICS server reporting set: %w", err)
	}

	return i.validateAgainstOrderType(set.SourceOrderType)
}

func (i *EbicsServerReportingItem) validateAgainstOrderType(orderType string) error {
	switch NormalizeEbicsOrderType(orderType) {
	case "HVD":
		return i.validateHVD()
	case "HVE":
		return i.validateHVE()
	case "HVU":
		return i.validateHVU()
	case "HVZ":
		return i.validateHVZ()
	case "HVT":
		return i.validateHVT()
	case "HAC":
		return i.validateHAC()
	case "HVS":
		return i.validateHVS()
	default:
		return database.NewValidationErrorf("%q is not a supported EBICS server reporting source order type", orderType)
	}
}

func (i *EbicsServerReportingItem) validateHVD() error {
	if err := i.requireOrderID("HVD"); err != nil {
		return err
	}
	if err := validateServerReportingService(i); err != nil {
		return err
	}
	if _, err := ebicsxml.ParseHVDResponseOrderData(i.ResponsePayload); err != nil {
		return database.NewValidationErrorf("invalid HVD response payload: %v", err)
	}

	return nil
}

func (i *EbicsServerReportingItem) validateHVE() error {
	if err := i.requireOrderID("HVE"); err != nil {
		return err
	}
	if err := validateServerReportingService(i); err != nil {
		return err
	}
	if len(i.ResponsePayload) == 0 {
		return database.NewValidationError("the EBICS server reporting HVE item requires a response payload")
	}
	if len(i.OriginalPayload) == 0 {
		return database.NewValidationError("the EBICS server reporting HVE item requires reference order data")
	}

	return nil
}

func (i *EbicsServerReportingItem) validateHVU() error {
	if _, err := ebicsxml.ParseHVUResponseOrderData(i.ResponsePayload); err != nil {
		return database.NewValidationErrorf("invalid HVU response payload: %v", err)
	}

	return nil
}

func (i *EbicsServerReportingItem) validateHVZ() error {
	if _, err := ebicsxml.ParseHVZResponseOrderData(i.ResponsePayload); err != nil {
		return database.NewValidationErrorf("invalid HVZ response payload: %v", err)
	}

	return nil
}

func (i *EbicsServerReportingItem) validateHVT() error {
	if err := i.requireOrderID("HVT"); err != nil {
		return err
	}
	if err := validateServerReportingService(i); err != nil {
		return err
	}
	if _, err := ebicsxml.ParseHVTResponseOrderData(i.ResponsePayload); err != nil {
		return database.NewValidationErrorf("invalid HVT response payload: %v", err)
	}

	return nil
}

func (i *EbicsServerReportingItem) validateHAC() error {
	if err := i.requireOrderID("HAC"); err != nil {
		return err
	}
	if _, err := ebicsxml.ParseHACDocument(i.ResponsePayload); err != nil {
		return database.NewValidationErrorf("invalid HAC response payload: %v", err)
	}

	return nil
}

func (i *EbicsServerReportingItem) validateHVS() error {
	if err := i.requireOrderID("HVS"); err != nil {
		return err
	}
	if err := validateServerReportingService(i); err != nil {
		return err
	}
	if len(i.OriginalPayload) == 0 {
		return database.NewValidationError("the EBICS server reporting HVS item requires reference order data")
	}

	return nil
}

func (i *EbicsServerReportingItem) requireOrderID(orderType string) error {
	if i.OrderID != "" {
		return nil
	}

	return database.NewValidationErrorf(
		"the EBICS server reporting %s item requires an order identifier",
		orderType,
	)
}

func validateServerReportingService(item *EbicsServerReportingItem) error {
	if item.ServiceName == "" || item.MsgName == "" {
		return database.NewValidationError("the EBICS server reporting service selector is incomplete")
	}

	service := ebicsxml.RestrictedService{
		ServiceName:   item.ServiceName,
		ServiceOption: item.ServiceOption,
		Scope:         item.Scope,
		MsgName:       ebicsxml.MessageType{Value: item.MsgName},
	}
	if item.ContainerType != "" {
		service.Container = &ebicsxml.ContainerFlag{ContainerType: item.ContainerType}
	}
	if err := ebicsxml.ValidateRestrictedService(service); err != nil {
		return database.NewValidationErrorf("invalid EBICS server reporting service selector: %v", err)
	}

	return nil
}
