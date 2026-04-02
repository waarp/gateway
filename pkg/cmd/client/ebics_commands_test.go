package wg

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEbicsPayloadUploadCommand(t *testing.T) {
	w := newTestOutput()
	command := &EbicsPayloadUpload{}

	expected := &expectedRequest{
		method: http.MethodPost,
		path:   "/api/ebics/payloads/btu/upload",
		body: map[string]any{
			"profile": "pain.001",
			"rule":    "push-ebics",
			"subscriber": map[string]any{
				"hostID":    "HOST-CLI",
				"partnerID": "PARTNER-CLI",
				"userID":    "USER-CLI",
			},
			"file": map[string]any{
				"path":       "payloads/order.xml",
				"outputName": "remote.xml",
			},
			"service": map[string]any{
				"orderType":     "BTU",
				"serviceName":   "MCT",
				"serviceOption": "URGP",
				"scope":         "BIL",
				"msgName":       "pain.001",
				"containerType": "XML",
			},
			"metadata": map[string]any{
				"channel": "treasury",
			},
		},
	}

	result := &expectedResponse{
		status:  http.StatusCreated,
		headers: http.Header{},
	}

	testServer(t, expected, result)

	require.NoError(t, executeCommand(t, w, command,
		"--order-type", "BTU",
		"--profile", "pain.001",
		"--rule", "push-ebics",
		"--host-id", "HOST-CLI",
		"--partner-id", "PARTNER-CLI",
		"--user-id", "USER-CLI",
		"--file", "payloads/order.xml",
		"--output-name", "remote.xml",
		"--service-name", "MCT",
		"--service-option", "URGP",
		"--scope", "BIL",
		"--msg-name", "pain.001",
		"--container-type", "XML",
		"--metadata", "channel:treasury",
	))

	assert.Equal(t,
		"The EBICS payload \"payloads/order.xml\" was successfully submitted.\n",
		w.String(),
	)
}

func TestEbicsPayloadRecoverCommand(t *testing.T) {
	w := newTestOutput()
	command := &EbicsPayloadRecover{}

	expected := &expectedRequest{
		method: http.MethodPut,
		path:   "/api/ebics/payloads/42/recover",
		body: map[string]any{
			"reason": "resume transfer",
			"metadata": map[string]any{
				"ticket": "EBICS-42",
			},
		},
	}

	result := &expectedResponse{
		status:  http.StatusCreated,
		headers: http.Header{},
	}

	testServer(t, expected, result)

	require.NoError(t, executeCommand(t, w, command,
		"42",
		"--reason", "resume transfer",
		"--metadata", "ticket:EBICS-42",
	))

	assert.Equal(t,
		"The EBICS payload \"42\" was successfully scheduled for recovery.\n",
		w.String(),
	)
}

