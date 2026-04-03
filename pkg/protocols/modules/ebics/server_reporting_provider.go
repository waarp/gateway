package ebics

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"slices"
	"strings"

	libebics "code.waarp.fr/lib/ebics/ebics"
	libcrypto "code.waarp.fr/lib/ebics/ebics/crypto"
	liborders "code.waarp.fr/lib/ebics/ebics/orders"
	libserver "code.waarp.fr/lib/ebics/ebics/server"
	ebicsxml "code.waarp.fr/lib/ebics/ebics/xml"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const (
	serverReportingOrderHVD = "HVD"
	serverReportingOrderHVE = "HVE"
	serverReportingOrderHVU = "HVU"
	serverReportingOrderHVZ = "HVZ"
	serverReportingOrderHVT = "HVT"
	serverReportingOrderHAC = "HAC"
	serverReportingOrderHVS = "HVS"
)

type serverReportingProvider struct {
	db    *database.DB
	store *providerStore
	order string
}

func newServerReportingProvider(db *database.DB, store *providerStore, orderType string) *serverReportingProvider {
	return &serverReportingProvider{
		db:    db,
		store: store,
		order: model.NormalizeEbicsOrderType(orderType),
	}
}

//nolint:gocritic // lib-ebics provider interface imposes value semantics here.
func (p *serverReportingProvider) Response(
	_ context.Context,
	req libebics.OrderContext,
) (ebicsxml.HVDResponseOrderData, error) {
	item, _, err := p.selectSingleItem(&req)
	if err != nil {
		return ebicsxml.HVDResponseOrderData{}, err
	}

	doc, err := ebicsxml.ParseHVDResponseOrderData(item.ResponsePayload)
	if err != nil {
		return ebicsxml.HVDResponseOrderData{}, fmt.Errorf("%w: parse HVD payload: %v", liborders.ErrInvalidOrderData, err)
	}

	return *doc, nil
}

//nolint:gocritic // lib-ebics provider interface imposes value semantics here.
func (p *serverReportingProvider) ResponseHVU(
	_ context.Context,
	req libebics.OrderContext,
) (ebicsxml.HVUResponseOrderData, error) {
	return p.buildHVUResponse(&req)
}

//nolint:gocritic // lib-ebics provider interface imposes value semantics here.
func (p *serverReportingProvider) ResponseHVZ(
	_ context.Context,
	req libebics.OrderContext,
) (ebicsxml.HVZResponseOrderData, error) {
	return p.buildHVZResponse(&req)
}

//nolint:gocritic // lib-ebics provider interface imposes value semantics here.
func (p *serverReportingProvider) ResponseHVT(
	_ context.Context,
	req libebics.OrderContext,
) (ebicsxml.HVTResponseOrderData, error) {
	item, params, err := p.selectSingleItem(&req)
	if err != nil {
		return ebicsxml.HVTResponseOrderData{}, err
	}
	if params != nil && params.OrderFlags.CompleteOrderData {
		return ebicsxml.HVTResponseOrderData{}, fmt.Errorf(
			"%w: completeOrderData requests original payload",
			liborders.ErrInvalidOrderParams,
		)
	}

	doc, err := ebicsxml.ParseHVTResponseOrderData(item.ResponsePayload)
	if err != nil {
		return ebicsxml.HVTResponseOrderData{}, fmt.Errorf("%w: parse HVT payload: %v", liborders.ErrInvalidOrderData, err)
	}

	return *doc, nil
}

//nolint:gocritic // lib-ebics provider interface imposes value semantics here.
func (p *serverReportingProvider) OriginalData(
	_ context.Context,
	req libebics.OrderContext,
	_ *ebicsxml.HVTOrderParams,
) ([]byte, error) {
	item, _, err := p.selectSingleItem(&req)
	if err != nil {
		return nil, err
	}
	if len(item.OriginalPayload) == 0 {
		return nil, fmt.Errorf("%w: no original payload configured for order %q", liborders.ErrNoDownloadData, item.OrderID)
	}

	return append([]byte(nil), item.OriginalPayload...), nil
}

//nolint:gocritic // lib-ebics provider interface imposes value semantics here.
func (p *serverReportingProvider) ResponseDocument(
	_ context.Context,
	req libebics.OrderContext,
) (ebicsxml.HACDocument, error) {
	item, _, err := p.selectSingleItem(&req)
	if err != nil {
		return ebicsxml.HACDocument{}, err
	}

	doc, err := ebicsxml.ParseHACDocument(item.ResponsePayload)
	if err != nil {
		return ebicsxml.HACDocument{}, fmt.Errorf("%w: parse HAC payload: %v", liborders.ErrInvalidOrderData, err)
	}

	return *doc, nil
}

