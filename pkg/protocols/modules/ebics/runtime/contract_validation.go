package runtime

import (
	"fmt"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const (
	contractValidationMatched           = "MATCHED"
	contractValidationNoValidationBase  = "NO_VALIDATION_BASE"
	contractValidationNoMatchingItem    = "NO_MATCHING_ITEM"
	contractValidationSourceNone        = "NONE"
	contractValidationSourceSpecific    = "SPECIFIC_CONTRACT"
	contractValidationSourceStandardNat = "STANDARD_COUNTRY_CATALOG"
	contractValidationSourceStandardGLB = "STANDARD_GLB_CATALOG"
)

type ContractValidationResult struct {
	Status                 string
	Message                string
	ValidationSource       string
	ContractViewID         int64
	StandardCatalogID      int64
	MatchedItems           []model.EbicsContractViewItem
	MatchedStandardEntries []model.EbicsStandardBTFEntry
}

type ContractViewResolver interface {
	GetActiveContractView(
		owner string,
		hostID string,
		partnerID string,
		userID string,
	) (*model.EbicsContractView, []model.EbicsContractViewItem, error)
	GetActiveStandardBTFCatalog(
		owner string,
		scope string,
	) (*model.EbicsStandardBTFCatalog, []model.EbicsStandardBTFEntry, error)
}

func ValidateResolvedPayloadRequest(
	owner string,
	request *ResolvedPayloadRequest,
	resolver ContractViewResolver,
) (*ContractValidationResult, error) {
	if request == nil {
		return nil, database.NewValidationError("the resolved EBICS payload request is missing")
	}

	if resolver == nil {
		return nil, database.NewValidationError("the EBICS contract view resolver is missing")
	}

	view, items, err := resolver.GetActiveContractView(
		owner,
		request.Subscriber.HostID,
		request.Subscriber.PartnerID,
		request.Subscriber.UserID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve the active EBICS contract view: %w", err)
	}

	if view != nil && view.ID != 0 {
		matchedItems := matchContractItems(request, items)
		if len(matchedItems) == 0 {
			return &ContractValidationResult{
				Status:           contractValidationNoMatchingItem,
				Message:          "the resolved payload request is not authorized by the active EBICS contract",
				ValidationSource: contractValidationSourceSpecific,
				ContractViewID:   view.ID,
			}, nil
		}

		request.ContractViewID = view.ID
		request.ContractItemIDs = make([]int64, 0, len(matchedItems))
		for i := range matchedItems {
			request.ContractItemIDs = append(request.ContractItemIDs, matchedItems[i].ID)
		}

		return &ContractValidationResult{
			Status:           contractValidationMatched,
			Message:          "the resolved payload request matches the active EBICS contract",
			ValidationSource: contractValidationSourceSpecific,
			ContractViewID:   view.ID,
			MatchedItems:     matchedItems,
		}, nil
	}

	return validateAgainstStandardCatalogs(owner, request, resolver)
}

func validateAgainstStandardCatalogs(
	owner string,
	request *ResolvedPayloadRequest,
	resolver ContractViewResolver,
) (*ContractValidationResult, error) {
	scopes := buildStandardCatalogLookupScopes(request.ResolvedService.Scope)
	attemptedCatalog := false
	lastAttemptedScope := ""

	for _, scope := range scopes {
		catalog, entries, err := resolver.GetActiveStandardBTFCatalog(owner, scope)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve the active EBICS standard BTF catalog for scope %q: %w", scope, err)
		}
		if catalog == nil || catalog.ID == 0 {
			continue
		}

		attemptedCatalog = true
		lastAttemptedScope = scope
		matchedEntries := matchStandardEntries(request, entries)
		if len(matchedEntries) == 0 {
			continue
		}

		request.StandardCatalogID = catalog.ID
		request.StandardEntryIDs = make([]int64, 0, len(matchedEntries))
		for i := range matchedEntries {
			request.StandardEntryIDs = append(request.StandardEntryIDs, matchedEntries[i].ID)
		}

		return &ContractValidationResult{
			Status:                 contractValidationMatched,
			Message:                standardCatalogMatchMessage(scope),
			ValidationSource:       validationSourceForStandardScope(scope),
			StandardCatalogID:      catalog.ID,
			MatchedStandardEntries: matchedEntries,
		}, nil
	}

	if attemptedCatalog {
		return &ContractValidationResult{
			Status:           contractValidationNoMatchingItem,
			Message:          "the resolved payload request is not allowed by the active standard EBICS BTF catalog",
			ValidationSource: validationSourceForStandardScope(lastAttemptedScope),
		}, nil
	}

	return &ContractValidationResult{
		Status:           contractValidationNoValidationBase,
		Message:          "no active EBICS contract view or standard BTF catalog found for the selected subscriber",
		ValidationSource: contractValidationSourceNone,
	}, nil
}