func TestEbicsOperationGetCommandDisplaysDetail(t *testing.T) {
	w := newTestOutput()
	command := &EbicsOperationGet{}

	expected := &expectedRequest{
		method: http.MethodGet,
		path:   "/api/ebics/operations/77",
	}

	result := &expectedResponse{
		status:  http.StatusOK,
		headers: http.Header{},
		body: map[string]any{
			"operation": map[string]any{
				"id":                     77,
				"operationType":          "PAYLOAD",
				"orderType":              "BTU",
				"signatureState":         "NOT_APPLICABLE",
				"direction":              "OUTBOUND",
				"transportMode":          "ASYNC",
				"status":                 "COMPLETED",
				"severity":               "INFO",
				"transactionID":          "TX-77",
				"correlationID":          "corr-77",
				"technicalReturnCode":    "091005",
				"technicalReturnMessage": "temporary transport warning",
				"businessReturnCode":     "090003",
				"businessReturnMessage":  "business validation warning",
				"gatewayOutcome":         "SUCCESS",
				"retryDecision":          "NO_RETRY",
				"manualActionRequired":   false,
			},
			"hostID":     "HOST-77",
			"partnerID":  "PARTNER-77",
			"userID":     "USER-77",
			"startedAt":  "2026-03-31T10:00:00Z",
			"finishedAt": "2026-03-31T10:01:00Z",
			"links": map[string]any{
				"transferID":     88,
				"contractViewID": 9,
				"rtnEventID":     15,
			},
			"transaction": map[string]any{
				"id":             12,
				"transactionID":  "TX-77",
				"orderType":      "BTU",
				"status":         "COMPLETED",
				"direction":      "OUTBOUND",
				"segmentCount":   2,
				"currentSegment": 2,
				"totalSize":      4096,
			},
			"segments": []map[string]any{
				{
					"id":               1,
					"segmentNumber":    1,
					"segmentStatus":    "STORED",
					"payloadSize":      2048,
					"checksum":         "seg-1",
					"storedPayloadRef": "payload-1",
				},
				{
					"id":               2,
					"segmentNumber":    2,
					"segmentStatus":    "COMPLETED",
					"payloadSize":      2048,
					"checksum":         "seg-2",
					"storedPayloadRef": "payload-2",
				},
			},
		},
	}

	testServer(t, expected, result)

	require.NoError(t, executeCommand(t, w, command, "77"))

	assert.Contains(t, w.String(), "EBICS operation #77 [COMPLETED]")
	assert.Contains(t, w.String(), "Operation type: PAYLOAD")
	assert.Contains(t, w.String(), "Host ID: HOST-77")
	assert.Contains(t, w.String(), "Technical return code: 091005")
	assert.Contains(t, w.String(), "Technical return message: temporary transport warning")
	assert.Contains(t, w.String(), "Business return code: 090003")
	assert.Contains(t, w.String(), "Business return message: business validation warning")
	assert.Contains(t, w.String(), "Transfer ID: 88")
	assert.Contains(t, w.String(), "RTN event ID: 15")
	assert.Contains(t, w.String(), "EBICS transaction #12 [COMPLETED]")
	assert.Contains(t, w.String(), "#1 [STORED] size=2048 checksum=seg-1 ref=payload-1")
}

func TestEbicsTransactionGetCommandFailsOnMissingTransactionPayload(t *testing.T) {
	w := newTestOutput()
	command := &EbicsTransactionGet{}

	expected := &expectedRequest{
		method: http.MethodGet,
		path:   "/api/ebics/transactions/91",
	}

	result := &expectedResponse{
		status:  http.StatusOK,
		headers: http.Header{},
		body: map[string]any{
			"segments": []map[string]any{},
		},
	}

	testServer(t, expected, result)

	err := executeCommand(t, w, command, "91")
	require.Error(t, err)
	assert.True(t, errors.Is(err, errMissingEbicsTransactionPayload))
	assert.Equal(t, "", w.String())
}

func TestEbicsTransactionGetCommandDisplaysCorrelationDetail(t *testing.T) {
	w := newTestOutput()
	command := &EbicsTransactionGet{}

	expected := &expectedRequest{
		method: http.MethodGet,
		path:   "/api/ebics/transactions/92",
	}

	result := &expectedResponse{
		status:  http.StatusOK,
		headers: http.Header{},
		body: map[string]any{
			"transaction": map[string]any{
				"id":             92,
				"transactionID":  "TX-92",
				"orderType":      "BTD",
				"status":         "RECOVERING",
				"direction":      "INBOUND",
				"segmentCount":   2,
				"currentSegment": 1,
				"totalSize":      2048,
				"transferID":     502,
			},
			"hostID":        "HOST-92",
			"partnerID":     "PARTNER-92",
			"userID":        "USER-92",
			"requestID":     "REQ-92",
			"correlationID": "CORR-92",
			"segments": []map[string]any{
				{
					"id":            1,
					"segmentNumber": 1,
					"segmentStatus": "STORED",
					"payloadSize":   2048,
				},
			},
		},
	}

	testServer(t, expected, result)

	require.NoError(t, executeCommand(t, w, command, "92"))
	assert.Contains(t, w.String(), "EBICS transaction #92 [RECOVERING]")
	assert.Contains(t, w.String(), "Transfer ID: 502")
	assert.Contains(t, w.String(), "Host ID: HOST-92")
	assert.Contains(t, w.String(), "Partner ID: PARTNER-92")
	assert.Contains(t, w.String(), "User ID: USER-92")
	assert.Contains(t, w.String(), "Request ID: REQ-92")
	assert.Contains(t, w.String(), "Correlation ID: CORR-92")
}

