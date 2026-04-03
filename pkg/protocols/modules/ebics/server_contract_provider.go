package ebics

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	libebics "code.waarp.fr/lib/ebics/ebics"
	liborders "code.waarp.fr/lib/ebics/ebics/orders"
	ebicsxml "code.waarp.fr/lib/ebics/ebics/xml"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

type contractOrderProvider struct {
	db     *database.DB
	store  *providerStore
	server *model.LocalAgent
	source string
}

const (
	contractItemTypeBTF               = "BTF"
	contractItemTypeOrder             = "ORDER_TYPE"
	contractItemTypePermission        = "PERMISSION"
	contractItemTypeAccountPermission = "ACCOUNT_PERMISSION"
	estimatedContractItemsPerView     = 4
)

var errServerContractNoAccessURL = errors.New("no EBICS access URL configured for host")

func newContractOrderProvider(
	db *database.DB,
	store *providerStore,
	server *model.LocalAgent,
	source string,
) *contractOrderProvider {
	return &contractOrderProvider{
		db:     db,
		store:  store,
		server: server,
		source: strings.ToUpper(strings.TrimSpace(source)),
	}
}

func (p *contractOrderProvider) AccessParams(
	_ context.Context,
	hostID libebics.HostID,
) (liborders.HPDAccessParams, error) {
	host, err := p.store.getHostByHostID(string(hostID))
	if err != nil {
		return liborders.HPDAccessParams{}, err
	}

	url := strings.TrimSpace(host.DefaultBankURL)
	if url == "" && p.server != nil {
		url = "https://" + p.server.Address.Host + ":" + utils.FormatUint(p.server.Address.Port)
	}
	if url == "" {
		return liborders.HPDAccessParams{}, fmt.Errorf("%w %q", errServerContractNoAccessURL, host.HostID)
	}

	institute := strings.TrimSpace(host.Name)
	if institute == "" {
		institute = strings.TrimSpace(host.HostID)
	}

	validFrom := host.UpdatedAt.UTC()
	if validFrom.IsZero() {
		validFrom = time.Now().UTC()
	}

	return liborders.HPDAccessParams{
		Institute: institute,
		HostID:    host.HostID,
		URLs: []liborders.HPDURL{{
			Value:     url,
			ValidFrom: validFrom.Format(time.RFC3339),
		}},
	}, nil
}

func (p *contractOrderProvider) ProtocolParams(
	_ context.Context,
	hostID libebics.HostID,
) (liborders.HPDProtocolParams, error) {
	host, err := p.store.getHostByHostID(string(hostID))
	if err != nil {
		return liborders.HPDProtocolParams{}, err
	}

	items, err := p.loadActiveHostContractItems(host.ID, "HPD")
	if err != nil {
		return liborders.HPDProtocolParams{}, err
	}

	recovery := true
	preValidation := true
	clientDataDownload := true
	downloadableOrderData := true

	out := liborders.HPDProtocolParams{
		Version: liborders.HPDVersion{
			Protocol:       []string{host.ProtocolVersion},
			Authentication: p.bankKeyVersions(host.ID, "AUTH", "X002"),
			Encryption:     p.bankKeyVersions(host.ID, "ENCRYPT", "E002"),
			Signature:      p.bankKeyVersions(host.ID, "SIGNATURE", "A006"),
			Timestamp:      time.Now().UTC().Format(time.RFC3339),
		},
		Recovery:              &recovery,
		PreValidation:         &preValidation,
		ClientDataDownload:    &clientDataDownload,
		DownloadableOrderData: &downloadableOrderData,
	}

	for idx := range items {
		item := &items[idx]
		switch strings.ToUpper(strings.TrimSpace(item.ItemKey)) {
		case "HPD:RECOVERY":
			flag := item.IsEnabled
			out.Recovery = &flag
		case "HPD:PRE_VALIDATION":
			flag := item.IsEnabled
			out.PreValidation = &flag
		case "HPD:CLIENT_DATA_DOWNLOAD":
			flag := item.IsEnabled
			out.ClientDataDownload = &flag
		case "HPD:DOWNLOADABLE_ORDER_DATA":
			flag := item.IsEnabled
			out.DownloadableOrderData = &flag
		case "HPD:VERSIONS:PROTOCOL":
			out.Version.Protocol = nonEmptyFields(item.Payload, out.Version.Protocol)
		case "HPD:VERSIONS:AUTHENTICATION":
			out.Version.Authentication = nonEmptyFields(item.Payload, out.Version.Authentication)
		case "HPD:VERSIONS:ENCRYPTION":
			out.Version.Encryption = nonEmptyFields(item.Payload, out.Version.Encryption)
		case "HPD:VERSIONS:SIGNATURE":
			out.Version.Signature = nonEmptyFields(item.Payload, out.Version.Signature)
		}
	}

	return out, nil
}

