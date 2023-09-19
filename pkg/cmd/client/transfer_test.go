package wg

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.waarp.fr/apps/gateway/gateway/pkg/model/types"
	"code.waarp.fr/apps/gateway/gateway/pkg/utils"
)

func TestTransferAdd(t *testing.T) {
	const (
		rule    = "push"
		isSend  = true
		client  = "cli"
		partner = "pard"
		account = "acc"
		file    = "dir/file"
		output  = "dir/out"
		start   = "2020-01-01T00:00:00Z"

		key = "key"
		val = "val"

		id       = "1234"
		path     = "/api/transfers"
		location = path + "/" + id
	)

	t.Run(`Testing the transfer "add" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &TransferAdd{}

		expected := &expectedRequest{
			method: http.MethodPost,
			path:   path,
			body: map[string]any{
				"rule":         rule,
				"isSend":       isSend,
				"client":       client,
				"partner":      partner,
				"account":      account,
				"file":         file,
				"output":       output,
				"start":        start,
				"transferInfo": map[string]any{key: val},
			},
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: http.Header{"Location": []string{location}},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--client", client,
					"--partner", partner,
					"--login", account,
					"--way", direction(isSend),
					"--rule", rule,
					"--file", file,
					"--out", output,
					"--date", start,
					"--info", key+":"+val,
				),
					"Then it should not return an error",
				)

				assert.Equal(t,
					fmt.Sprintf("The transfer of file %q was successfully added under the ID: %s\n",
						file, id),
					w.String(),
					"Then it should display a message saying the account was added",
				)
			})
		})
	})
}

func TestTransferGet(t *testing.T) {
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
		filesize   = 1024
		start      = "2020-01-01T00:00:00Z"
		stop       = "2021-01-01T00:00:00Z"
		status     = types.StatusCancelled
		step       = types.StepData
		progress   = 512
		taskNb     = 3
		errCode    = types.TeDataTransfer
		errMsg     = "some error"

		key = "key"
		val = "val"
	)

	var (
		path = "/api/transfers/" + utils.FormatInt(id)
		role = transferRole(isServer)
		way  = direction(isSend)
	)

	startTime := parseAsLocalTime(t, start)
	stopTime := parseAsLocalTime(t, stop)

	t.Run(`Testing the partner "get" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &TransferGet{}

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
				"client":         client,
				"requester":      account,
				"requested":      partner,
				"protocol":       proto,
				"srcFilename":    file,
				"destFilename":   output,
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
				"transferInfo":   map[string]any{key: val},
			},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, utils.FormatInt(id)),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("── Transfer %d (%s as %s) [%s]\n", id, way, role, status)+
						fmt.Sprintf("   ├─ Remote ID: %s\n", remoteID)+
						fmt.Sprintf("   ├─ Protocol: %s\n", proto)+
						fmt.Sprintf("   ├─ File to send: %s\n", file)+
						fmt.Sprintf("   ├─ File deposited as: %s\n", output)+
						fmt.Sprintf("   ├─ Rule: %s\n", rule)+
						fmt.Sprintf("   ├─ Requested by: %s\n", account)+
						fmt.Sprintf("   ├─ Requested to: %s\n", partner)+
						fmt.Sprintf("   ├─ With client: %s\n", client)+
						fmt.Sprintf("   ├─ Full local path: %s\n", localPath)+
						fmt.Sprintf("   ├─ Full remote path: %s\n", remotePath)+
						fmt.Sprintf("   ├─ File size: %d\n", filesize)+
						fmt.Sprintf("   ├─ Start date: %s\n", startTime)+
						fmt.Sprintf("   ├─ End date: %s\n", stopTime)+
						fmt.Sprintf("   ├─ Current step: %s\n", step)+
						fmt.Sprintf("   ├─ Bytes transferred: %d\n", progress)+
						fmt.Sprintf("   ├─ Tasks executed: %d\n", taskNb)+
						fmt.Sprintf("   ├─ Error code: %s\n", errCode)+
						fmt.Sprintf("   ├─ Error message: %s\n", errMsg)+
						fmt.Sprintf("   ╰─ Transfer values\n")+
						fmt.Sprintf("      ╰─ %s: %s\n", key, val),
					w.String(),
					"Then it should display the transfer",
				)
			})
		})
	})
}