func TestEbicsTransactionSegmentsCommandShowsEmptyMessage(t *testing.T) {
	w := newTestOutput()
	command := &EbicsTransactionSegments{}

	expected := &expectedRequest{
		method: http.MethodGet,
		path:   "/api/ebics/transactions/105/segments",
	}

	result := &expectedResponse{
		status:  http.StatusOK,
		headers: http.Header{},
		body: map[string]any{
			"segments": []map[string]any{},
		},
	}

	testServer(t, expected, result)

	require.NoError(t, executeCommand(t, w, command, "105"))
	assert.Equal(t, "No EBICS transaction segment found.\n", w.String())
}

func TestEbicsPayloadDownloadCommandBuildsTargetRequest(t *testing.T) {
	w := newTestOutput()
	command := &EbicsPayloadDownload{}

	expected := &expectedRequest{
		method: http.MethodPost,
		path:   "/api/ebics/payloads/btd/download",
		body: map[string]any{
			"profile": "camt.053",
			"rule":    "pull-ebics",
			"subscriber": map[string]any{
				"hostID":    "HOST-DL",
				"partnerID": "PARTNER-DL",
				"userID":    "USER-DL",
			},
			"target": map[string]any{
				"directory": "downloads",
			},
			"service": map[string]any{
				"orderType":   "BTD",
				"serviceName": "STM",
			},
		},
	}

	result := &expectedResponse{
		status:  http.StatusCreated,
		headers: http.Header{},
	}

	testServer(t, expected, result)

	require.NoError(t, executeCommand(t, w, command,
		"--order-type", "BTD",
		"--profile", "camt.053",
		"--rule", "pull-ebics",
		"--host-id", "HOST-DL",
		"--partner-id", "PARTNER-DL",
		"--user-id", "USER-DL",
		"--target-dir", "downloads",
		"--service-name", "STM",
	))

	assert.Equal(t,
		fmt.Sprintf("The EBICS payload download %q was successfully submitted.\n", "BTD"),
		w.String(),
	)
}

func TestEbicsContractViewGetCommandDisplaysItems(t *testing.T) {
	w := newTestOutput()
	command := &EbicsContractViewGet{}

	expected := &expectedRequest{
		method: http.MethodGet,
		path:   "/api/ebics/contract-views/21",
	}

	result := &expectedResponse{
		status:  http.StatusOK,
		headers: http.Header{},
		body: map[string]any{
			"contractView": map[string]any{
				"id":              21,
				"hostID":          "HOST-CV",
				"partnerID":       "PARTNER-CV",
				"userID":          "USER-CV",
				"sourceOrderType": "HTD",
				"versionTag":      "v2026",
				"status":          "ACTIVE",
				"fetchedAt":       "2026-04-01T09:00:00Z",
			},
			"items": []map[string]any{
				{
					"id":                 1,
					"itemType":           "ORDER_TYPE",
					"itemKey":            "BTU-001",
					"orderType":          "BTU",
					"serviceName":        "MCT",
					"serviceOption":      "URGP",
					"scope":              "BIL",
					"msgName":            "pain.001",
					"authorisationLevel": "A",
					"isEnabled":          true,
				},
			},
		},
	}

	testServer(t, expected, result)

	require.NoError(t, executeCommand(t, w, command, "21"))
	assert.Contains(t, w.String(), "EBICS contract view #21 [ACTIVE]")
	assert.Contains(t, w.String(), "Host ID: HOST-CV")
	assert.Contains(t, w.String(), "BTU-001 [ORDER_TYPE / BTU] service=MCT/URGP")
}

