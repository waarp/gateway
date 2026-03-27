package runtime

import (
	"testing"

	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

type fakeContractValidationResolver struct {
	view            *model.EbicsContractView
	items           []model.EbicsContractViewItem
	standardByScope map[string]fakeStandardCatalogResult
}

type fakeStandardCatalogResult struct {
	catalog *model.EbicsStandardBTFCatalog
	entries []model.EbicsStandardBTFEntry
}

func (r *fakeContractValidationResolver) GetActiveContractView(
	owner string,
	hostID string,
	partnerID string,
	userID string,
) (*model.EbicsContractView, []model.EbicsContractViewItem, error) {
	return r.view, r.items, nil
}

func (r *fakeContractValidationResolver) GetActiveStandardBTFCatalog(
	owner string,
	scope string,
) (*model.EbicsStandardBTFCatalog, []model.EbicsStandardBTFEntry, error) {
	if r.standardByScope == nil {
		return nil, nil, nil
	}

	result, ok := r.standardByScope[scope]
	if !ok {
		return nil, nil, nil
	}

	return result.catalog, result.entries, nil
}

func TestValidateResolvedPayloadRequestSpecificContractOverridesStandard(t *testing.T) {
	t.Parallel()

	request := testResolvedPayloadRequest()
	resolver := &fakeContractValidationResolver{
		view: &model.EbicsContractView{ID: 42},
		items: []model.EbicsContractViewItem{{
			ID:          51,
			ItemType:    "BTF",
			OrderType:   "BTU",
			ServiceName: "REP",
			Scope:       "GLB",
			MsgName:     "camt.053",
			IsEnabled:   true,
		}},
		standardByScope: map[string]fakeStandardCatalogResult{
			"GLB": {
				catalog: &model.EbicsStandardBTFCatalog{ID: 12, Scope: "GLB"},
				entries: []model.EbicsStandardBTFEntry{{
					ID:          13,
					OrderType:   "BTU",
					Direction:   "UPLOAD",
					ServiceName: "SCT",
					Scope:       "GLB",
					MsgName:     "pain.001",
					Status:      "ACTIVE",
				}},
			},
		},
	}

	result, err := ValidateResolvedPayloadRequest("gw", request, resolver)
	require.NoError(t, err)
	require.Equal(t, contractValidationNoMatchingItem, result.Status)
	require.Equal(t, contractValidationSourceSpecific, result.ValidationSource)
	require.Equal(t, int64(42), result.ContractViewID)
	require.Zero(t, result.StandardCatalogID)
}

func TestValidateResolvedPayloadRequestFallsBackToCountryCatalog(t *testing.T) {
	t.Parallel()

	request := testResolvedPayloadRequest()
	request.ResolvedService.Scope = "FR"

	resolver := &fakeContractValidationResolver{
		standardByScope: map[string]fakeStandardCatalogResult{
			"FR": {
				catalog: &model.EbicsStandardBTFCatalog{ID: 21, Scope: "FR"},
				entries: []model.EbicsStandardBTFEntry{{
					ID:          22,
					EntryKey:    "FR:SCT:pain.001",
					OrderType:   "BTU",
					Direction:   "UPLOAD",
					ServiceName: "SCT",
					Scope:       "FR",
					MsgName:     "pain.001",
					Status:      "ACTIVE",
				}},
			},
		},
	}

	result, err := ValidateResolvedPayloadRequest("gw", request, resolver)
	require.NoError(t, err)
	require.Equal(t, contractValidationMatched, result.Status)
	require.Equal(t, contractValidationSourceStandardNat, result.ValidationSource)
	require.Equal(t, int64(21), result.StandardCatalogID)
	require.Equal(t, []int64{22}, request.StandardEntryIDs)
}

func TestValidateResolvedPayloadRequestFallsBackToGLB(t *testing.T) {
	t.Parallel()

	request := testResolvedPayloadRequest()
	request.ResolvedService.Scope = "FR"

	resolver := &fakeContractValidationResolver{
		standardByScope: map[string]fakeStandardCatalogResult{
			"GLB": {
				catalog: &model.EbicsStandardBTFCatalog{ID: 31, Scope: "GLB"},
				entries: []model.EbicsStandardBTFEntry{{
					ID:          32,
					EntryKey:    "GLB:SCT:pain.001",
					OrderType:   "BTU",
					Direction:   "UPLOAD",
					ServiceName: "SCT",
					Scope:       "GLB",
					MsgName:     "pain.001",
					Status:      "ACTIVE",
				}},
			},
		},
	}

	result, err := ValidateResolvedPayloadRequest("gw", request, resolver)
	require.NoError(t, err)
	require.Equal(t, contractValidationMatched, result.Status)
	require.Equal(t, contractValidationSourceStandardGLB, result.ValidationSource)
	require.Equal(t, int64(31), result.StandardCatalogID)
	require.Equal(t, []int64{32}, request.StandardEntryIDs)

	request.ResolvedService.Scope = "GLB"
	result, err = ValidateResolvedPayloadRequest("gw", request, resolver)
	require.NoError(t, err)
	require.Equal(t, contractValidationMatched, result.Status)
	require.Equal(t, contractValidationSourceStandardGLB, result.ValidationSource)
	require.Equal(t, int64(31), result.StandardCatalogID)
	require.Equal(t, []int64{32}, request.StandardEntryIDs)
}

func TestValidateResolvedPayloadRequestWithoutAnyBaseline(t *testing.T) {
	t.Parallel()

	result, err := ValidateResolvedPayloadRequest("gw", testResolvedPayloadRequest(), &fakeContractValidationResolver{})
	require.NoError(t, err)
	require.Equal(t, contractValidationNoValidationBase, result.Status)
	require.Equal(t, contractValidationSourceNone, result.ValidationSource)
}

func testResolvedPayloadRequest() *ResolvedPayloadRequest {
	return &ResolvedPayloadRequest{
		OrderType: "BTU",
		Subscriber: PayloadSubscriberRef{
			HostID:    "BANKHOST",
			PartnerID: "PARTNER1",
			UserID:    "USER1",
		},
		ResolvedService: PayloadServiceRef{
			OrderType:   "BTU",
			ServiceName: "SCT",
			Scope:       "GLB",
			MsgName:     "pain.001",
		},
	}
}
