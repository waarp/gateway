package wg

import (
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func TestGetHistory(t *testing.T) {
	const (
		id         = 1234
		remoteID   = "9876"
		rule       = "push"
		isServer   = false
		isSend     = true
		proto      = "proto"
		client     = "cli"
		partner    = "pard"
		account    = "acc"
		file       = "dir/file"
		output     = "dir/out"
		localPath  = "local/" + file
		remotePath = "remote/" + file
		filesize   = 2048
		start      = "2020-01-01T00:00:00Z"
		stop       = "2021-01-01T00:00:00Z"
		status     = types.StatusDone
		step       = types.StepData
		progress   = 512
		taskNb     = 3
		errCode    = types.TeDataTransfer
		errMsg     = "some error"

		key1, key2 = "key1", "key2"
		val1, val2 = "val1", "val2"
	)

	path := "/api/history/" + utils.FormatInt(id)

	t.Run(`Testing the partner "get" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &HistoryGet{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body: map[string]any{
				"id":             id,
				"remoteID":       remoteID,
				"rule":           rule,
				"isServer":       isServer,
				"isSend":         isSend,
				"requester":      account,
				"requested":      partner,
				"protocol":       proto,
				"localFilepath":  localPath,
				"remoteFilepath": remotePath,
				"filesize":       filesize,
				"start":          start,
				"stop":           stop,
				"status":         status,
				"step":           step,
				"progress":       progress,
				"taskNumber":     taskNb,
				"errorCode":      errCode,
				"errorMsg":       errMsg,
				"transferInfo":   map[string]any{key1: val1, key2: val2},
			},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, utils.FormatInt(id)),
					"Then it should not return an error")

				outputData := maps.Clone(result.body)
				outputData["way"] = direction(isSend)
				outputData["role"] = transferRole(isServer)
				outputData["startTime"] = parseAsLocalTime(t, start)
				outputData["stopTime"] = parseAsLocalTime(t, stop)
				outputData["prettySize"] = prettyBytes(filesize)
				outputData["prettyProgress"] = prettyBytes(progress)
				outputData["key1"] = key1
				outputData["key2"] = key2
				outputData["val1"] = val1
				outputData["val2"] = val2

				assert.Equal(t,
					expectedOutput(t, outputData,
						`‣Transfer {{.id}} ({{.way}} as {{.role}}) [{{.status}}]`,
						`  •Remote ID: {{.remoteID}}`,
						`  •Protocol: {{.protocol}}`,
						`  •Rule: {{.rule}}`,
						`  •Requested by: {{.requester}}`,
						`  •Requested to: {{.requested}}`,
						`  •Full local path: {{.localFilepath}}`,
						`  •Full remote path: {{.remoteFilepath}}`,
						`  •File size: {{.prettySize}}`,
						`  •Start date: {{.startTime}}`,
						`  •End date: {{.stopTime}}`,
						`  •Data transferred: {{.prettyProgress}}`,
						`  •Current step: {{.step}}`,
						`  •Current task: #{{.taskNumber}}`,
						`  •Error code: {{.errorCode}}`,
						`  •Error message: {{.errorMsg}}`,
						`  •Transfer values:`,
						`    {{- range $key, $value := .transferInfo}}`,
						`    ⁃{{$key}}: {{$value}}`,
						`    {{- end }}`,
					),
					w.String(),
					"Then it should display the transfer",
				)
			})
		})
	})
}