func TestEbicsContractViewRefreshCommandBuildsRequest(t *testing.T) {
	w := newTestOutput()
	command := &EbicsContractViewRefresh{}

	expected := &expectedRequest{
		method: http.MethodPost,
		path:   "/api/ebics/contract-views/actions/refresh",
		body: map[string]any{
			"clientID":          float64(14),
			"ebicsSubscriberID": float64(44),
			"includeHEV":        false,
		},
	}

	result := &expectedResponse{
		status:  http.StatusOK,
		headers: http.Header{},
		body: map[string]any{
			"protocolCheckOperation": map[string]any{
				"id":                   5,
				"operationType":        "ADMIN",
				"orderType":            "HEV",
				"signatureState":       "NOT_APPLICABLE",
				"direction":            "OUTBOUND",
				"transportMode":        "SYNC",
				"status":               "COMPLETED",
				"severity":             "INFO",
				"gatewayOutcome":       "SUCCESS",
				"retryDecision":        "NO_RETRY",
				"manualActionRequired": false,
			},
			"contractViews": []map[string]any{
				{
					"id":              21,
					"hostID":          "HOST-CV",
					"sourceOrderType": "HTD",
					"status":          "ACTIVE",
					"fetchedAt":       "2026-04-01T09:00:00Z",
				},
			},
		},
	}

	testServer(t, expected, result)

	require.NoError(t, executeCommand(t, w, command, "--client-id", "14", "--subscriber", "44", "--no-hev"))
	assert.Contains(t, w.String(), "EBICS operation #5 [COMPLETED]")
	assert.Contains(t, w.String(), "=== EBICS contract views ===")
}

func TestEbicsKeyLifecycleActionCommandBuildsRequest(t *testing.T) {
	w := newTestOutput()
	command := &EbicsKeyLifecycleAction{}

	expected := &expectedRequest{
		method: http.MethodPut,
		path:   "/api/ebics/key-lifecycles/12/actions",
		body: map[string]any{
			"action":   "MARK_SENT",
			"operator": "ops",
			"reason":   "submitted",
			"evidence": map[string]any{
				"ticket": "KL-12",
			},
		},
	}

	result := &expectedResponse{
		status:  http.StatusCreated,
		headers: http.Header{},
	}

	testServer(t, expected, result)

	require.NoError(t, executeCommand(t, w, command,
		"12",
		"--action", "MARK_SENT",
		"--operator", "ops",
		"--reason", "submitted",
		"--evidence", "ticket:KL-12",
	))

	assert.Equal(t, "The EBICS key lifecycle \"12\" was successfully updated.\n", w.String())
}

func TestEbicsInitializationActionCommandBuildsRequest(t *testing.T) {
	w := newTestOutput()
	command := &EbicsInitializationAction{}

	expected := &expectedRequest{
		method: http.MethodPut,
		path:   "/api/ebics/initializations/18/actions",
		body: map[string]any{
			"clientID": float64(24),
			"action":   "CANCEL",
			"operator": "ops",
			"reason":   "aborted",
			"evidence": map[string]any{
				"ticket": "INIT-18",
			},
		},
	}

	result := &expectedResponse{
		status:  http.StatusCreated,
		headers: http.Header{},
	}

	testServer(t, expected, result)

	require.NoError(t, executeCommand(t, w, command,
		"18",
		"--client-id", "24",
		"--action", "CANCEL",
		"--operator", "ops",
		"--reason", "aborted",
		"--evidence", "ticket:INIT-18",
	))

	assert.Equal(t, "The EBICS initialization workflow \"18\" was successfully updated.\n", w.String())
}

