package ebics

import (
	"context"
	"crypto/sha256"
	"testing"

	libebics "code.waarp.fr/lib/ebics/ebics"
	liborders "code.waarp.fr/lib/ebics/ebics/orders"
	ebicsxml "code.waarp.fr/lib/ebics/ebics/xml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
)

func TestServerReportingProviderHACDocumentProvider(t *testing.T) {
	setEBICSConfigChecker(t)

	db := dbtest.TestDatabase(t)

	agent := &model.LocalAgent{
		Name:     "ebics-hac-server",
		Protocol: EBICS,
		Address:  types.Addr("localhost", 1),
	}
	require.NoError(t, db.Insert(agent).Run())
	account := insertTestLocalAccount(t, db, agent.ID, "ebics-hac-account")
	host := insertTestEbicsHost(t, db, "HOST-HAC")
	subscriber := insertTestEbicsSubscriber(t, db, host.ID, "PARTNER-HAC", "USER-HAC", true)
	subscriber.LocalAccountID = account.GetNullID()
	require.NoError(t, db.Update(subscriber).Run())

	payload, err := ebicsxml.BuildHACDocument(ebicsxml.HACDocument{
		CustomerReport: ebicsxml.HACCustomerPaymentStatusV3{
			GroupHeader: ebicsxml.HACGroupHeader{
				MessageID: "MSG-HAC",
				CreatedAt: "2026-04-03T10:00:00Z",
				InitiatingPt: ebicsxml.HACPartyInfo{
					ID: ebicsxml.HACPartyIDChoice{
						OrgID: ebicsxml.HACOrgID{
							Others: []ebicsxml.HACOtherOrgID{{ID: host.HostID}},
						},
					},
				},
			},
			OriginalGroupInfo: ebicsxml.HACOriginalGroupInformation{
				OriginalMessageID:   "EBICS",
				OriginalMessageName: "EBICS",
			},
			OriginalPaymentInfos: []ebicsxml.HACOriginalPaymentInfo{{
				ActionType: "FILE_UPLOAD",
				StatusReasonInfos: []ebicsxml.HACStatusReason{{
					Reason: ebicsxml.HACStatusReasonChoice{Code: "AM05"},
				}},
			}},
		},
	})
	require.NoError(t, err)

	insertTestActiveServerReportingSet(t, db, host.ID, subscriber.ID, "HAC",
		&model.EbicsServerReportingItem{
			ItemKey:         "HAC:OID-HAC-1",
			OrderID:         "OID-HAC-1",
			ResponsePayload: payload,
			IsEnabled:       true,
		},
	)

	provider := newServerReportingProvider(db, newProviderStore(db), "HAC")
	handler := liborders.HACHandler{DocumentProvider: provider}

	res, err := handler.Handle(context.Background(), libebics.OrderContext{
		HostID:    libebics.HostID(host.HostID),
		PartnerID: libebics.PartnerID(subscriber.PartnerID),
		UserID:    libebics.UserID(subscriber.UserID),
		OrderType: "HAC",
		OrderID:   "OID-HAC-1",
	})
	require.NoError(t, err)
	require.NotEmpty(t, res.PayloadRaw)

	doc, err := ebicsxml.ParseHACDocument(res.PayloadRaw)
	require.NoError(t, err)
	assert.Equal(t, "MSG-HAC", doc.CustomerReport.GroupHeader.MessageID)
	assert.Equal(t, "FILE_UPLOAD", doc.CustomerReport.OriginalPaymentInfos[0].ActionType)
}

