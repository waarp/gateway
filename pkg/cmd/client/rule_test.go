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
)

func TestRuleGet(t *testing.T) {
	const (
		rule = "push"
		way  = directionSend

		ruleComment   = "this is a comment"
		ruleIsSend    = true
		rulePath      = "push/path"
		ruleLocalDir  = "push/local"
		ruleRemoteDir = "push/remote"
		ruleTmpDir    = "push/tmp"

		preTask1 = "PRE_TASK_1"
		preTask2 = "PRE_TASK_2"
		postTask = "POST_TASK"
		errTask  = "ERR_TASK"

		postTaskArg1 = "arg1"
		postTaskVal1 = "val1"
		postTaskArg2 = "arg2"
		postTaskVal2 = "val2"

		server1      = "server1"
		server2      = "server2"
		partner1     = "partner1"
		partner2     = "partner2"
		locAccount1A = "loc_account1A"
		locAccount1B = "loc_account1B"
		locAccount2A = "loc_account2A"
		locAccount2B = "loc_account2B"
		remAccount1A = "rem_account1A"
		remAccount1B = "rem_account1B"
		remAccount2A = "rem_account2A"
		remAccount2B = "rem_account2B"

		path = "/api/rules/" + rule + "/" + way
	)

	t.Run(`Testing the rule "get" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &RuleGet{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
		}

		result := &expectedResponse{
			status: http.StatusOK,
			body: map[string]any{
				"name":           rule,
				"comment":        ruleComment,
				"isSend":         ruleIsSend,
				"path":           rulePath,
				"localDir":       ruleLocalDir,
				"remoteDir":      ruleRemoteDir,
				"tmpLocalRcvDir": ruleTmpDir,
				"preTasks": []map[string]any{
					{"type": preTask1},
					{"type": preTask2},
				},
				"postTasks": []map[string]any{{
					"type": postTask,
					"args": map[string]string{
						postTaskArg1: postTaskVal1,
						postTaskArg2: postTaskVal2,
					},
				}},
				"errorTasks": []map[string]any{
					{"type": errTask},
				},
				"authorized": map[string]any{
					"servers":  []string{server1, server2},
					"partners": []string{partner1, partner2},
					"localAccounts": map[string][]string{
						server1: {locAccount1A, locAccount1B},
						server2: {locAccount2A, locAccount2B},
					},
					"remoteAccounts": map[string][]string{
						partner1: {remAccount1A, remAccount1B},
						partner2: {remAccount2A, remAccount2B},
					},
				},
			},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, rule, way),
					"Then it should not return an error")

				outputData := maps.Clone(result.body)
				outputData["direction"] = way

				assert.Equal(t,
					expectedOutput(t, outputData,
						`-Rule "{{.name}}" ({{.direction}})`,
						`  -Comment: {{.comment}}`,
						`  -Path: {{.path}}`,
						`  -Local directory: {{.localDir}}`,
						`  -Remote directory: {{.remoteDir}}`,
						`  -Temp receive directory: {{.tmpLocalRcvDir}}`,
						`  -Pre tasks:`,
						`    -Task #1 "{{ (index .preTasks 0).type }}"`,
						`    -Task #2 "{{ (index .preTasks 1).type }}"`,
						`  -Post tasks:`,
						`	 {{- with $postTask := (index .postTasks 0) }}`,
						`    -Task #1 "{{$postTask.type}}" with args:`,
						`      {{- range $arg, $value := $postTask.args }}`,
						`      -{{$arg}}: {{$value}}`,
						`      {{- end }}`,
						`	 {{- end }}`,
						`  -Error tasks:`,
						`    -Task #1 "{{ (index .errorTasks 0).type }}"`,
						`  -Rule access:`,
						`    -Local servers: {{ join .authorized.servers }}`,
						`    -Remote partners: {{ join .authorized.partners }}`,
						`    -Local accounts: {{ flatten .authorized.localAccounts }}`,
						`    -Remote accounts: {{ flatten .authorized.remoteAccounts }}`,
					),
					w.String(),
					"Then is should display the rule's info",
				)
			})
		})
	})
}

