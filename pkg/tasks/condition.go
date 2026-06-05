package tasks

import (
	"path"
	"strings"

	"github.com/expr-lang/expr"
)

// EvalCondition evaluates a condition expression after variable substitution.
// Returns true if the condition is met, or if the condition is empty.
//
// The expression is evaluated by expr-lang/expr with full operator support:
//
//	"value" == "value"             string comparison
//	42 > 10                        numeric comparison
//	expr and expr                  logical AND
//	expr or expr                   logical OR
//	not expr                       logical NOT
//	(expr)                         grouping / precedence
//	"value" matches "pattern"      regex matching
//	glob("filename", "*.csv")      glob matching (*, ?)
//	contains("hello world", "lo")  substring check
func EvalCondition(condition string) bool {
	condition = strings.TrimSpace(condition)

	if condition == "" {
		return true
	}

	env := map[string]any{
		"glob":       globMatch,
		"hasSubstr":  stringContains,
		"contains":   stringContains,
	}

	program, err := expr.Compile(condition, expr.Env(env), expr.AsBool())
	if err != nil {
		// If the expression can't be parsed, treat non-empty as truthy
		// (existence check).
		return condition != ""
	}

	result, err := expr.Run(program, env)
	if err != nil {
		return false
	}

	val, ok := result.(bool)

	return ok && val
}

// globMatch performs glob-style pattern matching (*, ?).
func globMatch(value, pattern string) bool {
	matched, err := path.Match(pattern, value)

	return err == nil && matched
}

// stringContains checks if value contains the substring.
func stringContains(value, sub string) bool {
	return strings.Contains(value, sub)
}
