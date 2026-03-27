package ebics

import (
	"context"
	"database/sql"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
	"time"

	libebicsclient "code.waarp.fr/lib/ebics/ebics/client"
	ebicsxml "code.waarp.fr/lib/ebics/ebics/xml"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

// ContractRefreshResult exposes the result of a client-side HEV/contract refresh cycle.
type ContractRefreshResult struct {
	ProtocolCheckOperation *model.EbicsOperation
	ContractOperations     []*model.EbicsOperation
	ContractViews          []*model.EbicsContractView
}

const (
	contractRefreshOrderCount      = 4
	hpdBaseContractCapabilityCount = 4
)

var (
	errUnsupportedContractRefreshOrder = errors.New("unsupported contract refresh order")
	errMissingPartnerContractInfo      = errors.New("missing partner contract information")
	errMissingPartnerOrderInfo         = errors.New("missing partner order information")
	errMissingUserPermission           = errors.New("missing user permission")
)

// RefreshContractViews runs HEV optionally, then refreshes HPD/HKD/HTD/HAA and persists snapshots.
func RefreshContractViews(
	ctx context.Context,
	db *database.DB,
	subscriberID int64,
	includeHEV bool,
) (*ContractRefreshResult, error) {
	service, stop, err := startOperationalClient(ctx, db)
	if err != nil {
		return nil, err
	}
	defer stop()

	return service.refreshContractViews(subscriberID, includeHEV)
}

func (c *Client) refreshContractViews(subscriberID int64, includeHEV bool) (*ContractRefreshResult, error) {
	if !c.state.IsRunning() {
		return nil, errClientNotRunning
	}

	execCtx, err := c.newAdminExecutionContext(subscriberID)
	if err != nil {
		return nil, err
	}

	result := &ContractRefreshResult{
		ContractOperations: make([]*model.EbicsOperation, 0, contractRefreshOrderCount),
		ContractViews:      make([]*model.EbicsContractView, 0, contractRefreshOrderCount),
	}

	if includeHEV {
		op, hevErr := c.fetchProtocolVersion(execCtx)
		if hevErr != nil {
			return nil, hevErr
		}
		result.ProtocolCheckOperation = op
	}

	for _, orderType := range []string{"HPD", "HKD", "HTD", "HAA"} {
		view, op, refreshErr := c.refreshContractOrder(execCtx, orderType)
		if refreshErr != nil {
			return nil, refreshErr
		}

		result.ContractOperations = append(result.ContractOperations, op)
		result.ContractViews = append(result.ContractViews, view)
	}

	return result, nil
}

func (c *Client) fetchProtocolVersion(execCtx *adminExecutionContext) (*model.EbicsOperation, error) {
	operation, err := c.insertNonPayloadOperation(execCtx, "ADMIN", "HEV", "INBOUND")
	if err != nil {
		return nil, err
	}

	requestCtx, cancel := context.WithTimeout(context.Background(), c.adminRequestTimeout())
	defer cancel()

	hev, err := execCtx.libClient.DownloadHEV(requestCtx, libebicsclient.FlowHEVRequired{
		URL:    execCtx.endpointURL,
		HostID: execCtx.host.HostID,
	})
	if err != nil {
		return nil, c.failNonPayloadOperation(operation, "download HEV protocol version", err)
	}

	versions := make([]string, 0, len(hev.VersionNumbers))
	for _, version := range hev.VersionNumbers {
		versions = append(versions, strings.TrimSpace(version.ProtocolVersion)+"="+strings.TrimSpace(version.Value))
	}

	operation.MetadataMap["hev"] = map[string]any{
		"hostID":          execCtx.host.HostID,
		"returnCode":      strings.TrimSpace(hev.SystemReturnCode.ReturnCode),
		"reportText":      strings.TrimSpace(hev.SystemReturnCode.ReportText),
		"supportedLevels": versions,
	}

	if completeErr := c.completeNonPayloadOperation(
		operation,
		strings.TrimSpace(hev.SystemReturnCode.ReturnCode),
		"",
	); completeErr != nil {
		return nil, completeErr
	}

	return operation, nil
}

func (c *Client) refreshContractOrder(
	execCtx *adminExecutionContext,
	orderType string,
) (*model.EbicsContractView, *model.EbicsOperation, error) {
	operation, err := c.insertNonPayloadOperation(execCtx, "CONTRACT_REFRESH", orderType, "INBOUND")
	if err != nil {
		return nil, nil, err
	}

	requestCtx, cancel := context.WithTimeout(context.Background(), c.adminRequestTimeout())
	defer cancel()

	var (
		view        *model.EbicsContractView
		itemCount   int
		completeErr error
	)

	switch orderType {
	case "HPD":
		doc, response, downloadErr := execCtx.libClient.DownloadHPDDocument(
			requestCtx,
			libebicsclient.FlowHPDRequired{
				URL:       execCtx.endpointURL,
				HostID:    execCtx.host.HostID,
				PartnerID: execCtx.subscriber.PartnerID,
				UserID:    execCtx.subscriber.UserID,
			},
			libebicsclient.FlowHPDOptional{
				RequestSigner:  execCtx.requestSigner,
				ResponseSigner: execCtx.responseSigner,
				Cipher:         execCtx.downloadCipher,
			},
		)
		if downloadErr != nil {
			return nil, nil, c.failNonPayloadOperation(operation, "download HPD contract snapshot", downloadErr)
		}

		items := buildHPDContractViewItems(doc)
		itemCount = len(items)
		view, err = c.persistContractView(operation, execCtx, "HPD", versionTagForHPD(doc), items)
		if err != nil {
			return nil, nil, c.failNonPayloadOperation(operation, "persist HPD contract snapshot", err)
		}

		operation.MetadataMap["contractRefresh"] = map[string]any{
			"sourceOrderType": "HPD",
			"itemCount":       itemCount,
			"viewID":          view.ID,
		}
		technicalCode, businessCode := extractResponseReturnCodes(response)
		completeErr = c.completeNonPayloadOperation(operation, technicalCode, businessCode)
	case "HKD":
		view, completeErr = c.refreshPartnerContractOrder(requestCtx, execCtx, operation, "HKD")
	case "HTD":
		view, completeErr = c.refreshPartnerContractOrder(requestCtx, execCtx, operation, "HTD")
	case "HAA":
		doc, response, downloadErr := execCtx.libClient.DownloadHAADocument(
			requestCtx,
			libebicsclient.FlowHAARequired{
				URL:       execCtx.endpointURL,
				HostID:    execCtx.host.HostID,
				PartnerID: execCtx.subscriber.PartnerID,
				UserID:    execCtx.subscriber.UserID,
			},
			libebicsclient.FlowHAAOptional{
				RequestSigner:  execCtx.requestSigner,
				ResponseSigner: execCtx.responseSigner,
				Cipher:         execCtx.downloadCipher,
			},
		)
		if downloadErr != nil {
			return nil, nil, c.failNonPayloadOperation(operation, "download HAA contract snapshot", downloadErr)
		}

		items := buildHAAContractViewItems(doc.Service)
		itemCount = len(items)
		view, err = c.persistContractView(operation, execCtx, "HAA", versionTagForGenericContractSnapshot(), items)
		if err != nil {
			return nil, nil, c.failNonPayloadOperation(operation, "persist HAA contract snapshot", err)
		}

		operation.MetadataMap["contractRefresh"] = map[string]any{
			"sourceOrderType": "HAA",
			"itemCount":       itemCount,
			"viewID":          view.ID,
		}
		technicalCode, businessCode := extractResponseReturnCodes(response)
		completeErr = c.completeNonPayloadOperation(operation, technicalCode, businessCode)
	default:
		return nil, nil, c.failNonPayloadOperation(operation, "refresh contract snapshot", errUnsupportedInitializationOrder)
	}

	if completeErr != nil {
		return nil, nil, completeErr
	}

	return view, operation, nil
}

func (c *Client) refreshPartnerContractOrder(
	requestCtx context.Context,
	execCtx *adminExecutionContext,
	operation *model.EbicsOperation,
	orderType string,
) (*model.EbicsContractView, error) {
	required := libebicsclient.FlowHKDRequired{
		URL:       execCtx.endpointURL,
		HostID:    execCtx.host.HostID,
		PartnerID: execCtx.subscriber.PartnerID,
		UserID:    execCtx.subscriber.UserID,
	}
	optional := libebicsclient.FlowHKDOptional{
		RequestSigner:  execCtx.requestSigner,
		ResponseSigner: execCtx.responseSigner,
		Cipher:         execCtx.downloadCipher,
	}

	var (
		partnerInfo ebicsxml.PartnerInfo
		users       []ebicsxml.UserInfo
		response    *ebicsxml.EbicsResponse
		err         error
	)

	switch orderType {
	case "HKD":
		var doc *ebicsxml.HKDResponseOrderData
		doc, response, err = execCtx.libClient.DownloadHKDDocument(requestCtx, required, optional)
		if err != nil {
			return nil, c.failNonPayloadOperation(operation, "download HKD contract snapshot", err)
		}
		partnerInfo = doc.PartnerInfo
		users = doc.UserInfo
	case "HTD":
		var doc *ebicsxml.HTDResponseOrderData
		doc, response, err = execCtx.libClient.DownloadHTDDocument(
			requestCtx,
			libebicsclient.FlowHTDRequired(required),
			libebicsclient.FlowHTDOptional(optional),
		)
		if err != nil {
			return nil, c.failNonPayloadOperation(operation, "download HTD contract snapshot", err)
		}
		partnerInfo = doc.PartnerInfo
		users = doc.UserInfo
	default:
		return nil, c.failNonPayloadOperation(
			operation,
			"refresh partner contract snapshot",
			fmt.Errorf("%w: %q", errUnsupportedContractRefreshOrder, orderType),
		)
	}

	items, buildErr := buildPartnerContractViewItems(orderType, &partnerInfo, users)
	if buildErr != nil {
		return nil, c.failNonPayloadOperation(
			operation,
			fmt.Sprintf("build %s contract snapshot", orderType),
			buildErr,
		)
	}

	view, persistErr := c.persistContractView(
		operation,
		execCtx,
		orderType,
		versionTagForGenericContractSnapshot(),
		items,
	)
	if persistErr != nil {
		return nil, c.failNonPayloadOperation(
			operation,
			fmt.Sprintf("persist %s contract snapshot", orderType),
			persistErr,
		)
	}

	operation.MetadataMap["contractRefresh"] = map[string]any{
		"sourceOrderType": orderType,
		"itemCount":       len(items),
		"viewID":          view.ID,
	}
	technicalCode, businessCode := extractResponseReturnCodes(response)
	if completeErr := c.completeNonPayloadOperation(operation, technicalCode, businessCode); completeErr != nil {
		return nil, completeErr
	}

	return view, nil
}

func (c *Client) persistContractView(
	operation *model.EbicsOperation,
	execCtx *adminExecutionContext,
	sourceOrderType, versionTag string,
	items []model.EbicsContractViewItem,
) (*model.EbicsContractView, error) {
	now := time.Now().UTC()
	view := &model.EbicsContractView{}

	err := c.db.Transaction(func(tx *database.Session) error {
		var previous model.EbicsContractViews
		if selectErr := tx.Select(&previous).Owner().Where(
			"ebics_host_id=? AND ebics_subscriber_id=? AND source_order_type=? AND status=?",
			execCtx.host.ID,
			execCtx.subscriber.ID,
			sourceOrderType,
			"ACTIVE",
		).Run(); selectErr != nil {
			return fmt.Errorf("load active EBICS %s contract views: %w", sourceOrderType, selectErr)
		}

		for _, current := range previous {
			current.Status = "SUPERSEDED"
			if updateErr := tx.Update(current).Run(); updateErr != nil {
				return fmt.Errorf("supersede EBICS %s contract view %d: %w", sourceOrderType, current.ID, updateErr)
			}
		}

		*view = model.EbicsContractView{
			EbicsHostID:       execCtx.host.ID,
			EbicsSubscriberID: sql.NullInt64{Int64: execCtx.subscriber.ID, Valid: true},
			SourceOrderType:   sourceOrderType,
			SourceOperationID: sql.NullInt64{Int64: operation.ID, Valid: true},
			VersionTag:        versionTag,
			Status:            "ACTIVE",
			FetchedAt:         now,
		}
		if insertErr := tx.Insert(view).Run(); insertErr != nil {
			return fmt.Errorf("insert EBICS %s contract view: %w", sourceOrderType, insertErr)
		}

		for idx := range items {
			item := items[idx]
			item.ContractViewID = view.ID
			if insertErr := tx.Insert(&item).Run(); insertErr != nil {
				return fmt.Errorf(
					"insert EBICS %s contract item %q for view %d: %w",
					sourceOrderType,
					item.ItemKey,
					view.ID,
					insertErr,
				)
			}
		}

		operation.ContractViewID = sql.NullInt64{Int64: view.ID, Valid: true}
		if updateErr := tx.Update(operation).Run(); updateErr != nil {
			return fmt.Errorf("attach EBICS %s contract view %d to operation %d: %w",
				sourceOrderType, view.ID, operation.ID, updateErr)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return view, nil
}

func buildHPDContractViewItems(doc *ebicsxml.HPDResponseOrderData) []model.EbicsContractViewItem {
	items := make([]model.EbicsContractViewItem, 0, hpdBaseContractCapabilityCount+len(doc.AccessParams.URLs))

	if institute := strings.TrimSpace(doc.AccessParams.Institute); institute != "" {
		items = append(items, model.EbicsContractViewItem{
			ItemType:  "CAPABILITY",
			ItemKey:   "HPD:INSTITUTE",
			IsEnabled: true,
			Payload:   institute,
		})
	}

	for idx, entry := range doc.AccessParams.URLs {
		items = append(items, model.EbicsContractViewItem{
			ItemType:  "CAPABILITY",
			ItemKey:   fmt.Sprintf("HPD:URL:%d", idx+1),
			IsEnabled: true,
			Payload:   strings.TrimSpace(entry.Value),
		})
	}

	items = appendSupportFlagItem(items, "HPD:RECOVERY", doc.ProtocolParams.Recovery)
	items = appendSupportFlagItem(items, "HPD:PRE_VALIDATION", doc.ProtocolParams.PreValidation)
	items = appendSupportFlagItem(items, "HPD:CLIENT_DATA_DOWNLOAD", doc.ProtocolParams.ClientDataDownload)
	items = appendSupportFlagItem(items, "HPD:DOWNLOADABLE_ORDER_DATA", doc.ProtocolParams.DownloadableOrderData)

	items = append(items,
		model.EbicsContractViewItem{
			ItemType:  "CAPABILITY",
			ItemKey:   "HPD:VERSIONS:PROTOCOL",
			IsEnabled: true,
			Payload:   strings.Join([]string(doc.ProtocolParams.Version.Protocol), " "),
		},
		model.EbicsContractViewItem{
			ItemType:  "CAPABILITY",
			ItemKey:   "HPD:VERSIONS:AUTHENTICATION",
			IsEnabled: true,
			Payload:   strings.Join([]string(doc.ProtocolParams.Version.Authentication), " "),
		},
		model.EbicsContractViewItem{
			ItemType:  "CAPABILITY",
			ItemKey:   "HPD:VERSIONS:ENCRYPTION",
			IsEnabled: true,
			Payload:   strings.Join([]string(doc.ProtocolParams.Version.Encryption), " "),
		},
		model.EbicsContractViewItem{
			ItemType:  "CAPABILITY",
			ItemKey:   "HPD:VERSIONS:SIGNATURE",
			IsEnabled: true,
			Payload:   strings.Join([]string(doc.ProtocolParams.Version.Signature), " "),
		},
	)

	return items
}

func appendSupportFlagItem(
	items []model.EbicsContractViewItem,
	itemKey string,
	flag *ebicsxml.SupportFlag,
) []model.EbicsContractViewItem {
	if flag == nil || flag.Supported == nil {
		return items
	}

	items = append(items, model.EbicsContractViewItem{
		ItemType:  "CAPABILITY",
		ItemKey:   itemKey,
		IsEnabled: *flag.Supported,
	})

	return items
}

func buildPartnerContractViewItems(
	sourceOrderType string,
	partner *ebicsxml.PartnerInfo,
	users []ebicsxml.UserInfo,
) ([]model.EbicsContractViewItem, error) {
	if partner == nil {
		return nil, fmt.Errorf("%w for %s", errMissingPartnerContractInfo, sourceOrderType)
	}

	items := make([]model.EbicsContractViewItem, 0, len(partner.OrderInfo)+len(users)*2)

	for idx := range partner.OrderInfo {
		item, err := buildPartnerOrderItem(sourceOrderType, idx, &partner.OrderInfo[idx])
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}

	for userIdx := range users {
		for permIdx := range users[userIdx].Permission {
			item, err := buildUserPermissionItem(
				sourceOrderType,
				userIdx,
				permIdx,
				&users[userIdx],
				&users[userIdx].Permission[permIdx],
			)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
		}
	}

	return items, nil
}

func buildPartnerOrderItem(
	sourceOrderType string,
	index int,
	info *ebicsxml.AuthOrderInfo,
) (model.EbicsContractViewItem, error) {
	if info == nil {
		return model.EbicsContractViewItem{}, fmt.Errorf("%w for %s", errMissingPartnerOrderInfo, sourceOrderType)
	}

	orderType := model.NormalizeEbicsOrderType(info.AdminOrderType)
	service, hasService, err := parseRestrictedService(info.Service)
	if err != nil {
		return model.EbicsContractViewItem{}, fmt.Errorf(
			"parse %s partner order info %q: %w", sourceOrderType, orderType, err)
	}

	if model.IsEbicsPayloadOrderType(orderType) {
		itemType := "ORDER_TYPE"
		if hasService {
			itemType = "BTF"
		}

		item := model.EbicsContractViewItem{
			ItemType:  itemType,
			ItemKey:   fmt.Sprintf("%s:ORDER:%d:%s", sourceOrderType, index+1, orderType),
			OrderType: orderType,
			IsEnabled: true,
		}
		populateServiceDescriptor(&item, service)

		return item, nil
	}

	return model.EbicsContractViewItem{
		ItemType:       "ADMIN_ORDER",
		ItemKey:        fmt.Sprintf("%s:ADMIN:%d:%s", sourceOrderType, index+1, orderType),
		AdminOrderType: orderType,
		IsEnabled:      true,
		Payload:        strings.TrimSpace(info.Description),
	}, nil
}

func buildUserPermissionItem(
	sourceOrderType string,
	userIdx, permIdx int,
	user *ebicsxml.UserInfo,
	perm *ebicsxml.UserPermission,
) (model.EbicsContractViewItem, error) {
	if user == nil || perm == nil {
		return model.EbicsContractViewItem{}, fmt.Errorf("%w for %s", errMissingUserPermission, sourceOrderType)
	}

	orderType := model.NormalizeEbicsOrderType(perm.AdminOrderType)
	service, hasService, err := parseRestrictedService(perm.Service)
	if err != nil {
		return model.EbicsContractViewItem{}, fmt.Errorf(
			"parse %s user permission %q for user %q: %w",
			sourceOrderType,
			orderType,
			strings.TrimSpace(user.UserID.Value),
			err,
		)
	}

	if model.IsEbicsPayloadOrderType(orderType) {
		itemType := "ORDER_TYPE"
		if hasService {
			itemType = "BTF"
		}

		item := model.EbicsContractViewItem{
			ItemType:           itemType,
			ItemKey:            fmt.Sprintf("%s:USER:%d:%d:%s", sourceOrderType, userIdx+1, permIdx+1, orderType),
			OrderType:          orderType,
			AuthorisationLevel: strings.TrimSpace(perm.AuthorisationLevel),
			AccountID:          strings.TrimSpace(perm.AccountID),
			IsEnabled:          true,
		}
		populateServiceDescriptor(&item, service)
		populateMaxAmount(&item, perm.MaxAmount)

		return item, nil
	}

	item := model.EbicsContractViewItem{
		ItemType:           "PERMISSION",
		ItemKey:            fmt.Sprintf("%s:USER:%d:%d:%s", sourceOrderType, userIdx+1, permIdx+1, orderType),
		AdminOrderType:     orderType,
		AuthorisationLevel: strings.TrimSpace(perm.AuthorisationLevel),
		AccountID:          strings.TrimSpace(perm.AccountID),
		IsEnabled:          true,
	}
	// Keep service hints when the bank returns them even for non-payload orders.
	populateServiceDescriptor(&item, service)
	populateMaxAmount(&item, perm.MaxAmount)

	return item, nil
}

func buildHAAContractViewItems(services []ebicsxml.RestrictedService) []model.EbicsContractViewItem {
	items := make([]model.EbicsContractViewItem, 0, len(services))

	for idx := range services {
		item := model.EbicsContractViewItem{
			ItemType:  "CAPABILITY",
			ItemKey:   fmt.Sprintf("HAA:SERVICE:%d", idx+1),
			IsEnabled: true,
		}
		populateServiceDescriptor(&item, &services[idx])
		items = append(items, item)
	}

	return items
}

func populateServiceDescriptor(item *model.EbicsContractViewItem, service any) {
	if item == nil || service == nil {
		return
	}

	var descriptor ebicsxml.ServiceDescriptor

	switch value := service.(type) {
	case *ebicsxml.RestrictedService:
		descriptor = value.Descriptor()
	case ebicsxml.RestrictedService:
		descriptor = value.Descriptor()
	default:
		return
	}

	item.ServiceName = strings.TrimSpace(descriptor.ServiceName)
	item.ServiceOption = strings.TrimSpace(descriptor.ServiceOption)
	item.Scope = strings.TrimSpace(descriptor.Scope)
	if descriptor.Container != nil {
		item.ContainerType = strings.TrimSpace(descriptor.Container.ContainerType)
	}
	if descriptor.MsgName != nil {
		item.MsgName = strings.TrimSpace(descriptor.MsgName.Value)
	}
}

func populateMaxAmount(item *model.EbicsContractViewItem, amount *ebicsxml.Amount) {
	if item == nil || amount == nil {
		return
	}

	item.MaxAmountValue = strings.TrimSpace(amount.Value)
	item.MaxAmountCurrency = strings.ToUpper(strings.TrimSpace(amount.Currency))
}

func parseRestrictedService(raw *ebicsxml.RawElement) (ebicsxml.RestrictedService, bool, error) {
	if raw == nil || strings.TrimSpace(raw.InnerXML) == "" {
		return ebicsxml.RestrictedService{}, false, nil
	}

	var service ebicsxml.RestrictedService
	data := []byte(`<Service xmlns="` + ebicsxml.NamespaceH005 + `">` + raw.InnerXML + `</Service>`)
	if err := xml.Unmarshal(data, &service); err != nil {
		return ebicsxml.RestrictedService{}, false, fmt.Errorf("unmarshal restricted service: %w", err)
	}
	if err := ebicsxml.ValidateRestrictedService(service); err != nil {
		return ebicsxml.RestrictedService{}, false, fmt.Errorf("validate restricted service: %w", err)
	}

	return service, true, nil
}

func versionTagForHPD(doc *ebicsxml.HPDResponseOrderData) string {
	if doc == nil {
		return versionTagForGenericContractSnapshot()
	}

	if ts := strings.TrimSpace(doc.ProtocolParams.Version.Timestamp); ts != "" {
		return ts
	}

	return versionTagForGenericContractSnapshot()
}

func versionTagForGenericContractSnapshot() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func extractResponseReturnCodes(response *ebicsxml.EbicsResponse) (technicalCode, businessCode string) {
	if response == nil {
		return "", ""
	}

	if response.Header != nil && response.Header.Mutable != nil {
		if code := strings.TrimSpace(response.Header.Mutable.ReturnCode); code != "" {
			return code, ""
		}
	}

	if response.Body != nil {
		if code := strings.TrimSpace(response.Body.ReturnCode); code != "" {
			return "", code
		}
	}

	return "", ""
}

var errClientNotRunning = fmt.Errorf("%w: client service is not running", utils.ErrNotRunning)