func TestEbicsKeyRotationPrepareCommandBuildsRequest(t *testing.T) {
	w := newTestOutput()
	command := &EbicsKeyRotationPrepare{}

	expected := &expectedRequest{
		method: http.MethodPost,
		path:   "/api/ebics/key-lifecycles/actions/prepare-rotation",
		body: map[string]any{
			"clientID":                       float64(71),
			"ebicsSubscriberID":              float64(77),
			"coordinationID":                 "coord-77",
			"rotationType":                   "ROTATION",
			"nextAuthenticationCredentialID": float64(101),
			"nextEncryptionCredentialID":     float64(102),
			"nextSignatureCredentialID":      float64(103),
			"signatureOrderType":             "PUB",
			"operator":                       "ops",
			"reason":                         "rotate keys",
			"evidence": map[string]any{
				"ticket": "ROT-77",
			},
		},
	}

	result := &expectedResponse{
		status:  http.StatusCreated,
		headers: http.Header{},
		body: map[string]any{
			"coordinationID": "coord-77",
			"lifecycles": []map[string]any{
				{
					"id":                  1,
					"keyUsage":            "AUTHENTICATION",
					"rotationType":        "ROTATION",
					"coordinationID":      "coord-77",
					"status":              "ORDER_PLANNED",
					"currentCredentialID": 10,
				},
			},
			"operations": []map[string]any{
				{"id": 500},
			},
		},
	}

	testServer(t, expected, result)

	require.NoError(t, executeCommand(t, w, command,
		"--client-id", "71",
		"--subscriber", "77",
		"--coordination-id", "coord-77",
		"--rotation-type", "ROTATION",
		"--auth-credential", "101",
		"--enc-credential", "102",
		"--sig-credential", "103",
		"--signature-order-type", "PUB",
		"--operator", "ops",
		"--reason", "rotate keys",
		"--evidence", "ticket:ROT-77",
	))

	assert.Contains(t, w.String(), "EBICS key rotation \"coord-77\"")
	assert.Contains(t, w.String(), "Operations: 1")
	assert.Contains(t, w.String(), "EBICS key lifecycle #1 [ORDER_PLANNED]")
}

func TestEbicsKeyLifecycleGetCommandDisplaysEvidence(t *testing.T) {
	w := newTestOutput()
	command := &EbicsKeyLifecycleGet{}

	expected := &expectedRequest{
		method: http.MethodGet,
		path:   "/api/ebics/key-lifecycles/44",
	}

	result := &expectedResponse{
		status:  http.StatusOK,
		headers: http.Header{},
		body: map[string]any{
			"id":                  44,
			"keyUsage":            "AUTHENTICATION",
			"rotationType":        "ROTATION",
			"coordinationID":      "coord-44",
			"status":              "ORDER_SENT",
			"currentCredentialID": 10,
			"operator":            "ops",
			"reason":              "submitted",
			"evidence": map[string]any{
				"ticket": "KL-44",
			},
		},
	}

	testServer(t, expected, result)

	require.NoError(t, executeCommand(t, w, command, "44"))
	assert.Contains(t, w.String(), "Operator: ops")
	assert.Contains(t, w.String(), "Reason: submitted")
	assert.Contains(t, w.String(), "Evidence:")
	assert.Contains(t, w.String(), "ticket: KL-44")
}

func TestEbicsInitializationGetCommandDisplaysEvidence(t *testing.T) {
	w := newTestOutput()
	command := &EbicsInitializationGet{}

	expected := &expectedRequest{
		method: http.MethodGet,
		path:   "/api/ebics/initializations/45",
	}

	result := &expectedResponse{
		status:  http.StatusOK,
		headers: http.Header{},
		body: map[string]any{
			"id":           45,
			"status":       "WAITING_BANK_ACTIVATION",
			"currentStep":  "WAITING_BANK_ACTIVATION",
			"operator":     "ops",
			"reason":       "waiting bank feedback",
			"bankFeedback": "pending validation",
			"evidence": map[string]any{
				"ticket": "INIT-45",
			},
		},
	}

	testServer(t, expected, result)

	require.NoError(t, executeCommand(t, w, command, "45"))
	assert.Contains(t, w.String(), "Operator: ops")
	assert.Contains(t, w.String(), "Reason: waiting bank feedback")
	assert.Contains(t, w.String(), "Bank feedback: pending validation")
	assert.Contains(t, w.String(), "Evidence:")
	assert.Contains(t, w.String(), "ticket: INIT-45")
}