//nolint:gocritic // lib-ebics provider interface imposes value semantics here.
func (p *serverReportingProvider) EDSReferenceData(
	_ context.Context,
	req libebics.OrderContext,
	_ ebicsxml.OrderSignatureData,
) ([]byte, error) {
	item, _, err := p.selectSingleItem(&req)
	if err != nil {
		return nil, err
	}
	if len(item.OriginalPayload) == 0 {
		return nil, fmt.Errorf("%w: no reference data configured for order %q", liborders.ErrNoDownloadData, item.OrderID)
	}

	return append([]byte(nil), item.OriginalPayload...), nil
}

//nolint:gocritic // lib-ebics provider interface imposes value semantics here.
func (p *serverReportingProvider) Cancel(
	_ context.Context,
	req libebics.OrderContext,
	data ebicsxml.HVSRequestOrderData,
) error {
	item, _, err := p.selectSingleItem(&req)
	if err != nil {
		return err
	}
	if len(item.OriginalPayload) == 0 {
		return fmt.Errorf("%w: no reference data configured for order %q", liborders.ErrProcessing, item.OrderID)
	}

	expected, err := digestForReferenceData(item.OriginalPayload, data.CancelledDataDigest.SignatureVersion)
	if err != nil {
		return fmt.Errorf("%w: compute HVS reference digest: %v", liborders.ErrProcessing, err)
	}
	received := normalizeDigestValue(data.CancelledDataDigest.Value)
	if subtle.ConstantTimeCompare(expected, received) != 1 {
		return fmt.Errorf("%w: HVS cancelled digest does not match reference order data", liborders.ErrInvalidOrderData)
	}

	return nil
}

func (p *serverReportingProvider) ResponseRaw(
	req *libebics.OrderContext,
) ([]byte, error) {
	item, _, err := p.selectSingleItem(req)
	if err != nil {
		return nil, err
	}
	if len(item.ResponsePayload) == 0 {
		return nil, fmt.Errorf("%w: no raw payload configured for order %q", liborders.ErrNoDownloadData, item.OrderID)
	}

	return append([]byte(nil), item.ResponsePayload...), nil
}

func (p *serverReportingProvider) buildHVUResponse(
	req *libebics.OrderContext,
) (ebicsxml.HVUResponseOrderData, error) {
	items, err := p.selectMatchingItems(req)
	if err != nil {
		return ebicsxml.HVUResponseOrderData{}, err
	}

	out := ebicsxml.HVUResponseOrderData{
		OrderDetails: make([]ebicsxml.HVUOrderDetails, 0, len(items)),
	}
	for _, item := range items {
		doc, parseErr := ebicsxml.ParseHVUResponseOrderData(item.ResponsePayload)
		if parseErr != nil {
			return ebicsxml.HVUResponseOrderData{}, fmt.Errorf(
				"%w: parse HVU payload for item %q: %v",
				liborders.ErrInvalidOrderData,
				item.ItemKey,
				parseErr,
			)
		}
		out.OrderDetails = append(out.OrderDetails, doc.OrderDetails...)
	}
	if len(out.OrderDetails) == 0 {
		return ebicsxml.HVUResponseOrderData{}, fmt.Errorf("%w: no HVU reporting data available", liborders.ErrNoDownloadData)
	}

	return out, nil
}

func (p *serverReportingProvider) buildHVZResponse(
	req *libebics.OrderContext,
) (ebicsxml.HVZResponseOrderData, error) {
	items, err := p.selectMatchingItems(req)
	if err != nil {
		return ebicsxml.HVZResponseOrderData{}, err
	}

	out := ebicsxml.HVZResponseOrderData{
		OrderDetails: make([]ebicsxml.HVZOrderDetails, 0, len(items)),
	}
	for _, item := range items {
		doc, parseErr := ebicsxml.ParseHVZResponseOrderData(item.ResponsePayload)
		if parseErr != nil {
			return ebicsxml.HVZResponseOrderData{}, fmt.Errorf(
				"%w: parse HVZ payload for item %q: %v",
				liborders.ErrInvalidOrderData,
				item.ItemKey,
				parseErr,
			)
		}
		out.OrderDetails = append(out.OrderDetails, doc.OrderDetails...)
	}
	if len(out.OrderDetails) == 0 {
		return ebicsxml.HVZResponseOrderData{}, fmt.Errorf("%w: no HVZ reporting data available", liborders.ErrNoDownloadData)
	}

	return out, nil
}

