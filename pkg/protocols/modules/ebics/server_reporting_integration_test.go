package ebics

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"io"
	"net/http"
	"testing"
	"time"

	libebics "code.waarp.fr/lib/ebics/ebics"
	libebicsclient "code.waarp.fr/lib/ebics/ebics/client"
	libebicscrypto "code.waarp.fr/lib/ebics/ebics/crypto"
	ebicsxml "code.waarp.fr/lib/ebics/ebics/xml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/database"
	"code.waarp.fr/apps/gateway/gateway/pkg/database/dbtest"
	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils/testhelpers"
)

func TestServerHTTPReportingOrdersServeConfiguredSnapshots(t *testing.T) {
	setEBICSConfigChecker(t)

	serverDB := dbtest.TestDatabase(t)
	serverRoot := t.TempDir()
	serverService, serverAccount := startTestEBICSIntegrationServer(t, serverDB, serverRoot)
	serverURL := "https://" + serverService.server.Address.Host + ":" + utils.FormatUint(serverService.server.Address.Port)

	serverAccount.Login = "foo"
	require.NoError(t, serverDB.Update(serverAccount).Run())
	insertTrustedTLSCredentialForLocalAccount(t, serverDB, serverAccount.ID, "client-foo-trust", testhelpers.ClientFooCert)

	serverHost := insertTestEbicsHost(t, serverDB, "HOST-REPORTING")
	serverHost.DefaultBankURL = serverURL
	require.NoError(t, serverDB.Update(serverHost).Run())

	serverSubscriber := insertTestEbicsSubscriber(t, serverDB, serverHost.ID, "PARTNER-REPORTING", "USER-REPORTING", true)
	serverSubscriber.LocalAccountID = utils.NewNullInt64(serverAccount.ID)
	require.NoError(t, serverDB.Update(serverSubscriber).Run())
	insertSubscriberKeyMaterial(
		t,
		serverDB,
		serverSubscriber.ID,
		model.EbicsKeyUsageAuthenticationForRuntime(),
		"X002",
		testhelpers.ClientFooCert,
	)
	insertSubscriberKeyMaterial(
		t,
		serverDB,
		serverSubscriber.ID,
		model.EbicsKeyUsageSignatureForRuntime(),
		"A006",
		testhelpers.ClientFooCert,
	)

	hvdPayload, err := ebicsxml.BuildHVDResponseOrderData(ebicsxml.HVDResponseOrderData{
		DataDigest:            ebicsxml.DataDigest{Value: []byte("digest-hvd"), SignatureVersion: "A006"},
		DisplayFile:           []byte("display-hvd"),
		OrderDataAvailable:    true,
		OrderDataSize:         321,
		OrderDetailsAvailable: false,
		SignerInfo: []ebicsxml.SignerInfo{{
			PartnerID: "PARTNER-REPORTING",
			UserID:    "USER-REPORTING",
			Timestamp: "2026-04-03T10:00:00Z",
			Permission: ebicsxml.SignerPermission{
				AuthorisationLevel: "A",
			},
		}},
	})
	require.NoError(t, err)
	hvuPayload, err := ebicsxml.BuildHVUResponseOrderData(ebicsxml.HVUResponseOrderData{
		OrderDetails: []ebicsxml.HVUOrderDetails{{
			Service: ebicsxml.RestrictedService{
				ServiceName: "MCT",
				Scope:       "FR",
				MsgName:     ebicsxml.MessageType{Value: "pain.001"},
			},
			OrderID:       "OID-HVU-1",
			OrderDataSize: 456,
			SigningInfo: ebicsxml.HVUSigningInfo{
				ReadyToBeSigned: true,
				NumSigRequired:  2,
				NumSigDone:      1,
			},
			OriginatorInfo: ebicsxml.HVUOriginatorInfo{
				PartnerID: "PARTNER-REPORTING",
				UserID:    "USER-REPORTING",
				Timestamp: "2026-04-03T10:00:00Z",
			},
		}},
	})
	require.NoError(t, err)
	hvzPayload, err := ebicsxml.BuildHVZResponseOrderData(ebicsxml.HVZResponseOrderData{
		OrderDetails: []ebicsxml.HVZOrderDetails{{
			Service: ebicsxml.RestrictedService{
				ServiceName: "MCT",
				Scope:       "FR",
				MsgName:     ebicsxml.MessageType{Value: "pain.001"},
			},
			OrderID:               "OID-HVZ-1",
			DataDigest:            ebicsxml.DataDigest{Value: []byte("digest-hvz"), SignatureVersion: "A006"},
			OrderDataAvailable:    true,
			OrderDataSize:         654,
			OrderDetailsAvailable: true,
			SigningInfo: ebicsxml.HVUSigningInfo{
				ReadyToBeSigned: false,
				NumSigRequired:  1,
				NumSigDone:      1,
			},
			OriginatorInfo: ebicsxml.HVUOriginatorInfo{
				PartnerID: "PARTNER-REPORTING",
				UserID:    "USER-REPORTING",
				Timestamp: "2026-04-03T10:00:00Z",
			},
		}},
	})
	require.NoError(t, err)
	isCredit := true
	hvtPayload, err := ebicsxml.BuildHVTResponseOrderData(ebicsxml.HVTResponseOrderData{
		NumOrderInfos: 1,
		OrderInfo: []ebicsxml.HVTOrderInfo{{
			MsgName: &ebicsxml.MessageType{Value: "pain.001"},
			Amount:  ebicsxml.HVTAmount{Value: "123.45", Currency: "EUR", IsCredit: &isCredit},
		}},
	})
	require.NoError(t, err)
	hacPayload, err := ebicsxml.BuildHACDocument(ebicsxml.HACDocument{
		CustomerReport: ebicsxml.HACCustomerPaymentStatusV3{
			GroupHeader: ebicsxml.HACGroupHeader{
				MessageID: "MSG-HAC",
				CreatedAt: "2026-04-03T10:00:00Z",
				InitiatingPt: ebicsxml.HACPartyInfo{
					ID: ebicsxml.HACPartyIDChoice{
						OrgID: ebicsxml.HACOrgID{
							Others: []ebicsxml.HACOtherOrgID{{ID: serverHost.HostID}},
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
	hveResponsePayload := []byte("SIGNED-ORDER-PAYLOAD")
	hveReferencePayload := []byte("ORIGINAL-HVE-ORDER-DATA")

	insertTestActiveServerReportingSet(t, serverDB, serverHost.ID, serverSubscriber.ID, "HVD",
		&model.EbicsServerReportingItem{
			ItemKey:         "HVD:OID-HVD-1",
			OrderID:         "OID-HVD-1",
			ServiceName:     "MCT",
			Scope:           "FR",
			MsgName:         "pain.001",
			ResponsePayload: hvdPayload,
			IsEnabled:       true,
		},
	)
	insertTestActiveServerReportingSet(t, serverDB, serverHost.ID, serverSubscriber.ID, "HVU",
		&model.EbicsServerReportingItem{
			ItemKey:         "HVU:OID-HVU-1",
			OrderID:         "OID-HVU-1",
			ServiceName:     "MCT",
			Scope:           "FR",
			MsgName:         "pain.001",
			ResponsePayload: hvuPayload,
			IsEnabled:       true,
		},
	)
	insertTestActiveServerReportingSet(t, serverDB, serverHost.ID, serverSubscriber.ID, "HVZ",
		&model.EbicsServerReportingItem{
			ItemKey:         "HVZ:OID-HVZ-1",
			OrderID:         "OID-HVZ-1",
			ServiceName:     "MCT",
			Scope:           "FR",
			MsgName:         "pain.001",
			ResponsePayload: hvzPayload,
			IsEnabled:       true,
		},
	)
	insertTestActiveServerReportingSet(t, serverDB, serverHost.ID, serverSubscriber.ID, "HVT",
		&model.EbicsServerReportingItem{
			ItemKey:         "HVT:OID-HVT-1",
			OrderID:         "OID-HVT-1",
			ServiceName:     "MCT",
			Scope:           "FR",
			MsgName:         "pain.001",
			ResponsePayload: hvtPayload,
			OriginalPayload: []byte("ORIGINAL-HVT-PAYLOAD"),
			IsEnabled:       true,
		},
	)
	insertTestActiveServerReportingSet(t, serverDB, serverHost.ID, serverSubscriber.ID, "HAC",
		&model.EbicsServerReportingItem{
			ItemKey:         "HAC:OID-HAC-1",
			OrderID:         "OID-HAC-1",
			ResponsePayload: hacPayload,
			IsEnabled:       true,
		},
	)
	insertTestActiveServerReportingSet(t, serverDB, serverHost.ID, serverSubscriber.ID, "HVE",
		&model.EbicsServerReportingItem{
			ItemKey:         "HVE:OID-HVE-1",
			OrderID:         "OID-HVE-1",
			ServiceName:     "MCT",
			Scope:           "FR",
			MsgName:         "pain.001",
			ResponsePayload: hveResponsePayload,
			OriginalPayload: hveReferencePayload,
			IsEnabled:       true,
		},
	)
	insertTestActiveServerReportingSet(t, serverDB, serverHost.ID, serverSubscriber.ID, "HVS",
		&model.EbicsServerReportingItem{
			ItemKey:         "HVS:OID-HVE-1",
			OrderID:         "OID-HVE-1",
			ServiceName:     "MCT",
			Scope:           "FR",
			MsgName:         "pain.001",
			ResponsePayload: []byte("unused-hvs-payload"),
			OriginalPayload: hveReferencePayload,
			IsEnabled:       true,
		},
	)

	clientDB := startSecondaryTestDatabase(t, "server_reporting_orders_client")
	clientModel := &model.Client{
		Name:     "ebics-reporting-client",
		Protocol: EBICS,
		ProtoConfig: map[string]any{
			"endpointURL":    serverURL,
			"verifyBankKeys": false,
		},
	}
	require.NoError(t, clientDB.Insert(clientModel).Run())

	clientHost := insertTestEbicsHost(t, clientDB, "HOST-REPORTING")
	clientHost.DefaultBankURL = serverURL
	require.NoError(t, clientDB.Update(clientHost).Run())
	insertValidatedBankKey(t, clientDB, clientHost.ID, "AUTH", "X002", testhelpers.LocalhostCert)
	insertValidatedBankKey(t, clientDB, clientHost.ID, "ENCRYPT", "E002", testhelpers.LocalhostCert)
	insertValidatedBankKey(t, clientDB, clientHost.ID, "SIGNATURE", "A006", testhelpers.LocalhostCert)

	remoteAccount := insertTestRTNRemoteAccountForURL(t, clientDB, "reporting-account", serverURL)
	insertTrustedTLSCredential(t, clientDB, remoteAccount.RemoteAgentID, "reporting-trust", testhelpers.LocalhostCert)
	remoteAgent := &model.RemoteAgent{}
	require.NoError(t, clientDB.Get(remoteAgent, "id=?", remoteAccount.RemoteAgentID).Run())

	clientSubscriber := insertTestEbicsSubscriber(t, clientDB, clientHost.ID, "PARTNER-REPORTING", "USER-REPORTING", true)
	clientSubscriber.RemoteAccountID = utils.NewNullInt64(remoteAccount.ID)
	require.NoError(t, clientDB.Update(clientSubscriber).Run())

	authCred := insertTLSCredentialForRemoteAccount(
		t,
		clientDB,
		remoteAccount.ID,
		"ebics-auth-reporting",
		testhelpers.ClientFooCert,
		testhelpers.ClientFooKey,
	)
	encCred := insertTLSCredentialForRemoteAccount(
		t,
		clientDB,
		remoteAccount.ID,
		"ebics-enc-reporting",
		testhelpers.ClientFooCert2,
		testhelpers.ClientFooKey2,
	)
	signatureCred := insertTLSCredentialForRemoteAccount(
		t,
		clientDB,
		remoteAccount.ID,
		"ebics-signature-reporting",
		testhelpers.ClientFooCert,
		testhelpers.ClientFooKey,
	)
	insertActiveLifecycle(t, clientDB, clientSubscriber.ID, model.EbicsKeyUsageSignatureForRuntime(), signatureCred.ID)
	insertActiveLifecycle(t, clientDB, clientSubscriber.ID, model.EbicsKeyUsageAuthenticationForRuntime(), authCred.ID)
	insertActiveLifecycle(t, clientDB, clientSubscriber.ID, model.EbicsKeyUsageEncryptionForRuntime(), encCred.ID)
	signaturePriv, _, _, err := parseCredentialKeyPair(signatureCred)
	require.NoError(t, err)

	clientService := NewClient(clientDB, clientModel)
	require.NoError(t, clientService.Start())
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		_ = clientService.Stop(ctx)
	})

	execCtx, err := clientService.newAdminExecutionContext(clientSubscriber.ID)
	require.NoError(t, err)

	requestCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	hvdDoc, _, err := execCtx.libClient.DownloadHVDDocument(
		requestCtx,
		libebicsclient.FlowHVDRequired{
			URL:       execCtx.endpointURL,
			HostID:    execCtx.host.HostID,
			PartnerID: execCtx.subscriber.PartnerID,
			UserID:    execCtx.subscriber.UserID,
			Params: &ebicsxml.HVDOrderParams{
				HVRequest: ebicsxml.HVRequest{
					PartnerID: execCtx.subscriber.PartnerID,
					Service: ebicsxml.RestrictedService{
						ServiceName: "MCT",
						Scope:       "FR",
						MsgName:     ebicsxml.MessageType{Value: "pain.001"},
					},
					OrderID: "OID-HVD-1",
				},
			},
		},
		libebicsclient.FlowHVDOptional{
			RequestSigner:  execCtx.requestSigner,
			ResponseSigner: execCtx.responseSigner,
			Cipher:         execCtx.downloadCipher,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, 321, hvdDoc.OrderDataSize)
	assert.Equal(t, "A", hvdDoc.SignerInfo[0].Permission.AuthorisationLevel)

	hvuDoc, _, err := execCtx.libClient.DownloadHVUDocument(
		requestCtx,
		libebicsclient.FlowHVURequired{
			URL:       execCtx.endpointURL,
			HostID:    execCtx.host.HostID,
			PartnerID: execCtx.subscriber.PartnerID,
			UserID:    execCtx.subscriber.UserID,
		},
		libebicsclient.FlowHVUOptional{
			Params: &ebicsxml.HVUOrderParams{
				ServiceFilter: []ebicsxml.ServiceFilter{{
					ServiceName: "MCT",
					Scope:       "FR",
					MsgName:     &ebicsxml.MessageType{Value: "pain.001"},
				}},
			},
			RequestSigner:  execCtx.requestSigner,
			ResponseSigner: execCtx.responseSigner,
			Cipher:         execCtx.downloadCipher,
		},
	)
	require.NoError(t, err)
	require.Len(t, hvuDoc.OrderDetails, 1)
	assert.Equal(t, "OID-HVU-1", hvuDoc.OrderDetails[0].OrderID)

	hvzDoc, _, err := execCtx.libClient.DownloadHVZDocument(
		requestCtx,
		libebicsclient.FlowHVZRequired{
			URL:       execCtx.endpointURL,
			HostID:    execCtx.host.HostID,
			PartnerID: execCtx.subscriber.PartnerID,
			UserID:    execCtx.subscriber.UserID,
		},
		libebicsclient.FlowHVZOptional{
			Params: &ebicsxml.HVZOrderParams{
				ServiceFilter: []ebicsxml.ServiceFilter{{
					ServiceName: "MCT",
					Scope:       "FR",
					MsgName:     &ebicsxml.MessageType{Value: "pain.001"},
				}},
			},
			RequestSigner:  execCtx.requestSigner,
			ResponseSigner: execCtx.responseSigner,
			Cipher:         execCtx.downloadCipher,
		},
	)
	require.NoError(t, err)
	require.Len(t, hvzDoc.OrderDetails, 1)
	assert.Equal(t, "OID-HVZ-1", hvzDoc.OrderDetails[0].OrderID)

	hvtDetails, _, err := execCtx.libClient.DownloadHVTDetails(
		requestCtx,
		libebicsclient.FlowHVTRequired{
			URL:       execCtx.endpointURL,
			HostID:    execCtx.host.HostID,
			PartnerID: execCtx.subscriber.PartnerID,
			UserID:    execCtx.subscriber.UserID,
			Params: &ebicsxml.HVTOrderParams{
				HVRequest: ebicsxml.HVRequest{
					PartnerID: execCtx.subscriber.PartnerID,
					Service: ebicsxml.RestrictedService{
						ServiceName: "MCT",
						Scope:       "FR",
						MsgName:     ebicsxml.MessageType{Value: "pain.001"},
					},
					OrderID: "OID-HVT-1",
				},
			},
		},
		libebicsclient.FlowHVTOptional{
			RequestSigner:  execCtx.requestSigner,
			ResponseSigner: execCtx.responseSigner,
			Cipher:         execCtx.downloadCipher,
		},
	)
	require.NoError(t, err)
	require.Len(t, hvtDetails.OrderInfo, 1)
	assert.Equal(t, "123.45", hvtDetails.OrderInfo[0].Amount.Value)

	hvtOriginal, _, err := execCtx.libClient.DownloadHVTOriginal(
		requestCtx,
		libebicsclient.FlowHVTRequired{
			URL:       execCtx.endpointURL,
			HostID:    execCtx.host.HostID,
			PartnerID: execCtx.subscriber.PartnerID,
			UserID:    execCtx.subscriber.UserID,
			Params: &ebicsxml.HVTOrderParams{
				HVRequest: ebicsxml.HVRequest{
					PartnerID: execCtx.subscriber.PartnerID,
					Service: ebicsxml.RestrictedService{
						ServiceName: "MCT",
						Scope:       "FR",
						MsgName:     ebicsxml.MessageType{Value: "pain.001"},
					},
					OrderID: "OID-HVT-1",
				},
				OrderFlags: ebicsxml.HVTOrderFlags{CompleteOrderData: true},
			},
		},
		libebicsclient.FlowHVTOptional{
			RequestSigner:  execCtx.requestSigner,
			ResponseSigner: execCtx.responseSigner,
			Cipher:         execCtx.downloadCipher,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, []byte("ORIGINAL-HVT-PAYLOAD"), hvtOriginal)

	hacHTTPClient, err := clientService.buildAdminHTTPClient(execCtx.endpointURL, remoteAgent, remoteAccount)
	require.NoError(t, err)
	hacTransport := &captureTransport{base: hacHTTPClient}
	hacLibClient, err := libebicsclient.New(libebicsclient.RequiredConfig{Transport: hacTransport})
	require.NoError(t, err)

	hacDoc, _, err := hacLibClient.DownloadHACDocument(
		requestCtx,
		libebicsclient.FlowHACRequired{
			URL:       execCtx.endpointURL,
			HostID:    execCtx.host.HostID,
			PartnerID: execCtx.subscriber.PartnerID,
			UserID:    execCtx.subscriber.UserID,
			OrderID:   "OID-HAC-1",
		},
		libebicsclient.FlowHACOptional{
			RequestSigner:  execCtx.requestSigner,
			ResponseSigner: execCtx.responseSigner,
			Cipher:         execCtx.downloadCipher,
		},
	)
	if err != nil {
		t.Fatalf(
			"captured HAC failure err=%v\nrequest:\n%s\nresponse:\n%s",
			err,
			string(hacTransport.lastRequest),
			string(hacTransport.lastResponse),
		)
	}
	assert.Equal(t, "MSG-HAC", hacDoc.CustomerReport.GroupHeader.MessageID)
	assert.Equal(t, "FILE_UPLOAD", hacDoc.CustomerReport.OriginalPaymentInfos[0].ActionType)

	hvePayload, _, err := execCtx.libClient.DownloadHVE(
		requestCtx,
		libebicsclient.FlowHVERequired{
			URL:       execCtx.endpointURL,
			HostID:    execCtx.host.HostID,
			PartnerID: execCtx.subscriber.PartnerID,
			UserID:    execCtx.subscriber.UserID,
			Params: &ebicsxml.HVEOrderParams{
				HVRequest: ebicsxml.HVRequest{
					PartnerID: execCtx.subscriber.PartnerID,
					Service: ebicsxml.RestrictedService{
						ServiceName: "MCT",
						Scope:       "FR",
						MsgName:     ebicsxml.MessageType{Value: "pain.001"},
					},
					OrderID: "OID-HVE-1",
				},
			},
		},
		libebicsclient.FlowHVEOptional{
			RequestSigner:  execCtx.requestSigner,
			ResponseSigner: execCtx.responseSigner,
			Cipher:         execCtx.downloadCipher,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, hveResponsePayload, hvePayload)

	cancelDigest := mustHashA006(t, hveReferencePayload)
	hvsOrderData := []byte(
		`<HVSRequestOrderData xmlns="urn:org:ebics:H005"><CancelledDataDigest SignatureVersion="A006">` +
			base64.StdEncoding.EncodeToString(cancelDigest) +
			`</CancelledDataDigest></HVSRequestOrderData>`,
	)
	hvsSig, err := libebicsclient.SignEDS(
		libebicscrypto.ESA006,
		signaturePriv,
		cancelDigest,
		execCtx.subscriber.PartnerID,
		execCtx.subscriber.UserID,
		nil,
	)
	require.NoError(t, err)
	require.NoError(t, libebicscrypto.VerifyESHash(
		libebicscrypto.ESA006,
		&signaturePriv.PublicKey,
		cancelDigest,
		hvsSig.SignatureValue,
	))
	serverKeys, err := newProviderStore(serverDB).GetSubscriberKeys(
		context.Background(),
		libebics.HostID(serverHost.HostID),
		libebics.PartnerID(serverSubscriber.PartnerID),
		libebics.UserID(serverSubscriber.UserID),
	)
	require.NoError(t, err)
	serverPub, err := libebicscrypto.ParseRSAPublicKey(serverKeys.SigCertificate)
	require.NoError(t, err)
	require.NoError(t, libebicscrypto.VerifyESHash(
		libebicscrypto.ESA006,
		serverPub,
		cancelDigest,
		hvsSig.SignatureValue,
	))
	hvsSignatureData, err := ebicsxml.MarshalUserSignatureData(ebicsxml.UserSignatureData{
		OrderSignatureData: []ebicsxml.OrderSignatureData{{
			SignatureVersion: hvsSig.SignatureVersion,
			SignatureValue:   ebicsxml.Base64Value(hvsSig.SignatureValue),
			PartnerID:        hvsSig.PartnerID,
			UserID:           hvsSig.UserID,
		}},
	})
	require.NoError(t, err)
	hvsHTTPClient, err := clientService.buildAdminHTTPClient(execCtx.endpointURL, remoteAgent, remoteAccount)
	require.NoError(t, err)
	hvsTransport := &captureTransport{base: hvsHTTPClient}
	hvsLibClient, err := libebicsclient.New(libebicsclient.RequiredConfig{Transport: hvsTransport})
	require.NoError(t, err)
	err = hvsLibClient.UploadHVS(
		requestCtx,
		libebicsclient.FlowHVSRequired{
			URL:       execCtx.endpointURL,
			HostID:    execCtx.host.HostID,
			PartnerID: execCtx.subscriber.PartnerID,
			UserID:    execCtx.subscriber.UserID,
			Params: &ebicsxml.HVSOrderParams{
				HVRequest: ebicsxml.HVRequest{
					PartnerID: execCtx.subscriber.PartnerID,
					Service: ebicsxml.RestrictedService{
						ServiceName: "MCT",
						Scope:       "FR",
						MsgName:     ebicsxml.MessageType{Value: "pain.001"},
					},
					OrderID: "OID-HVE-1",
				},
			},
			OrderData:     hvsOrderData,
			SignatureData: hvsSignatureData,
		},
		libebicsclient.FlowHVSOptional{
			ResponseSigner: execCtx.responseSigner,
		},
	)
	if err != nil {
		t.Fatalf(
			"captured HVS failure err=%v\nrequest:\n%s\nresponse:\n%s",
			err,
			string(hvsTransport.lastRequest),
			string(hvsTransport.lastResponse),
		)
	}
}

type captureTransport struct {
	base         *http.Client
	lastRequest  []byte
	lastResponse []byte
}

func (t *captureTransport) Do(req *http.Request) (*http.Response, error) {
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		t.lastRequest = append([]byte(nil), body...)
		req.Body = io.NopCloser(bytes.NewReader(body))
	}

	resp, err := t.base.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.Body == nil {
		return resp, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = resp.Body.Close()
		return nil, err
	}
	t.lastResponse = append([]byte(nil), body...)
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))

	return resp, nil
}

func insertTestActiveServerReportingSet(
	t *testing.T,
	db *database.DB,
	hostPK, subscriberPK int64,
	sourceOrderType string,
	items ...*model.EbicsServerReportingItem,
) *model.EbicsServerReportingSet {
	t.Helper()

	set := &model.EbicsServerReportingSet{
		EbicsHostID:       hostPK,
		EbicsSubscriberID: sql.NullInt64{Int64: subscriberPK, Valid: true},
		SourceOrderType:   sourceOrderType,
		VersionTag:        "v1",
		Status:            "ACTIVE",
	}
	require.NoError(t, db.Insert(set).Run())

	for _, item := range items {
		item.ServerReportingSetID = set.ID
		require.NoError(t, db.Insert(item).Run())
	}

	return set
}

func mustHashA006(t *testing.T, raw []byte) []byte {
	t.Helper()

	sum, err := digestForReferenceData(raw, string(libebicscrypto.ESA006))
	require.NoError(t, err)

	return sum
}