func matchContractItems(
	request *ResolvedPayloadRequest,
	items []model.EbicsContractViewItem,
) []model.EbicsContractViewItem {
	matched := make([]model.EbicsContractViewItem, 0, len(items))

	for i := range items {
		if isContractItemCompatible(request, &items[i]) {
			matched = append(matched, items[i])
		}
	}

	return matched
}

func matchStandardEntries(
	request *ResolvedPayloadRequest,
	entries []model.EbicsStandardBTFEntry,
) []model.EbicsStandardBTFEntry {
	matched := make([]model.EbicsStandardBTFEntry, 0, len(entries))

	for i := range entries {
		if isStandardEntryCompatible(request, &entries[i]) {
			matched = append(matched, entries[i])
		}
	}

	return matched
}

func isContractItemCompatible(
	request *ResolvedPayloadRequest,
	item *model.EbicsContractViewItem,
) bool {
	if !item.IsEnabled {
		return false
	}

	switch item.ItemType {
	case "ORDER_TYPE", "BTF":
	default:
		return false
	}

	if model.NormalizeEbicsPayloadOrderType(item.OrderType) !=
		model.NormalizeEbicsPayloadOrderType(request.OrderType) {
		return false
	}

	if item.ServiceName != "" && item.ServiceName != request.ResolvedService.ServiceName {
		return false
	}

	if item.ServiceOption != "" && item.ServiceOption != request.ResolvedService.ServiceOption {
		return false
	}

	if item.Scope != "" && item.Scope != request.ResolvedService.Scope {
		return false
	}

	if item.MsgName != "" && item.MsgName != request.ResolvedService.MsgName {
		return false
	}

	if item.ContainerType != "" && item.ContainerType != request.ResolvedService.ContainerType {
		return false
	}

	return true
}

func isStandardEntryCompatible(
	request *ResolvedPayloadRequest,
	entry *model.EbicsStandardBTFEntry,
) bool {
	if entry.Status != "ACTIVE" {
		return false
	}
	if model.NormalizeEbicsPayloadOrderType(entry.OrderType) !=
		model.NormalizeEbicsPayloadOrderType(request.OrderType) {
		return false
	}
	if entry.ServiceName != "" && entry.ServiceName != request.ResolvedService.ServiceName {
		return false
	}
	if entry.ServiceOption != "" && entry.ServiceOption != request.ResolvedService.ServiceOption {
		return false
	}
	if entry.Scope != "" &&
		entry.Scope != request.ResolvedService.Scope &&
		entry.Scope != model.EbicsStandardBTFScopeGLB {
		return false
	}
	if entry.MsgName != "" && entry.MsgName != request.ResolvedService.MsgName {
		return false
	}
	if entry.ContainerType != "" && entry.ContainerType != request.ResolvedService.ContainerType {
		return false
	}

	return true
}

func buildStandardCatalogLookupScopes(scope string) []string {
	normalized := strings.ToUpper(strings.TrimSpace(scope))
	if normalized == "" || normalized == model.EbicsStandardBTFScopeGLB {
		return []string{model.EbicsStandardBTFScopeGLB}
	}

	return []string{normalized, model.EbicsStandardBTFScopeGLB}
}

func validationSourceForStandardScope(scope string) string {
	if strings.ToUpper(strings.TrimSpace(scope)) == model.EbicsStandardBTFScopeGLB {
		return contractValidationSourceStandardGLB
	}

	return contractValidationSourceStandardNat
}

func standardCatalogMatchMessage(scope string) string {
	if strings.ToUpper(strings.TrimSpace(scope)) == model.EbicsStandardBTFScopeGLB {
		return "the resolved payload request matches the active standard EBICS GLB catalog"
	}

	return fmt.Sprintf("the resolved payload request matches the active standard EBICS catalog for scope %q", scope)
}