func TestTransferList(t *testing.T) {
	const (
		path = "/api/transfers"

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
		progress2   = 256
	)

	var (
		way1  = direction(isSend1)
		way2  = direction(isSend2)
		role1 = transferRole(isServer1)
		role2 = transferRole(isServer2)
	)

	startTime1 := parseAsLocalTime(t, start1)
	startTime2 := parseAsLocalTime(t, start2)

	t.Run(`Testing the transfer "list" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &TransferList{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
			values: url.Values{
				"sort": {sort}, "limit": {limit}, "offset": {offset},
				"rule": {rule}, "status": {status}, "start": {date},
			},
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body: map[string]any{
				"transfers": []map[string]any{
					{
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
						"progress":       progress2,
					},
				},
			},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--limit", limit, "--offset", offset,
					"--sort", sort, "--rule", rule,
					"--status", status, "--date", date,
				),
					"Then it should not return an error",
				)

				assert.Equal(t, "Transfers:\n"+
					fmt.Sprintf("╭─ Transfer %d (%s as %s) [%s]\n", id1, way1, role1, status1)+
					fmt.Sprintf("│  ├─ Remote ID: %s\n", remoteID1)+
					fmt.Sprintf("│  ├─ Protocol: %s\n", proto1)+
					fmt.Sprintf("│  ├─ File to send: %s\n", file1)+
					fmt.Sprintf("│  ├─ File deposited as: %s\n", output1)+
					fmt.Sprintf("│  ├─ Rule: %s\n", rule1)+
					fmt.Sprintf("│  ├─ Requested by: %s\n", account1)+
					fmt.Sprintf("│  ├─ Requested to: %s\n", partner1)+
					fmt.Sprintf("│  ├─ With client: %s\n", client1)+
					fmt.Sprintf("│  ├─ Full local path: %s\n", localPath1)+
					fmt.Sprintf("│  ├─ Full remote path: %s\n", remotePath1)+
					fmt.Sprintf("│  ├─ File size: 0\n")+
					fmt.Sprintf("│  ├─ Start date: %s\n", startTime1)+
					fmt.Sprintf("│  ├─ End date: %s\n", NotApplicable)+
					fmt.Sprintf("│  ╰─ Bytes transferred: %d\n", progress1)+
					fmt.Sprintf("╰─ Transfer %d (%s as %s) [%s]\n", id2, way2, role2, status2)+
					fmt.Sprintf("   ├─ Remote ID: %s\n", remoteID2)+
					fmt.Sprintf("   ├─ Protocol: %s\n", proto2)+
					fmt.Sprintf("   ├─ File pushed: %s\n", output2)+
					fmt.Sprintf("   ├─ Rule: %s\n", rule2)+
					fmt.Sprintf("   ├─ Requested by: %s\n", account2)+
					fmt.Sprintf("   ├─ Requested to: %s\n", server2)+
					fmt.Sprintf("   ├─ Full local path: %s\n", localPath2)+
					fmt.Sprintf("   ├─ Full remote path: %s\n", remotePath2)+
					fmt.Sprintf("   ├─ File size: 0\n")+
					fmt.Sprintf("   ├─ Start date: %s\n", startTime2)+
					fmt.Sprintf("   ├─ End date: %s\n", NotApplicable)+
					fmt.Sprintf("   ╰─ Bytes transferred: %d\n", progress2),
					w.String(),
					"Then it should display the transfers",
				)
			})
		})
	})
}

func TestTransferPause(t *testing.T) {
	const (
		id = "1234"

		path = "/api/transfers/" + id + "/pause"
	)

	t.Run(`Testing the transfer "pause" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &TransferPause{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusAccepted}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, id),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The transfer %q was successfully paused. "+
						"It can be resumed using the 'resume' command.\n", id),
					w.String(),
					"Then it should display a message saying the transfer was paused",
				)
			})
		})
	})
}

func TestTransferResume(t *testing.T) {
	const (
		id = "1234"

		path = "/api/transfers/" + id + "/resume"
	)

	t.Run(`Testing the transfer "resume" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &TransferResume{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusAccepted}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, id),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The transfer %q was successfully resumed.\n", id),
					w.String(),
					"Then it should display a message saying the transfer was resumed",
				)
			})
		})
	})
}

func TestTransferCancel(t *testing.T) {
	const (
		id = "1234"

		path = "/api/transfers/" + id + "/cancel"
	)

	t.Run(`Testing the transfer "cancel" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &TransferCancel{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusAccepted}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, id),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The transfer %q was successfully canceled.\n", id),
					w.String(),
					"Then it should display a message saying the transfer was canceled",
				)
			})
		})
	})
}

func TestTransferRetry(t *testing.T) {
	const (
		id    = "1234"
		newID = "4567"

		path     = "/api/transfers/" + id + "/retry"
		location = "/api/transfers/" + newID
	)

	t.Run(`Testing the transfer "retry" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &TransferRetry{}

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
					fmt.Sprintf("The transfer will be retried under the ID: %q\n", newID),
					w.String(),
					"Then it should display a message saying the transfer will be retried",
				)
			})
		})
	})
}

func TestTransfersCancelAll(t *testing.T) {
	const (
		target = "planned"

		path = "/api/transfers"
	)

	t.Run(`Testing the transfer "cancel all" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &TransferCancelAll{}

		expected := &expectedRequest{
			method: http.MethodDelete,
			path:   path,
			values: url.Values{"target": {target}},
		}

		result := &expectedResponse{status: http.StatusAccepted}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, "--target", target),
					"Then it should not return an error")

				assert.Equal(t,
					"The transfers were successfully canceled.\n",
					w.String(),
					"Then it should display a message saying the transfers were canceled",
				)
			})
		})
	})
}