func (p *serverReportingProvider) selectSingleItem(
	req *libebics.OrderContext,
) (*model.EbicsServerReportingItem, *ebicsxml.HVTOrderParams, error) {
	items, params, err := p.selectMatchingItemsWithParams(req)
	if err != nil {
		return nil, nil, err
	}
	if len(items) == 0 {
		return nil, nil, fmt.Errorf("%w: no EBICS server reporting data configured", liborders.ErrNoDownloadData)
	}
	if len(items) > 1 {
		return nil, nil, fmt.Errorf(
			"%w: multiple EBICS server reporting items match order %q",
			liborders.ErrProcessing,
			p.order,
		)
	}

	return items[0], params, nil
}

func (p *serverReportingProvider) selectMatchingItems(
	req *libebics.OrderContext,
) ([]*model.EbicsServerReportingItem, error) {
	items, _, err := p.selectMatchingItemsWithParams(req)
	return items, err
}

func (p *serverReportingProvider) selectMatchingItemsWithParams(
	req *libebics.OrderContext,
) ([]*model.EbicsServerReportingItem, *ebicsxml.HVTOrderParams, error) {
	if err := (&serverAdminPolicy{store: p.store}).validateOperationalSubscriber(*req); err != nil {
		return nil, nil, err
	}

	host, err := p.store.getHostByHostID(string(req.HostID))
	if err != nil {
		return nil, nil, err
	}
	subscriber, err := p.store.getSubscriber(string(req.HostID), string(req.PartnerID), string(req.UserID))
	if err != nil {
		return nil, nil, err
	}

	var sets model.EbicsServerReportingSets
	queryErr := p.db.Select(&sets).
		Owner().
		Where(
			"ebics_host_id=? AND ebics_subscriber_id=? AND source_order_type=? AND status=?",
			host.ID,
			subscriber.ID,
			p.order,
			"ACTIVE",
		).
		OrderBy("id", true).
		Run()
	if queryErr != nil {
		return nil, nil, fmt.Errorf("load active EBICS server reporting sets: %w", queryErr)
	}
	if len(sets) == 0 {
		return nil, nil, fmt.Errorf(
			"%w: no active EBICS server reporting set for %s",
			liborders.ErrNoDownloadData,
			p.order,
		)
	}

	items := make([]*model.EbicsServerReportingItem, 0, len(sets))
	for _, set := range sets {
		var rows model.EbicsServerReportingItems
		loadErr := p.db.Select(&rows).
			Owner().
			Where("server_reporting_set_id=? AND is_enabled=?", set.ID, true).
			OrderBy("id", true).
			Run()
		if loadErr != nil {
			return nil, nil, fmt.Errorf(
				"load EBICS server reporting items for set %d: %w",
				set.ID,
				loadErr,
			)
		}
		for _, row := range rows {
			items = append(items, row)
		}
	}

	params, filters, requestOrderID, criteriaErr := p.matchingRequestCriteria(req)
	if criteriaErr != nil {
		return nil, nil, criteriaErr
	}
	effectiveOrderID := strings.TrimSpace(req.OrderID)
	if effectiveOrderID == "" {
		effectiveOrderID = requestOrderID
	}

	filtered := make([]*model.EbicsServerReportingItem, 0, len(items))
	for _, item := range items {
		if !matchesReportingOrderID(item, effectiveOrderID) {
			continue
		}
		if !matchesReportingService(item, filters, p.order) {
			continue
		}
		filtered = append(filtered, item)
	}
	if len(filtered) == 0 && len(items) == 1 {
		filtered = append(filtered, items[0])
	}

	return filtered, params, nil
}