func (p *contractOrderProvider) PartnerInfo(
	_ context.Context,
	hostID libebics.HostID,
	partnerID libebics.PartnerID,
	userID libebics.UserID,
) (ebicsxml.PartnerInfo, error) {
	host, subscriber, items, err := p.loadSubscriberContractSnapshot(
		string(hostID),
		string(partnerID),
		string(userID),
		p.sourceOrderTypes()...,
	)
	if err != nil {
		return ebicsxml.PartnerInfo{}, err
	}

	info := ebicsxml.PartnerInfo{
		AddressInfo: ebicsxml.AddressInfo{Name: strings.TrimSpace(subscriber.PartnerID)},
		BankInfo:    ebicsxml.BankInfo{BankName: strings.TrimSpace(host.Name)},
	}

	accountMap := map[string]*ebicsxml.AccountInfo{}
	orderMap := map[string]ebicsxml.AuthOrderInfo{}

	for idx := range items {
		item := &items[idx]
		p.collectPartnerAccountInfo(accountMap, item)

		order, ok, orderErr := p.contractItemToPartnerOrder(item)
		if orderErr != nil {
			return ebicsxml.PartnerInfo{}, orderErr
		}
		if !ok {
			continue
		}

		key := partnerOrderKey(&order)
		if _, exists := orderMap[key]; !exists {
			orderMap[key] = order
		}
	}

	info.AccountInfo = sortAccountInfos(accountMap)
	info.OrderInfo = sortPartnerOrders(orderMap)

	return info, nil
}

func (p *contractOrderProvider) UserInfo(
	_ context.Context,
	hostID libebics.HostID,
	partnerID libebics.PartnerID,
	_ libebics.UserID,
) ([]ebicsxml.UserInfo, error) {
	host, err := p.store.getHostByHostID(string(hostID))
	if err != nil {
		return nil, err
	}

	var subscribers model.EbicsSubscribers
	if selectErr := p.db.Select(&subscribers).
		Where(
			"owner=? AND ebics_host_id=? AND partner_id=? AND enabled=?",
			host.Owner,
			host.ID,
			string(partnerID),
			true,
		).
		OrderBy("id", true).
		Run(); selectErr != nil {
		return nil, fmt.Errorf(
			"load EBICS subscribers for partner %q on host %q: %w",
			partnerID,
			host.HostID,
			selectErr,
		)
	}

	users := make([]ebicsxml.UserInfo, 0, len(subscribers))
	for _, subscriber := range subscribers {
		items, loadErr := p.loadActiveSubscriberContractItems(host.ID, subscriber.ID, p.sourceOrderTypes()...)
		if loadErr != nil {
			return nil, loadErr
		}

		user := ebicsxml.UserInfo{
			UserID: ebicsxml.UserIDWithStatus{
				Value:  subscriber.UserID,
				Status: "1",
			},
			Name: strings.TrimSpace(subscriber.Name),
		}

		seen := map[string]struct{}{}
		for idx := range items {
			item := &items[idx]
			perm, ok, permErr := p.contractItemToUserPermission(item)
			if permErr != nil {
				return nil, permErr
			}
			if !ok {
				continue
			}

			key := userPermissionKey(&perm)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			user.Permission = append(user.Permission, perm)
		}

		users = append(users, user)
	}

	return users, nil
}