func TestListHistory(t *testing.T) {
	const (
		path = "/api/history"

		sort   = "id+"
		limit  = "10"
		offset = "5"
		rule   = "rule"
		status = "DONE"
		date   = "2019-01-01T00:00:00Z"

		id1         = 1
		remoteID1   = "456"
		rule1       = "push"
		isServer1   = false
		isSend1     = true
		proto1      = "proto1"
		client1     = "cli1"
		partner1    = "pard1"
		account1    = "acc1"
		file1       = "dir/file/1"
		output1     = "dir/out/1"
		localPath1  = "local/" + file1
		remotePath1 = "remote/" + output1
		start1      = "2020-01-01T00:00:00Z"
		status1     = types.StatusRunning
		filesize1   = 0
		progress1   = 512

		id2         = 2
		remoteID2   = "789"
		rule2       = "pull"
		isServer2   = true
		isSend2     = false
		proto2      = "proto2"
		server2     = "serv2"
		account2    = "acc2"
		file2       = "dir/file/2"
		output2     = "dir/out/2"
		localPath2  = "local/" + file2
		remotePath2 = "remote/" + output2
		start2      = "2021-01-01T00:00:00Z"
		status2     = types.StatusPaused
		filesize2   = model.UnknownSize
		progress2   = 256
	)

	t.Run(`Testing the history "list" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &HistoryList{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
			values: url.Values{
				"sort": {sort}, "limit": {limit}, "offset": {offset},
				"rule": {rule}, "status": {status}, "start": {date},
			},
		}

		history := []map[string]any{{
			"id":             id1,
			"remoteID":       remoteID1,
			"protocol":       proto1,
			"rule":           rule1,
			"isServer":       isServer1,
			"isSend":         isSend1,
			"client":         client1,
			"requester":      account1,
			"requested":      partner1,
			"srcFilename":    file1,
			"destFilename":   output1,
			"localFilepath":  localPath1,
			"remoteFilepath": remotePath1,
			"start":          start1,
			"status":         status1,
			"filesize":       filesize1,
			"progress":       progress1,
		}, {
			"id":             id2,
			"remoteID":       remoteID2,
			"protocol":       proto2,
			"rule":           rule2,
			"isServer":       isServer2,
			"isSend":         isSend2,
			"requester":      account2,
			"requested":      server2,
			"srcFilename":    file2,
			"destFilename":   output2,
			"localFilepath":  localPath2,
			"remoteFilepath": remotePath2,
			"start":          start2,
			"status":         status2,
			"filesize":       filesize2,
			"progress":       progress2,
		}}

		result := &expectedResponse{
			status: http.StatusOK,
			body:   map[string]any{"history": history},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--limit", limit, "--offset", offset,
					"--sort", sort, "--rule", rule,
					"--status", status, "--start", date,
				),
					"Then it should not return an error",
				)

				outputData := slices.Clone(history)
				outputData[0]["way"] = direction(isSend1)
				outputData[0]["role"] = transferRole(isServer1)
				outputData[0]["startTime"] = parseAsLocalTime(t, start1)
				outputData[0]["prettySize"] = prettyBytes(filesize1)
				outputData[0]["prettyProgress"] = prettyBytes(progress1)
				outputData[1]["way"] = direction(isSend2)
				outputData[1]["role"] = transferRole(isServer2)
				outputData[1]["startTime"] = parseAsLocalTime(t, start2)
				outputData[1]["prettyProgress"] = prettyBytes(progress2)

				assert.Equal(t,
					expectedOutput(t, outputData,
						`=== History ===`,
						`{{- with (index . 0) }}`,
						`‣Transfer {{.id}} ({{.way}} as {{.role}}) [{{.status}}]`,
						`  •Remote ID: {{.remoteID}}`,
						`  •Protocol: {{.protocol}}`,
						`  •Rule: {{.rule}}`,
						`  •Requested by: {{.requester}}`,
						`  •Requested to: {{.requested}}`,
						`  •Full local path: {{.localFilepath}}`,
						`  •Full remote path: {{.remoteFilepath}}`,
						`  •File size: {{.prettySize}}`,
						`  •Start date: {{.startTime}}`,
						`  •End date: N/A`,
						`  •Data transferred: {{.prettyProgress}}`,
						`{{- end }}`,
						`{{- with (index . 1) }}`,
						`‣Transfer {{.id}} ({{.way}} as {{.role}}) [{{.status}}]`,
						`  •Remote ID: {{.remoteID}}`,
						`  •Protocol: {{.protocol}}`,
						`  •Rule: {{.rule}}`,
						`  •Requested by: {{.requester}}`,
						`  •Requested to: {{.requested}}`,
						`  •Full local path: {{.localFilepath}}`,
						`  •Full remote path: {{.remoteFilepath}}`,
						`  •File size: <unknown>`,
						`  •Start date: {{.startTime}}`,
						`  •End date: N/A`,
						`  •Data transferred: {{.prettyProgress}}`,
						`{{- end }}`,
					),
					w.String(),
					"Then it should display the history",
				)
			})
		})
	})
}

func TestRetryHistory(t *testing.T) {
	const (
		id    = "1234"
		newID = "4567"

		path     = "/api/history/" + id + "/retry"
		location = "/api/history/" + newID
	)

	t.Run(`Testing the transfer "retry" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &HistoryRetry{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   path,
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: http.Header{"Location": {location}},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, id),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The transfer will be retried under the ID %q\n", newID),
					w.String(),
					"Then it should display a message saying the transfer will be retried",
				)
			})
		})
	})
}
