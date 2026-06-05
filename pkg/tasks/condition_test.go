package tasks

import (
	"testing"
)

func TestEvalCondition(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		want      bool
	}{
		// Empty condition
		{"empty", "", true},
		{"whitespace", "   ", true},

		// Existence checks (unparseable = truthy)
		{"non-empty is true", "hello", true},

		// String equality
		{"equal strings", `"EBCDIC" == "EBCDIC"`, true},
		{"unequal strings", `"EBCDIC" == "UTF-8"`, false},
		{"not equal strings", `"EBCDIC" != "UTF-8"`, true},
		{"not equal same", `"EBCDIC" != "EBCDIC"`, false},

		// Numeric comparisons
		{"greater than", "1360 > 0", true},
		{"greater than false", "0 > 1360", false},
		{"less than", "10 < 100", true},
		{"greater or equal", "100 >= 100", true},
		{"less or equal", "99 <= 100", true},
		{"numeric equal", "42 == 42", true},
		{"numeric not equal", "42 != 43", true},
		{"large number", "10485760 > 1048576", true},

		// glob() function (glob-style: *, ?)
		{"glob csv", `glob("report.csv", "*.csv")`, true},
		{"glob no match", `glob("report.txt", "*.csv")`, false},
		{"glob prefix", `glob("data-001.txt", "data-*.txt")`, true},
		{"glob single char", `glob("data-A.txt", "data-?.txt")`, true},

		// matches (regex — native expr-lang operator)
		{"regex match", `"report.csv" matches ".*\\.csv$"`, true},
		{"regex no match", `"report.txt" matches ".*\\.csv$"`, false},

		// contains (native expr-lang operator)
		{"contains found", `"hello world" contains "world"`, true},
		{"contains not found", `"hello world" contains "xyz"`, false},
		// hasSubstr() function form
		{"hasSubstr found", `hasSubstr("hello world", "world")`, true},
		{"hasSubstr not found", `hasSubstr("hello world", "xyz")`, false},

		// not
		{"not true", `not ("EBCDIC" == "UTF-8")`, true},
		{"not false", `not ("EBCDIC" == "EBCDIC")`, false},

		// and
		{"and both true", `"EBCDIC" == "EBCDIC" and 1360 > 0`, true},
		{"and one false", `"EBCDIC" == "EBCDIC" and 0 > 1360`, false},
		{"and both false", `"EBCDIC" == "UTF-8" and 0 > 1360`, false},

		// or
		{"or both true", `"EBCDIC" == "EBCDIC" or 1360 > 0`, true},
		{"or one true", `"EBCDIC" == "UTF-8" or 1360 > 0`, true},
		{"or both false", `"EBCDIC" == "UTF-8" or 0 > 1360`, false},

		// Parentheses and precedence
		{"parentheses override", `("a" == "b" or "c" == "c") and 1 > 0`, true},
		{"without parens", `"a" == "b" or "c" == "c" and 1 > 0`, true},
		{"complex nested", `("a" == "a" and "b" == "b") or ("x" == "y")`, true},
		{"complex nested false", `("a" == "b" and "b" == "b") or ("x" == "y")`, false},

		// Realistic PeSIT conditions (after variable substitution)
		{"pesit encoding", `"EBCDIC" == "EBCDIC"`, true},
		{"pesit format fixed", `"fixed" == "fixed"`, true},
		{"pesit filesize check", "1360 > 1048576", false},
		{"pesit combined", `"EBCDIC" != "UTF-8" and "fixed" == "fixed"`, true},
		{"pesit glob filename", `glob("data-003.txt", "data-*.txt")`, true},
		{"pesit complex", `("EBCDIC" == "EBCDIC" or "ASCII" == "EBCDIC") and 1360 > 0`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EvalCondition(tt.condition)
			if got != tt.want {
				t.Errorf("EvalCondition(%q) = %v, want %v", tt.condition, got, tt.want)
			}
		})
	}
}
