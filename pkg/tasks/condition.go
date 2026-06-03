package tasks

import (
	"path"
	"strconv"
	"strings"
)

// EvalCondition evaluates a condition string after variable substitution.
// Returns true if the condition is met, or if the condition is empty.
//
// Supported syntax:
//
//	value OPERATOR value    — comparison
//	value                   — existence check (non-empty = true)
//	clause AND clause       — logical AND
//	clause OR clause        — logical OR
//	NOT clause              — logical NOT
//
// Operators: ==, !=, >, <, >=, <=, MATCHES (glob), CONTAINS
func EvalCondition(condition string) bool {
	condition = strings.TrimSpace(condition)
	if condition == "" {
		return true
	}

	return evalOr(condition)
}

// evalOr splits by " OR " and returns true if any clause is true.
func evalOr(expr string) bool {
	clauses := splitKeyword(expr, " OR ")
	for _, clause := range clauses {
		if evalAnd(strings.TrimSpace(clause)) {
			return true
		}
	}

	return false
}

// evalAnd splits by " AND " and returns true only if all clauses are true.
func evalAnd(expr string) bool {
	clauses := splitKeyword(expr, " AND ")
	for _, clause := range clauses {
		if !evalClause(strings.TrimSpace(clause)) {
			return false
		}
	}

	return true
}

// evalClause evaluates a single clause (with optional NOT prefix).
func evalClause(clause string) bool {
	if strings.HasPrefix(clause, "NOT ") {
		return !evalClause(strings.TrimPrefix(clause, "NOT "))
	}

	return evalComparison(clause)
}

// operators in evaluation order (longest first to avoid partial matches).
var operators = []string{"==", "!=", ">=", "<=", ">", "<", " MATCHES ", " CONTAINS "}

// evalComparison evaluates a single comparison or existence check.
func evalComparison(clause string) bool {
	for _, op := range operators {
		idx := strings.Index(clause, op)
		if idx < 0 {
			continue
		}

		left := strings.TrimSpace(clause[:idx])
		right := strings.TrimSpace(clause[idx+len(op):])

		return compare(left, right, strings.TrimSpace(op))
	}

	// No operator found: existence check (non-empty string = true).
	return strings.TrimSpace(clause) != ""
}

func compare(left, right, op string) bool {
	// Attempt numeric comparison if both values are valid numbers.
	leftNum, leftErr := strconv.ParseFloat(left, 64)
	rightNum, rightErr := strconv.ParseFloat(right, 64)
	isNumeric := leftErr == nil && rightErr == nil

	switch op {
	case "==":
		if isNumeric {
			return leftNum == rightNum
		}

		return left == right
	case "!=":
		if isNumeric {
			return leftNum != rightNum
		}

		return left != right
	case ">":
		if isNumeric {
			return leftNum > rightNum
		}

		return left > right
	case "<":
		if isNumeric {
			return leftNum < rightNum
		}

		return left < right
	case ">=":
		if isNumeric {
			return leftNum >= rightNum
		}

		return left >= right
	case "<=":
		if isNumeric {
			return leftNum <= rightNum
		}

		return left <= right
	case "MATCHES":
		matched, err := path.Match(right, left)

		return err == nil && matched
	case "CONTAINS":
		return strings.Contains(left, right)
	default:
		return false
	}
}

// splitKeyword splits a string by a keyword, but only at the top level
// (not inside quoted strings). This is a simple split that doesn't handle
// nested quotes but covers practical use cases.
func splitKeyword(s, keyword string) []string {
	parts := strings.Split(s, keyword)
	if len(parts) == 1 {
		return parts
	}

	result := make([]string, 0, len(parts))

	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}

	return result
}