func TestRuleAdd(t *testing.T) {
	const (
		ruleName = "pull"
		isSend   = true

		ruleComment   = "this is a comment"
		rulePath      = "rule/path"
		ruleLocalDir  = "rule/local"
		ruleRemoteDir = "rule/remote"
		ruleTmpDir    = "rule/tmp"

		preTask1    = "PRE_TASK_1"
		preTask2    = "PRE_TASK_2"
		postTask    = "POST_TASK"
		errTask     = "ERR_TASK"
		postTaskArg = "arg"
		postTaskVal = "val"

		path     = "/api/rules"
		location = path + "/" + ruleName
	)

	way := direction(isSend)

	t.Run(`Testing the rule "add" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &RuleAdd{}

		expected := &expectedRequest{
			method: http.MethodPost,
			path:   path,
			body: map[string]any{
				"name":           ruleName,
				"isSend":         isSend,
				"comment":        ruleComment,
				"path":           rulePath,
				"localDir":       ruleLocalDir,
				"remoteDir":      ruleRemoteDir,
				"tmpLocalRcvDir": ruleTmpDir,
				"preTasks": []any{
					map[string]any{"type": preTask1},
					map[string]any{"type": preTask2},
				},
				"postTasks": []any{
					map[string]any{
						"type": postTask,
						"args": map[string]any{postTaskArg: postTaskVal},
					},
				},
				"errorTasks": []any{
					map[string]any{"type": errTask},
				},
			},
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--name", ruleName,
					"--comment", ruleComment,
					"--direction", way,
					"--path", rulePath,
					"--local-dir", ruleLocalDir,
					"--remote-dir", ruleRemoteDir,
					"--tmp-dir", ruleTmpDir,
					"--pre", fmt.Sprintf(`{"type":"%s"}`, preTask1),
					"--pre", fmt.Sprintf(`{"type":"%s"}`, preTask2),
					"--post", fmt.Sprintf(`{"type":"%s","args":{"%s": "%s"}}`, postTask, postTaskArg, postTaskVal),
					"--err", fmt.Sprintf(`{"type":"%s"}`, errTask)),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The rule %q was successfully added.\n", ruleName),
					w.String(),
					"Then is should display a message saying the rule was added",
				)
			})
		})
	})
}

func TestRuleUpdate(t *testing.T) {
	const (
		oldName  = "old_pull"
		ruleName = "pull"
		way      = directionRecv

		ruleComment   = "this is a comment"
		rulePath      = "rule/path"
		ruleLocalDir  = "rule/local"
		ruleRemoteDir = "rule/remote"
		ruleTmpDir    = "rule/tmp"

		preTask1    = "PRE_TASK_1"
		preTask2    = "PRE_TASK_2"
		preTask2Arg = "arg"
		preTask2Val = "val"

		path     = "/api/rules/" + oldName + "/" + way
		location = "/api/rules/" + ruleName + "/" + way
	)

	t.Run(`Testing the rule "update" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &RuleUpdate{}

		expected := &expectedRequest{
			method: http.MethodPatch,
			path:   path,
			body: map[string]any{
				"name":           ruleName,
				"comment":        ruleComment,
				"path":           rulePath,
				"localDir":       ruleLocalDir,
				"remoteDir":      ruleRemoteDir,
				"tmpLocalRcvDir": ruleTmpDir,
				"preTasks": []any{
					map[string]any{"type": preTask1},
					map[string]any{
						"type": preTask2,
						"args": map[string]any{preTask2Arg: preTask2Val},
					},
				},
				"errorTasks": []any{},
			},
		}

		result := &expectedResponse{
			status:  http.StatusCreated,
			headers: map[string][]string{"Location": {location}},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--name", ruleName,
					"--comment", ruleComment,
					"--path", rulePath,
					"--local-dir", ruleLocalDir,
					"--remote-dir", ruleRemoteDir,
					"--tmp-dir", ruleTmpDir,
					"--pre", fmt.Sprintf(`{"type":"%s"}`, preTask1),
					"--pre", fmt.Sprintf(`{"type":"%s","args":{"%s":"%s"}}`, preTask2, preTask2Arg, preTask2Val),
					"--err", "",
					oldName, way,
				),
					"Then it should not return an error",
				)

				assert.Equal(t,
					fmt.Sprintf("The rule %q was successfully updated.\n", ruleName),
					w.String(),
					"Then is should display a message saying the rule was updated",
				)
			})
		})
	})
}