func (p *serverReportingProvider) matchingRequestCriteria(
	req *libebics.OrderContext,
) (*ebicsxml.HVTOrderParams, []serviceFilterRef, string, error) {
	emptyHVTParams := &ebicsxml.HVTOrderParams{}

	switch p.order {
	case serverReportingOrderHVD:
		var params ebicsxml.HVDOrderParams
		if err := xml.Unmarshal(req.OrderParamsXML, &params); err != nil {
			return nil, nil, "", fmt.Errorf("%w: parse HVD order params: %v", liborders.ErrInvalidOrderParams, err)
		}

		return emptyHVTParams, singleRestrictedServiceFilter(&params.Service), strings.TrimSpace(params.OrderID), nil
	case serverReportingOrderHVE:
		var params ebicsxml.HVEOrderParams
		if err := xml.Unmarshal(req.OrderParamsXML, &params); err != nil {
			return nil, nil, "", fmt.Errorf("%w: parse HVE order params: %v", liborders.ErrInvalidOrderParams, err)
		}

		return emptyHVTParams, singleRestrictedServiceFilter(&params.Service), strings.TrimSpace(params.OrderID), nil
	case serverReportingOrderHVS:
		var params ebicsxml.HVSOrderParams
		if err := xml.Unmarshal(req.OrderParamsXML, &params); err != nil {
			return nil, nil, "", fmt.Errorf("%w: parse HVS order params: %v", liborders.ErrInvalidOrderParams, err)
		}

		return emptyHVTParams, singleRestrictedServiceFilter(&params.Service), strings.TrimSpace(params.OrderID), nil
	case serverReportingOrderHVT:
		var params ebicsxml.HVTOrderParams
		if err := xml.Unmarshal(req.OrderParamsXML, &params); err != nil {
			return nil, nil, "", fmt.Errorf("%w: parse HVT order params: %v", liborders.ErrInvalidOrderParams, err)
		}

		return &params, singleRestrictedServiceFilter(&params.Service), strings.TrimSpace(params.OrderID), nil
	case serverReportingOrderHVU:
		if len(req.OrderParamsXML) == 0 {
			return emptyHVTParams, []serviceFilterRef{}, "", nil
		}
		var params ebicsxml.HVUOrderParams
		if err := xml.Unmarshal(req.OrderParamsXML, &params); err != nil {
			return nil, nil, "", fmt.Errorf("%w: parse HVU order params: %v", liborders.ErrInvalidOrderParams, err)
		}

		return emptyHVTParams, serviceFiltersFromParams(params.ServiceFilter), "", nil
	case serverReportingOrderHVZ:
		if len(req.OrderParamsXML) == 0 {
			return emptyHVTParams, []serviceFilterRef{}, "", nil
		}
		var params ebicsxml.HVZOrderParams
		if err := xml.Unmarshal(req.OrderParamsXML, &params); err != nil {
			return nil, nil, "", fmt.Errorf("%w: parse HVZ order params: %v", liborders.ErrInvalidOrderParams, err)
		}

		return emptyHVTParams, serviceFiltersFromParams(params.ServiceFilter), "", nil
	default:
		return emptyHVTParams, []serviceFilterRef{}, "", nil
	}
}

func matchesReportingOrderID(item *model.EbicsServerReportingItem, reqOrderID string) bool {
	if strings.TrimSpace(reqOrderID) == "" {
		return true
	}

	return strings.EqualFold(strings.TrimSpace(item.OrderID), strings.TrimSpace(reqOrderID))
}

func matchesReportingService(
	item *model.EbicsServerReportingItem,
	filters []serviceFilterRef,
	orderType string,
) bool {
	switch orderType {
	case serverReportingOrderHVD, serverReportingOrderHVE, serverReportingOrderHVT, serverReportingOrderHVS:
		if len(filters) == 0 {
			return false
		}

		return serviceFilterFromItem(item) == filters[0]
	case serverReportingOrderHVU, serverReportingOrderHVZ:
		if len(filters) == 0 {
			return true
		}

		itemFilter := serviceFilterFromItem(item)

		return slices.Contains(filters, itemFilter)
	default:
		return true
	}
}

func normalizeDigestValue(raw []byte) []byte {
	trimmed := bytesTrimSpace(raw)
	if len(trimmed) == 0 {
		return nil
	}
	if decoded, err := base64.StdEncoding.DecodeString(string(trimmed)); err == nil {
		return decoded
	}

	return trimmed
}

func digestForReferenceData(raw []byte, signatureVersion string) ([]byte, error) {
	cfg, err := libcrypto.ESProfileConfigFor(libcrypto.ESProfile(strings.TrimSpace(signatureVersion)))
	if err != nil {
		return nil, fmt.Errorf("resolve ES profile for reporting digest: %w", err)
	}
	h := cfg.Hash.New()
	if _, writeErr := h.Write(raw); writeErr != nil {
		return nil, fmt.Errorf("hash reporting reference payload: %w", writeErr)
	}

	return h.Sum(nil), nil
}

func bytesTrimSpace(raw []byte) []byte {
	return []byte(strings.TrimSpace(string(raw)))
}

