package wg

import (
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
		status: http.StatusOK,
		headers: http.Header{},
		body: map[string]any{
			"operation": map[string]any{
				"id":                  77,
				"operationType":       "PAYLOAD",
				"orderType":           "BTU",
				"signatureState":      "NOT_APPLICABLE",
				"direction":           "OUTBOUND",
				"transportMode":       "ASYNC",
				"status":              "COMPLETED",
				"severity":            "INFO",
				"transactionID":       "TX-77",
				"correlationID":       "corr-77",
				"gatewayOutcome":      "SUCCESS",
				"retryDecision":       "NO_RETRY",
				"manualActionRequired": false,
			},
			"hostID":     "HOST-77",
			"partnerID":  "PARTNER-77",
			"userID":     "USER-77",
			"startedAt":  "2026-03-31T10:00:00Z",
			"finishedAt": "2026-03-31T10:01:00Z",
			"links": map[string]any{
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
			"ebicsSubscriberID": float64(44),
			"includeHEV":        false,
		},
	}

	result := &expectedResponse{
		status:  http.StatusOK,
		headers: http.Header{},
		body: map[string]any{
			"protocolCheckOperation": map[string]any{
				"id":                  5,
				"operationType":       "ADMIN",
				"orderType":           "HEV",
				"signatureState":      "NOT_APPLICABLE",
				"direction":           "OUTBOUND",
				"transportMode":       "SYNC",
				"status":              "COMPLETED",
				"severity":            "INFO",
				"gatewayOutcome":      "SUCCESS",
				"retryDecision":       "NO_RETRY",
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

	require.NoError(t, executeCommand(t, w, command, "--subscriber", "44", "--no-hev"))
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