func TestEbicsRTNProviderAddCommandBuildsRequest(t *testing.T) {
	w := newTestOutput()
	command := &EbicsRTNProviderAdd{}

	expected := &expectedRequest{
		method: http.MethodPost,
		path:   "/api/ebics/rtn/providers",
		body: map[string]any{
			"name":           "provider-a",
			"transport":      "WSS",
			"enabled":        true,
			"subscriberID":   float64(81),
			"clientID":       float64(18),
			"autoPullPolicy": "AUTO",
			"configuration": map[string]any{
				"endpoint": "wss://bank.example/rtn",
			},
		},
	}

	result := &expectedResponse{
		status:  http.StatusCreated,
		headers: http.Header{},
	}

	testServer(t, expected, result)

	require.NoError(t, executeCommand(t, w, command,
		"--name", "provider-a",
		"--transport", "WSS",
		"--enabled",
		"--subscriber-id", "81",
		"--client-id", "18",
		"--auto-pull-policy", "AUTO",
		"--config", "endpoint:wss://bank.example/rtn",
	))

	assert.Equal(t, "The EBICS RTN provider \"provider-a\" was successfully added.\n", w.String())
}

func TestEbicsRTNEventRetryCommandBuildsRequest(t *testing.T) {
	w := newTestOutput()
	command := &EbicsRTNEventRetry{}

	expected := &expectedRequest{
		method: http.MethodPut,
		path:   "/api/ebics/rtn/events/31/retry",
		body: map[string]any{
			"reason": "temporary outage",
		},
	}

	result := &expectedResponse{
		status:  http.StatusCreated,
		headers: http.Header{},
	}

	testServer(t, expected, result)

	require.NoError(t, executeCommand(t, w, command, "31", "--reason", "temporary outage"))
	assert.Equal(t, "The EBICS RTN event \"31\" was successfully scheduled for retry.\n", w.String())
}

func TestEbicsRTNEventQuarantineCommandBuildsRequest(t *testing.T) {
	w := newTestOutput()
	command := &EbicsRTNEventQuarantine{}

	expected := &expectedRequest{
		method: http.MethodPut,
		path:   "/api/ebics/rtn/events/32/quarantine",
		body: map[string]any{
			"reason": "manual review",
		},
	}

	result := &expectedResponse{
		status:  http.StatusCreated,
		headers: http.Header{},
	}

	testServer(t, expected, result)

	require.NoError(t, executeCommand(t, w, command, "32", "--reason", "manual review"))
	assert.Equal(t, "The EBICS RTN event \"32\" was successfully quarantined.\n", w.String())
}

func TestEbicsRTNEventGetCommandDisplaysOperatorMetadata(t *testing.T) {
	w := newTestOutput()
	command := &EbicsRTNEventGet{}

	expected := &expectedRequest{
		method: http.MethodGet,
		path:   "/api/ebics/rtn/events/33",
	}

	result := &expectedResponse{
		status:  http.StatusOK,
		headers: http.Header{},
		body: map[string]any{
			"id":                  33,
			"source":              "BANK_PUSH",
			"idempotenceKey":      "IDEMP-33",
			"status":              "QUARANTINED",
			"attempts":            2,
			"receivedAt":          "2026-04-01T10:00:00Z",
			"lastError":           "manual review",
			"autoPullOperationID": 901,
			"autoPullTransferID":  902,
			"autoPullOrderType":   "BTD",
			"autoPullStatus":      "FAILED",
			"autoPullOutcome":     "TECHNICAL_FATAL_FAILURE",
			"autoPullRetry":       "NO_RETRY",
			"operatorAction":      "QUARANTINE",
			"operatorReason":      "suspect payload",
			"operatorMetadata": map[string]any{
				"ticket": "RTN-33",
			},
		},
	}

	testServer(t, expected, result)

	require.NoError(t, executeCommand(t, w, command, "33"))
	assert.Contains(t, w.String(), "Operator action: QUARANTINE")
	assert.Contains(t, w.String(), "Operator reason: suspect payload")
	assert.Contains(t, w.String(), "Auto-pull operation ID: 901")
	assert.Contains(t, w.String(), "Auto-pull transfer ID: 902")
	assert.Contains(t, w.String(), "Auto-pull status: FAILED")
	assert.Contains(t, w.String(), "Auto-pull outcome: TECHNICAL_FATAL_FAILURE")
	assert.Contains(t, w.String(), "Operator metadata:")
	assert.Contains(t, w.String(), "ticket: RTN-33")
}

