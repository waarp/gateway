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

		// Existence checks
		{"non-empty is true", "hello", true},
		{"empty string is false", "", true}, // empty condition = always true
		{"value exists", "EBCDIC", true},

		// Equality
		{"equal strings", "EBCDIC == EBCDIC", true},
		{"unequal strings", "EBCDIC == UTF-8", false},
		{"not equal strings", "EBCDIC != UTF-8", true},
		{"not equal same", "EBCDIC != EBCDIC", false},

		// Numeric comparisons
		{"greater than", "1360 > 0", true},
		{"greater than false", "0 > 1360", false},
		{"less than", "10 < 100", true},
		{"greater or equal", "100 >= 100", true},
		{"less or equal", "99 <= 100", true},
		{"numeric equal", "42 == 42", true},
		{"numeric not equal", "42 != 43", true},
		{"large number", "10485760 > 1048576", true},

		// MATCHES (glob)
		{"matches csv", "report.csv MATCHES *.csv", true},
		{"matches no match", "report.txt MATCHES *.csv", false},
		{"matches prefix", "data-001.txt MATCHES data-*.txt", true},
		{"matches single char", "data-A.txt MATCHES data-?.txt", true},

		// CONTAINS
		{"contains found", "hello world CONTAINS world", true},
		{"contains not found", "hello world CONTAINS xyz", false},

		// NOT
		{"not true", "NOT EBCDIC == UTF-8", true},
		{"not false", "NOT EBCDIC == EBCDIC", false},

		// AND
		{"and both true", "EBCDIC == EBCDIC AND 1360 > 0", true},
		{"and one false", "EBCDIC == EBCDIC AND 0 > 1360", false},
		{"and both false", "EBCDIC == UTF-8 AND 0 > 1360", false},

		// OR
		{"or both true", "EBCDIC == EBCDIC OR 1360 > 0", true},
		{"or one true", "EBCDIC == UTF-8 OR 1360 > 0", true},
		{"or both false", "EBCDIC == UTF-8 OR 0 > 1360", false},

		// Combined AND/OR (OR has lower precedence)
		{"and-or precedence", "a == b OR EBCDIC == EBCDIC AND 1 > 0", true},

		// Realistic PeSIT conditions
		{"pesit encoding", "EBCDIC == EBCDIC", true},
		{"pesit format fixed", "fixed == fixed", true},
		{"pesit filesize check", "1360 > 1048576", false},
		{"pesit combined", "EBCDIC != UTF-8 AND fixed == fixed", true},
		{"pesit glob filename", "data-003.txt MATCHES data-*.txt", true},
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
