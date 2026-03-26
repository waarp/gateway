package runtime

import (
	"fmt"
	"strings"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

const (
	contractValidationMatched          = "MATCHED"
	contractValidationNoActiveContract = "NO_ACTIVE_CONTRACT"
	contractValidationNoMatchingItem   = "NO_MATCHING_ITEM"
)

type ContractValidationResult struct {
	Status         string
	Message        string
	ContractViewID int64
	MatchedItems   []model.EbicsContractViewItem
}

type ContractViewResolver interface {
	GetActiveContractView(
		owner string,
		hostID string,
		partnerID string,
		userID string,
	) (*model.EbicsContractView, []model.EbicsContractViewItem, error)
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

	if view == nil {
		return &ContractValidationResult{
			Status:  contractValidationNoActiveContract,
			Message: "no active EBICS contract view found for the selected subscriber",
		}, nil
	}

	matchedItems := matchContractItems(request, items)
	if len(matchedItems) == 0 {
		return &ContractValidationResult{
			Status:         contractValidationNoMatchingItem,
			Message:        "the resolved payload request does not match the active EBICS contract",
			ContractViewID: view.ID,
		}, nil
	}

	request.ContractViewID = view.ID
	request.ContractItemIDs = make([]int64, 0, len(matchedItems))
	for i := range matchedItems {
		request.ContractItemIDs = append(request.ContractItemIDs, matchedItems[i].ID)
	}

	return &ContractValidationResult{
		Status:         contractValidationMatched,
		Message:        "the resolved payload request matches the active EBICS contract",
		ContractViewID: view.ID,
		MatchedItems:   matchedItems,
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

	if strings.ToUpper(strings.TrimSpace(item.OrderType)) != request.OrderType {
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