func TestEbicsOperationReportingCommandBuildsHVTRequest(t *testing.T) {
	w := newTestOutput()
	command := &EbicsOperationReporting{}

	expected := &expectedRequest{
		method: http.MethodPost,
		path:   "/api/ebics/operations/actions/reporting",
		body: map[string]any{
			"clientID":          float64(31),
			"ebicsSubscriberID": float64(51),
			"orderType":         "HVT",
			"orderID":           "ORDER-51",
			"completeOrderData": true,
			"fetchLimit":        float64(10),
			"fetchOffset":       float64(2),
			"service": map[string]any{
				"serviceName":   "STM",
				"serviceOption": "ALL",
				"scope":         "CH",
				"msgName":       "camt.053",
				"containerType": "ZIP",
			},
			"metadata": map[string]any{
				"channel": "ops",
			},
		},
	}

	result := &expectedResponse{
		status:  http.StatusCreated,
		headers: http.Header{},
		body: map[string]any{
			"id":                   401,
			"operationType":        "REPORTING",
			"orderType":            "HVT",
			"signatureState":       "NOT_APPLICABLE",
			"direction":            "OUTBOUND",
			"transportMode":        "ASYNC",
			"status":               "PLANNED",
			"severity":             "INFO",
			"gatewayOutcome":       "PENDING_BANK",
			"retryDecision":        "AUTO_RETRY_ALLOWED",
			"manualActionRequired": false,
		},
	}

	testServer(t, expected, result)

	require.NoError(t, executeCommand(t, w, command,
		"--client-id", "31",
		"--subscriber", "51",
		"--order-type", "HVT",
		"--order-id", "ORDER-51",
		"--service-name", "STM",
		"--service-option", "ALL",
		"--scope", "CH",
		"--msg-name", "camt.053",
		"--container-type", "ZIP",
		"--complete-order-data",
		"--fetch-limit", "10",
		"--fetch-offset", "2",
		"--metadata", "channel:ops",
	))

	assert.Contains(t, w.String(), "EBICS operation #401 [PLANNED]")
	assert.Contains(t, w.String(), "Order type: HVT")
}

func TestEbicsOperationSignatureCommandBuildsRequest(t *testing.T) {
	orderData := writeFile(t, "order-data.bin", "order-data")
	signatureData := writeFile(t, "signature-data.bin", "signature-data")

	w := newTestOutput()
	command := &EbicsOperationSignature{}

	expected := &expectedRequest{
		method: http.MethodPost,
		path:   "/api/ebics/operations/actions/signature",
		body: map[string]any{
			"clientID":          float64(41),
			"ebicsSubscriberID": float64(61),
			"orderType":         "HVS",
			"orderID":           "ORDER-61",
			"service": map[string]any{
				"serviceName": "MCT",
			},
			"orderData":     base64.StdEncoding.EncodeToString([]byte("order-data")),
			"signatureData": base64.StdEncoding.EncodeToString([]byte("signature-data")),
			"metadata": map[string]any{
				"ticket": "SIG-61",
			},
		},
	}

	result := &expectedResponse{
		status:  http.StatusCreated,
		headers: http.Header{},
		body: map[string]any{
			"id":                   402,
			"operationType":        "SIGNATURE",
			"orderType":            "HVS",
			"signatureState":       "PENDING_SIGNATURE",
			"direction":            "OUTBOUND",
			"transportMode":        "ASYNC",
			"status":               "PLANNED",
			"severity":             "INFO",
			"gatewayOutcome":       "PENDING_BANK",
			"retryDecision":        "NO_RETRY",
			"manualActionRequired": false,
		},
	}

	testServer(t, expected, result)

	require.NoError(t, executeCommand(t, w, command,
		"--client-id", "41",
		"--subscriber", "61",
		"--order-type", "HVS",
		"--order-id", "ORDER-61",
		"--service-name", "MCT",
		"--order-data", orderData,
		"--signature-data", signatureData,
		"--metadata", "ticket:SIG-61",
	))

	assert.Contains(t, w.String(), "EBICS operation #402 [PLANNED]")
	assert.Contains(t, w.String(), "Order type: HVS")
}