func TestRuleDelete(t *testing.T) {
	const (
		ruleName = "pull"
		way      = directionRecv

		path = "/api/rules/" + ruleName + "/" + way
	)

	t.Run(`Testing the rule "delete" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &RuleDelete{}

		expected := &expectedRequest{
			method: http.MethodDelete,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusNoContent}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, ruleName, way),
					"Then it should not return an error")

				assert.Equal(t,
					fmt.Sprintf("The rule %q was successfully deleted.\n", ruleName),
					w.String(),
					"Then is should display a message saying the rule was deleted",
				)
			})
		})
	})
}

func TestRulesList(t *testing.T) {
	const (
		path = "/api/rules"

		sort   = "name+"
		limit  = "10"
		offset = "5"

		rule1Name   = "pull"
		rule1IsSend = true
		rule1Path   = "pull/path"

		rule2Name   = "push"
		rule2IsSend = false
		rule2Path   = "push/path"
	)

	t.Run(`Testing the rule "list" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &RuleList{}

		expected := &expectedRequest{
			method: http.MethodGet,
			path:   path,
			values: url.Values{"sort": {sort}, "limit": {limit}, "offset": {offset}},
		}

		rules := []map[string]any{{
			"name":   rule1Name,
			"isSend": rule1IsSend,
			"path":   rule1Path,
		}, {
			"name":   rule2Name,
			"isSend": rule2IsSend,
			"path":   rule2Path,
		}}

		result := &expectedResponse{
			status: http.StatusOK,
			body:   map[string]any{"rules": rules},
		}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command,
					"--limit", limit, "--offset", offset, "--sort", sort,
				),
					"Then it should not return an error",
				)

				outputData := slices.Clone(rules)
				outputData[0]["direction"] = direction(rule1IsSend)
				outputData[1]["direction"] = direction(rule2IsSend)

				assert.Equal(t,
					expectedOutput(t, outputData,
						`=== Rules ===`,
						`{{- with (index . 0) }}`,
						`-Rule "{{.name}}" ({{.direction}})`,
						`  -Path: {{.path}}`,
						`  -Pre tasks: <none>`,
						`  -Post tasks: <none>`,
						`  -Error tasks: <none>`,
						`  -Rule access: <unrestricted>`,
						`{{- end }}`,
						`{{- with (index . 1) }}`,
						`-Rule "{{.name}}" ({{.direction}})`,
						`  -Path: {{.path}}`,
						`  -Pre tasks: <none>`,
						`  -Post tasks: <none>`,
						`  -Error tasks: <none>`,
						`  -Rule access: <unrestricted>`,
						`{{- end }}`,
					),
					w.String(),
					"Then is should display the list of rules",
				)
			})
		})
	})
}

func TestRuleAllowAll(t *testing.T) {
	const (
		rule = "push"
		way  = directionSend

		path = "/api/rules/" + rule + "/" + way + "/allow_all"
	)

	t.Run(`Testing the rule "list" command`, func(t *testing.T) {
		w := newTestOutput()
		command := &RuleAllowAll{}

		expected := &expectedRequest{
			method: http.MethodPut,
			path:   path,
		}

		result := &expectedResponse{status: http.StatusOK}

		t.Run("Given a dummy gateway REST interface", func(t *testing.T) {
			testServer(t, expected, result)

			t.Run("When executing the command", func(t *testing.T) {
				require.NoError(t, executeCommand(t, w, command, rule, way),
					"Then it should not return an error")
			})

			assert.Equal(t,
				fmt.Sprintf("The use of the %s rule %q is now unrestricted.\n", way, rule),
				w.String(),
				"Then is should display a message saying the rule's use is now unrestricted",
			)
		})
	})
}
