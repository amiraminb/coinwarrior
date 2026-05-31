package money

import (
	"math"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int64
	}{
		{"whole number", "5", 500},
		{"zero", "0", 0},
		{"one decimal", "5.5", 550},
		{"two decimals", "5.50", 550},
		{"leading decimal", ".5", 50},
		{"trailing decimal point", "5.", 500},
		{"negative", "-5", -500},
		{"negative with decimals", "-5.25", -525},
		{"negative leading decimal", "-.5", -50},
		{"surrounding whitespace", "  12.34  ", 1234},
		{"whitespace after sign", "- 5", -500},
		{"thousands separator", "1,234.50", 123450},
		{"multiple thousands separators", "1,234,567", 123456700},
		{"max int64 boundary", "92233720368547758.07", math.MaxInt64},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.input)
			if err != nil {
				t.Fatalf("Parse(%q) returned error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("Parse(%q) = %d, want %d", tc.input, got, tc.want)
			}
		})
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"whitespace only", "   "},
		{"bare minus", "-"},
		{"lone decimal point", "."},
		{"double minus", "--5"},
		{"non-numeric", "abc"},
		{"trailing letters", "5abc"},
		{"three decimals", "5.555"},
		{"two decimal points", "5.5.5"},
		{"comma in fractional part", "1.2,3"},
		{"overflow", "100000000000000000"},
		{"just over max int64", "92233720368547758.08"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got, err := Parse(tc.input); err == nil {
				t.Errorf("Parse(%q) = %d, want error", tc.input, got)
			}
		})
	}
}

func TestFormat(t *testing.T) {
	tests := []struct {
		name  string
		input int64
		want  string
	}{
		{"zero", 0, "0.00"},
		{"under a dollar", 50, "0.50"},
		{"single digit fraction", 5, "0.05"},
		{"whole dollars", 500, "5.00"},
		{"with fraction", 1234, "12.34"},
		{"thousands", 123450, "1,234.50"},
		{"millions", 123456789, "1,234,567.89"},
		{"negative", -525, "-5.25"},
		{"negative thousands", -123450, "-1,234.50"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := Format(tc.input); got != tc.want {
				t.Errorf("Format(%d) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

// Format must produce a string that Parse reads back to the same value, so
// amounts copied from displayed output are accepted verbatim.
func TestParseFormatRoundTrip(t *testing.T) {
	values := []int64{0, 5, 50, 500, 1234, 123450, 123456789, -525, -123450, math.MaxInt64}
	for _, v := range values {
		formatted := Format(v)
		got, err := Parse(formatted)
		if err != nil {
			t.Fatalf("Parse(Format(%d)) = Parse(%q) returned error: %v", v, formatted, err)
		}
		if got != v {
			t.Errorf("round trip for %d: Format = %q, Parse = %d", v, formatted, got)
		}
	}
}