func (p *contractOrderProvider) Services(
	_ context.Context,
	hostID libebics.HostID,
) ([]ebicsxml.RestrictedService, error) {
	host, err := p.store.getHostByHostID(string(hostID))
	if err != nil {
		return nil, err
	}

	items, err := p.loadActiveHostContractItems(host.ID, p.sourceOrderTypes()...)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		items, err = p.loadActiveHostContractItems(host.ID)
		if err != nil {
			return nil, err
		}
	}

	services := make([]ebicsxml.RestrictedService, 0, len(items))
	seen := map[string]struct{}{}
	for idx := range items {
		item := &items[idx]
		service, ok, serviceErr := contractItemRestrictedService(item)
		if serviceErr != nil {
			return nil, serviceErr
		}
		if !ok {
			continue
		}

		key := serviceDescriptorKey(service.Descriptor())
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		services = append(services, service)
	}

	return services, nil
}

func (p *contractOrderProvider) bankKeyVersions(hostID int64, keyType, fallback string) []string {
	var rows model.EbicsBankKeys
	if err := p.db.Select(&rows).
		Where("owner=? AND ebics_host_id=? AND key_type=? AND state=?", p.store.gatewayOwner(), hostID, keyType, "validated").
		OrderBy("id", true).
		Run(); err != nil {
		return []string{fallback}
	}

	versions := make([]string, 0, len(rows))
	for _, row := range rows {
		version := strings.ToUpper(strings.TrimSpace(row.Version))
		if version == "" || slices.Contains(versions, version) {
			continue
		}
		versions = append(versions, version)
	}
	if len(versions) == 0 {
		return []string{fallback}
	}

	return versions
}

func (p *contractOrderProvider) loadSubscriberContractSnapshot(
	hostID, partnerID, userID string,
	sourceOrderTypes ...string,
) (*model.EbicsHost, *model.EbicsSubscriber, []model.EbicsServerContractItem, error) {
	host, err := p.store.getHostByHostID(hostID)
	if err != nil {
		return nil, nil, nil, err
	}

	subscriber, err := p.store.getSubscriber(hostID, partnerID, userID)
	if err != nil {
		return nil, nil, nil, err
	}

	items, err := p.loadActiveSubscriberContractItems(host.ID, subscriber.ID, sourceOrderTypes...)
	if err != nil {
		return nil, nil, nil, err
	}

	return host, subscriber, items, nil
}

func (p *contractOrderProvider) loadActiveSubscriberContractItems(
	hostID, subscriberID int64,
	sourceOrderTypes ...string,
) ([]model.EbicsServerContractItem, error) {
	items, err := p.selectContractItems(hostID, utils.NewNullInt64(subscriberID), sourceOrderTypes...)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 && len(sourceOrderTypes) != 0 {
		return p.selectContractItems(hostID, utils.NewNullInt64(subscriberID))
	}

	return items, nil
}

func (p *contractOrderProvider) loadActiveHostContractItems(
	hostID int64,
	sourceOrderTypes ...string,
) ([]model.EbicsServerContractItem, error) {
	items, err := p.selectContractItems(hostID, sql.NullInt64{}, sourceOrderTypes...)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 && len(sourceOrderTypes) != 0 {
		return p.selectContractItems(hostID, sql.NullInt64{})
	}

	return items, nil
}

func (p *contractOrderProvider) selectContractItems(
	hostID int64,
	subscriberID sql.NullInt64,
	sourceOrderTypes ...string,
) ([]model.EbicsServerContractItem, error) {
	var sets model.EbicsServerContractSets
	query := p.db.Select(&sets).Owner().Where("ebics_host_id=? AND status=?", hostID, "ACTIVE")
	if subscriberID.Valid {
		query = query.Where("ebics_subscriber_id=?", subscriberID.Int64)
	} else {
		query = query.Where("ebics_subscriber_id IS NULL")
	}
	if len(sourceOrderTypes) != 0 {
		args := make([]any, 0, len(sourceOrderTypes))
		for _, sourceOrderType := range sourceOrderTypes {
			normalized := strings.ToUpper(strings.TrimSpace(sourceOrderType))
			if normalized == "" {
				continue
			}
			args = append(args, normalized)
		}
		if len(args) != 0 {
			query = query.In("source_order_type", args...)
		}
	}
	if err := query.OrderBy("id", true).Run(); err != nil {
		return nil, fmt.Errorf("load active EBICS server contract sets: %w", err)
	}
	if len(sets) == 0 {
		return nil, nil
	}

	items := make([]model.EbicsServerContractItem, 0, len(sets)*estimatedContractItemsPerView)
	for _, set := range sets {
		var rows model.EbicsServerContractItems
		if err := p.db.Select(&rows).
			Owner().
			Where("server_contract_set_id=?", set.ID).
			OrderBy("id", true).
			Run(); err != nil {
			return nil, fmt.Errorf(
				"load EBICS server contract items for set %d: %w",
				set.ID,
				err,
			)
		}
		for _, row := range rows {
			items = append(items, *row)
		}
	}

	return items, nil
}

