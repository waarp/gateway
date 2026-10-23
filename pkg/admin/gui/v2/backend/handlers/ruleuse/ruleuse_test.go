package ruleuse

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"code.waarp.fr/apps/gateway/gateway/pkg/model"
)

func TestOf(t *testing.T) {
	t.Parallel()

	params := Params{
		"ENCRYPT":        {"keyName"},
		"DECRYPT":        {"keyName"},
		"DECRYPT&VERIFY": {"decryptKeyName", "verifyKeyName"},
	}
	rules := model.Rules{
		{Identifier: model.Identifier{ID: 1}, Name: "b-rule", IsSend: false},
		{Identifier: model.Identifier{ID: 2}, Name: "b-rule", IsSend: true},
		{Identifier: model.Identifier{ID: 3}, Name: "a-rule", IsSend: true},
	}
	list := model.Tasks{
		{RuleID: 1, Type: "DECRYPT", Args: map[string]string{"keyName": "k1"}},
		{RuleID: 2, Type: "ENCRYPT", Args: map[string]string{"keyName": "k1"}},
		{RuleID: 2, Type: "ENCRYPT", Args: map[string]string{"keyName": "k1"}, Chain: model.ChainError},
		{RuleID: 3, Type: "DECRYPT&VERIFY", Args: map[string]string{"decryptKeyName": "k2", "verifyKeyName": "k3"}},
		{RuleID: 3, Type: "ENCRYPT", Args: map[string]string{"keyName": "k1"}},
		{RuleID: 3, Type: "EMAIL", Args: map[string]string{"keyName": "k4"}},
		{RuleID: 9, Type: "ENCRYPT", Args: map[string]string{"keyName": "k5"}},
	}

	uses := Of(list, rules, params)

	assert.Equal(t, []Use{
		{RuleID: 3, Rule: "a-rule", Send: true},
		{RuleID: 2, Rule: "b-rule", Send: true},
		{RuleID: 1, Rule: "b-rule", Send: false},
	}, uses["k1"], "one line per rule, by name then sending first")
	assert.Equal(t, []Use{{RuleID: 3, Rule: "a-rule", Send: true}}, uses["k2"])
	assert.Equal(t, []Use{{RuleID: 3, Rule: "a-rule", Send: true}}, uses["k3"])
	assert.NotContains(t, uses, "k4", "a task type the object is not read by")
	assert.NotContains(t, uses, "k5", "a task without its rule")
}

func TestRowsAndCount(t *testing.T) {
	t.Parallel()

	echo := func(key string, args ...any) string {
		return strings.TrimSpace(fmt.Sprintln(append([]any{key}, args...)...))
	}

	assert.Equal(t, []Row{
		{Rule: "a-rule", URL: "/webui/tasks?ruleID=3", Direction: "rule_uses.direction.send"},
		{Rule: "b-rule", URL: "/webui/tasks?ruleID=1", Direction: "rule_uses.direction.receive"},
	}, Rows(echo, []Use{{RuleID: 3, Rule: "a-rule", Send: true}, {RuleID: 1, Rule: "b-rule"}}))

	assert.Equal(t, "rule_uses.count.none", Count(echo, 0))
	assert.Equal(t, "rule_uses.count.one", Count(echo, 1))
	assert.Equal(t, "rule_uses.count.many n 3", Count(echo, 3))
}