func singleRestrictedServiceFilter(service *ebicsxml.RestrictedService) []serviceFilterRef {
	return []serviceFilterRef{serviceFilterFromRestricted(service)}
}

type hveHandler struct {
	provider *serverReportingProvider
}

//nolint:gocritic,wrapcheck // Handler contract imposes value semantics and mapped errors must be returned as-is.
func (h hveHandler) Handle(_ context.Context, req libebics.OrderContext) (libebics.OrderResult, error) {
	if h.provider == nil {
		return libebics.OrderResult{}, liborders.MapOrderError(
			fmt.Errorf("%w: hve provider not configured", liborders.ErrProcessing),
		)
	}
	payload, err := h.provider.ResponseRaw(&req)
	if err != nil {
		return libebics.OrderResult{}, liborders.MapOrderError(err)
	}

	return libebics.SuccessResult(payload), nil
}

type serviceFilterRef struct {
	serviceName   string
	serviceOption string
	scope         string
	msgName       string
	containerType string
}

func serviceFilterFromRestricted(service *ebicsxml.RestrictedService) serviceFilterRef {
	out := serviceFilterRef{
		serviceName:   strings.TrimSpace(service.ServiceName),
		serviceOption: strings.TrimSpace(service.ServiceOption),
		scope:         strings.TrimSpace(service.Scope),
		msgName:       strings.TrimSpace(service.MsgName.Value),
	}
	if service.Container != nil {
		out.containerType = strings.TrimSpace(service.Container.ContainerType)
	}

	return out
}

func serviceFilterFromItem(item *model.EbicsServerReportingItem) serviceFilterRef {
	return serviceFilterRef{
		serviceName:   strings.TrimSpace(item.ServiceName),
		serviceOption: strings.TrimSpace(item.ServiceOption),
		scope:         strings.TrimSpace(item.Scope),
		msgName:       strings.TrimSpace(item.MsgName),
		containerType: strings.TrimSpace(item.ContainerType),
	}
}

func serviceFiltersFromParams(filters []ebicsxml.ServiceFilter) []serviceFilterRef {
	if len(filters) == 0 {
		return []serviceFilterRef{}
	}

	out := make([]serviceFilterRef, 0, len(filters))
	for _, filter := range filters {
		ref := serviceFilterRef{
			serviceName:   strings.TrimSpace(filter.ServiceName),
			serviceOption: strings.TrimSpace(filter.ServiceOption),
			scope:         strings.TrimSpace(filter.Scope),
		}
		if filter.MsgName != nil {
			ref.msgName = strings.TrimSpace(filter.MsgName.Value)
		}
		if filter.Container != nil {
			ref.containerType = strings.TrimSpace(filter.Container.ContainerType)
		}
		out = append(out, ref)
	}

	return out
}

var (
	_ liborders.HVDProvider              = (*serverReportingProvider)(nil)
	_ liborders.HVSProvider              = (*serverReportingProvider)(nil)
	_ liborders.HVUProvider              = hvuProviderAdapter{}
	_ liborders.HVZProvider              = hvzProviderAdapter{}
	_ liborders.HVTProvider              = hvtProviderAdapter{}
	_ liborders.HVTOriginalDataProvider  = (*serverReportingProvider)(nil)
	_ liborders.HACDocumentProvider      = (*serverReportingProvider)(nil)
	_ libserver.EDSReferenceDataProvider = (*serverReportingProvider)(nil)
)

type hvuProviderAdapter struct{ *serverReportingProvider }

//nolint:gocritic // lib-ebics provider interface imposes value semantics here.
func (a hvuProviderAdapter) Response(
	ctx context.Context,
	req libebics.OrderContext,
) (ebicsxml.HVUResponseOrderData, error) {
	return a.ResponseHVU(ctx, req)
}

type hvzProviderAdapter struct{ *serverReportingProvider }

//nolint:gocritic // lib-ebics provider interface imposes value semantics here.
func (a hvzProviderAdapter) Response(
	ctx context.Context,
	req libebics.OrderContext,
) (ebicsxml.HVZResponseOrderData, error) {
	return a.ResponseHVZ(ctx, req)
}

type hvtProviderAdapter struct{ *serverReportingProvider }

//nolint:gocritic // lib-ebics provider interface imposes value semantics here.
func (a hvtProviderAdapter) Response(
	ctx context.Context,
	req libebics.OrderContext,
) (ebicsxml.HVTResponseOrderData, error) {
	return a.ResponseHVT(ctx, req)
}