func (p *contractOrderProvider) collectPartnerAccountInfo(
	accountMap map[string]*ebicsxml.AccountInfo,
	item *model.EbicsServerContractItem,
) {
	accountID := strings.TrimSpace(item.AccountID)
	if accountID == "" {
		return
	}

	account, exists := accountMap[accountID]
	if !exists {
		account = &ebicsxml.AccountInfo{ID: accountID}
		accountMap[accountID] = account
	}

	orderType := contractItemOrderType(item)
	if orderType == "" {
		return
	}

	if account.UsageOrderTypes == nil {
		account.UsageOrderTypes = &ebicsxml.UsageOrderTypes{}
	}
	if !slices.Contains(account.UsageOrderTypes.AdminOrderType, orderType) {
		account.UsageOrderTypes.AdminOrderType = append(account.UsageOrderTypes.AdminOrderType, orderType)
	}
}

func (p *contractOrderProvider) contractItemToPartnerOrder(
	item *model.EbicsServerContractItem,
) (ebicsxml.AuthOrderInfo, bool, error) {
	orderType := contractItemOrderType(item)
	if orderType == "" {
		return ebicsxml.AuthOrderInfo{}, false, nil
	}

	order := ebicsxml.AuthOrderInfo{
		AdminOrderType: orderType,
		Description:    strings.TrimSpace(item.Payload),
	}

	service, ok, err := contractItemRestrictedService(item)
	if err != nil {
		return ebicsxml.AuthOrderInfo{}, false, err
	}
	if ok {
		raw, marshalErr := ebicsxml.MarshalOrderParams(service)
		if marshalErr != nil {
			return ebicsxml.AuthOrderInfo{}, false, fmt.Errorf(
				"marshal EBICS service filter for %q: %w",
				orderType,
				marshalErr,
			)
		}
		order.Service = raw
	}

	return order, true, nil
}

func (p *contractOrderProvider) contractItemToUserPermission(
	item *model.EbicsServerContractItem,
) (ebicsxml.UserPermission, bool, error) {
	orderType := contractItemOrderType(item)
	if orderType == "" {
		return ebicsxml.UserPermission{}, false, nil
	}

	permission := ebicsxml.UserPermission{
		AdminOrderType: orderType,
		AccountID:      strings.TrimSpace(item.AccountID),
	}
	if !model.IsEbicsPayloadDownloadOrder(orderType) {
		permission.AuthorisationLevel = strings.TrimSpace(item.AuthorisationLevel)
	}
	if item.MaxAmountValue != "" {
		permission.MaxAmount = &ebicsxml.Amount{
			Value:    strings.TrimSpace(item.MaxAmountValue),
			Currency: strings.ToUpper(strings.TrimSpace(item.MaxAmountCurrency)),
		}
	}

	service, ok, err := contractItemRestrictedService(item)
	if err != nil {
		return ebicsxml.UserPermission{}, false, err
	}
	if ok {
		raw, marshalErr := ebicsxml.MarshalOrderParams(service)
		if marshalErr != nil {
			return ebicsxml.UserPermission{}, false, fmt.Errorf(
				"marshal EBICS permission service for %q: %w",
				orderType,
				marshalErr,
			)
		}
		permission.Service = raw
	}

	return permission, true, nil
}