func TestServerReportingProviderHVEAndHVS(t *testing.T) {
	setEBICSConfigChecker(t)

	db := dbtest.TestDatabase(t)

	agent := &model.LocalAgent{
		Name:     "ebics-hve-server",
		Protocol: EBICS,
		Address:  types.Addr("localhost", 1),
	}
	require.NoError(t, db.Insert(agent).Run())
	account := insertTestLocalAccount(t, db, agent.ID, "ebics-hve-account")
	host := insertTestEbicsHost(t, db, "HOST-HVE")
	subscriber := insertTestEbicsSubscriber(t, db, host.ID, "PARTNER-HVE", "USER-HVE", true)
	subscriber.LocalAccountID = account.GetNullID()
	require.NoError(t, db.Update(subscriber).Run())

	referenceData := []byte("ORIGINAL-ORDER-DATA")
	signaturePayload := []byte("SIGNATURE-PAYLOAD")
	insertTestActiveServerReportingSet(t, db, host.ID, subscriber.ID, "HVE",
		&model.EbicsServerReportingItem{
			ItemKey:         "HVE:OID-HVE-1",
			OrderID:         "OID-HVE-1",
			ServiceName:     "MCT",
			Scope:           "FR",
			MsgName:         "pain.001",
			ResponsePayload: signaturePayload,
			OriginalPayload: referenceData,
			IsEnabled:       true,
		},
	)
	insertTestActiveServerReportingSet(t, db, host.ID, subscriber.ID, "HVS",
		&model.EbicsServerReportingItem{
			ItemKey:         "HVS:OID-HVE-1",
			OrderID:         "OID-HVE-1",
			ServiceName:     "MCT",
			Scope:           "FR",
			MsgName:         "pain.001",
			ResponsePayload: []byte("unused-hvs-payload"),
			OriginalPayload: referenceData,
			IsEnabled:       true,
		},
	)

	req := libebics.OrderContext{
		HostID:    libebics.HostID(host.HostID),
		PartnerID: libebics.PartnerID(subscriber.PartnerID),
		UserID:    libebics.UserID(subscriber.UserID),
		OrderType: "HVE",
		OrderParamsXML: []byte(`<HVEOrderParams xmlns="urn:org:ebics:H005"><PartnerID>` +
			subscriber.PartnerID + `</PartnerID><Service><ServiceName>MCT</ServiceName><Scope>FR</Scope><MsgName>pain.001</MsgName></Service><OrderID>OID-HVE-1</OrderID></HVEOrderParams>`),
	}

	provider := newServerReportingProvider(db, newProviderStore(db), "HVE")
	payload, err := provider.ResponseRaw(&req)
	require.NoError(t, err)
	assert.Equal(t, signaturePayload, payload)

	ref, err := provider.EDSReferenceData(context.Background(), req, ebicsxml.OrderSignatureData{})
	require.NoError(t, err)
	assert.Equal(t, referenceData, ref)

	digest := sha256.Sum256(referenceData)
	hvsProvider := newServerReportingProvider(db, newProviderStore(db), "HVS")
	err = hvsProvider.Cancel(context.Background(), libebics.OrderContext{
		HostID:    libebics.HostID(host.HostID),
		PartnerID: libebics.PartnerID(subscriber.PartnerID),
		UserID:    libebics.UserID(subscriber.UserID),
		OrderType: "HVS",
		OrderParamsXML: []byte(`<HVSOrderParams xmlns="urn:org:ebics:H005"><PartnerID>` +
			subscriber.PartnerID + `</PartnerID><Service><ServiceName>MCT</ServiceName><Scope>FR</Scope><MsgName>pain.001</MsgName></Service><OrderID>OID-HVE-1</OrderID></HVSOrderParams>`),
	}, ebicsxml.HVSRequestOrderData{
		CancelledDataDigest: ebicsxml.DataDigest{
			Value:            digest[:],
			SignatureVersion: "A005",
		},
	})
	require.NoError(t, err)

	err = hvsProvider.Cancel(context.Background(), libebics.OrderContext{
		HostID:    libebics.HostID(host.HostID),
		PartnerID: libebics.PartnerID(subscriber.PartnerID),
		UserID:    libebics.UserID(subscriber.UserID),
		OrderType: "HVS",
		OrderParamsXML: []byte(`<HVSOrderParams xmlns="urn:org:ebics:H005"><PartnerID>` +
			subscriber.PartnerID + `</PartnerID><Service><ServiceName>MCT</ServiceName><Scope>FR</Scope><MsgName>pain.001</MsgName></Service><OrderID>OID-HVE-1</OrderID></HVSOrderParams>`),
	}, ebicsxml.HVSRequestOrderData{
		CancelledDataDigest: ebicsxml.DataDigest{
			Value:            []byte("wrong"),
			SignatureVersion: "A005",
		},
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, liborders.ErrInvalidOrderData)
}