func contractItemOrderType(item *model.EbicsServerContractItem) string {
	switch strings.ToUpper(strings.TrimSpace(item.ItemType)) {
	case contractItemTypeBTF, contractItemTypeOrder,
		contractItemTypePermission, contractItemTypeAccountPermission:
		return model.NormalizeEbicsOrderType(item.OrderType)
	case "ADMIN_ORDER":
		return model.NormalizeEbicsOrderType(item.AdminOrderType)
	default:
		return ""
	}
}

func contractItemRestrictedService(
	item *model.EbicsServerContractItem,
) (ebicsxml.RestrictedService, bool, error) {
	if strings.TrimSpace(item.ServiceName) == "" || strings.TrimSpace(item.MsgName) == "" {
		return ebicsxml.RestrictedService{}, false, nil
	}

	service := ebicsxml.RestrictedService{
		ServiceName:   strings.TrimSpace(item.ServiceName),
		Scope:         strings.TrimSpace(item.Scope),
		ServiceOption: strings.TrimSpace(item.ServiceOption),
		MsgName: ebicsxml.MessageType{
			Value: strings.TrimSpace(item.MsgName),
		},
	}
	if item.ContainerType != "" {
		service.Container = &ebicsxml.ContainerFlag{ContainerType: strings.TrimSpace(item.ContainerType)}
	}
	if err := ebicsxml.ValidateRestrictedService(service); err != nil {
		return ebicsxml.RestrictedService{}, false, fmt.Errorf(
			"invalid EBICS restricted service for item %q: %w",
			item.ItemKey,
			err,
		)
	}

	return service, true, nil
}

func partnerOrderKey(order *ebicsxml.AuthOrderInfo) string {
	service := ""
	if order.Service != nil {
		service = strings.TrimSpace(order.Service.InnerXML)
	}

	return strings.ToUpper(strings.TrimSpace(order.AdminOrderType)) + "|" + service
}

func userPermissionKey(permission *ebicsxml.UserPermission) string {
	service := ""
	if permission.Service != nil {
		service = strings.TrimSpace(permission.Service.InnerXML)
	}

	return strings.Join([]string{
		strings.ToUpper(strings.TrimSpace(permission.AdminOrderType)),
		strings.TrimSpace(permission.AuthorisationLevel),
		strings.TrimSpace(permission.AccountID),
		service,
	}, "|")
}

func serviceDescriptorKey(desc ebicsxml.ServiceDescriptor) string {
	container := ""
	if desc.Container != nil {
		container = strings.TrimSpace(desc.Container.ContainerType)
	}
	msg := ""
	if desc.MsgName != nil {
		msg = strings.TrimSpace(desc.MsgName.Value)
	}

	return strings.Join([]string{
		strings.TrimSpace(desc.ServiceName),
		strings.TrimSpace(desc.ServiceOption),
		strings.TrimSpace(desc.Scope),
		container,
		msg,
	}, "|")
}

func sortAccountInfos(accountMap map[string]*ebicsxml.AccountInfo) []ebicsxml.AccountInfo {
	keys := make([]string, 0, len(accountMap))
	for key := range accountMap {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	accounts := make([]ebicsxml.AccountInfo, 0, len(keys))
	for _, key := range keys {
		account := *accountMap[key]
		if account.UsageOrderTypes != nil {
			slices.Sort(account.UsageOrderTypes.AdminOrderType)
		}
		accounts = append(accounts, account)
	}

	return accounts
}

func sortPartnerOrders(orderMap map[string]ebicsxml.AuthOrderInfo) []ebicsxml.AuthOrderInfo {
	keys := make([]string, 0, len(orderMap))
	for key := range orderMap {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	orders := make([]ebicsxml.AuthOrderInfo, 0, len(keys))
	for _, key := range keys {
		orders = append(orders, orderMap[key])
	}

	return orders
}

func nonEmptyFields(payload string, fallback []string) []string {
	fields := strings.Fields(strings.TrimSpace(payload))
	if len(fields) == 0 {
		return fallback
	}

	return fields
}

func (p *contractOrderProvider) sourceOrderTypes() []string {
	if p.source == "" {
		return nil
	}

	return []string{p.source}
}
