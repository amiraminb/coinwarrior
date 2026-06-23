package importer

import (
	"strings"
	"testing"

	"github.com/amiraminb/coinwarrior/internal/model"
)

// header mirrors the real bank export: 10 columns, with the relevant ones at
// non-leading positions, so the tests exercise name-based column mapping.
const header = "Transaction Date,Account Rtn,Account Number,Transaction Type,Customer Reference Number,Debit Amount,Credit Amount,Principal Balance Amount,Extended Text,Bank Reference Number\n"

// row builds a 10-column data line matching header from the fields the parser cares about.
func row(date, txType, debit, credit, extended string) string {
	cols := []string{date, "021000021", "12345678", txType, "REF001", debit, credit, "1000.00", extended, "BNK999"}
	return strings.Join(cols, ",") + "\n"
}

func TestParseReader(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		wantDate   string
		wantType   string
		wantAmount string
		wantNote   string
		wantErr    bool
	}{
		{
			name:       "debit becomes expense",
			line:       row("06/23/2026", "POS PURCHASE", "12.34", "", "COFFEE SHOP"),
			wantDate:   "2026-06-23",
			wantType:   model.TransactionTypeExpense,
			wantAmount: "12.34",
			wantNote:   "COFFEE SHOP",
		},
		{
			name:       "credit becomes income",
			line:       row("01/05/2026", "DIRECT DEPOSIT", "", "2000.00", "PAYROLL"),
			wantDate:   "2026-01-05",
			wantType:   model.TransactionTypeIncome,
			wantAmount: "2000.00",
			wantNote:   "PAYROLL",
		},
		{
			name:       "note with only extended text",
			line:       row("12/31/2025", "", "5.00", "", "SOMETHING"),
			wantDate:   "2025-12-31",
			wantType:   model.TransactionTypeExpense,
			wantAmount: "5.00",
			wantNote:   "SOMETHING",
		},
		{
			name:     "invalid date flagged",
			line:     row("2026-06-23", "POS", "12.34", "", "COFFEE"),
			wantErr:  true,
			wantType: model.TransactionTypeExpense,
		},
		{
			name:    "both debit and credit flagged",
			line:    row("06/23/2026", "XFER", "10.00", "20.00", "AMBIGUOUS"),
			wantErr: true,
		},
		{
			name:    "neither debit nor credit flagged",
			line:    row("06/23/2026", "FEE", "", "", "NO AMOUNT"),
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := ParseReader(strings.NewReader(header + tc.line))
			if err != nil {
				t.Fatalf("ParseReader returned file error: %v", err)
			}
			if len(rows) != 1 {
				t.Fatalf("got %d rows, want 1", len(rows))
			}
			r := rows[0]

			if tc.wantErr {
				if r.ParseErr == nil {
					t.Fatalf("expected ParseErr, got none (row=%+v)", r)
				}
				return
			}
			if r.ParseErr != nil {
				t.Fatalf("unexpected ParseErr: %v", r.ParseErr)
			}
			if r.Date != tc.wantDate {
				t.Errorf("Date = %q, want %q", r.Date, tc.wantDate)
			}
			if r.Type != tc.wantType {
				t.Errorf("Type = %q, want %q", r.Type, tc.wantType)
			}
			if r.AmountInput != tc.wantAmount {
				t.Errorf("AmountInput = %q, want %q", r.AmountInput, tc.wantAmount)
			}
			if r.Note != tc.wantNote {
				t.Errorf("Note = %q, want %q", r.Note, tc.wantNote)
			}
		})
	}
}

func TestParseReaderMapsByHeaderNameIgnoringOrder(t *testing.T) {
	// Columns deliberately reordered and an extra column added; name-based mapping
	// should still pick out the right fields.
	input := "Extended Text,Credit Amount,Transaction Date,Debit Amount,Transaction Type,Junk\n" +
		"GROCERIES,,06/23/2026,45.67,POS,ignored\n"

	rows, err := ParseReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseReader error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want 1", len(rows))
	}
	r := rows[0]
	if r.ParseErr != nil {
		t.Fatalf("unexpected ParseErr: %v", r.ParseErr)
	}
	if r.Date != "2026-06-23" || r.Type != model.TransactionTypeExpense || r.AmountInput != "45.67" || r.Note != "GROCERIES" {
		t.Errorf("mismapped row: %+v", r)
	}
}

func TestParseReaderStripsUTF8BOM(t *testing.T) {
	// A UTF-8 BOM prepended to the file lands on the first header cell; it must be
	// stripped so the column still matches and the file is not rejected.
	input := "\uFEFF" + header + row("06/23/2026", "POS", "12.34", "", "COFFEE")
	rows, err := ParseReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseReader error with BOM: %v", err)
	}
	if len(rows) != 1 || rows[0].ParseErr != nil {
		t.Fatalf("BOM not handled: rows=%+v", rows)
	}
	if rows[0].Date != "2026-06-23" || rows[0].AmountInput != "12.34" {
		t.Errorf("BOM row mismapped: %+v", rows[0])
	}
}

func TestParseReaderMissingRequiredColumn(t *testing.T) {
	// No Debit Amount / Credit Amount columns at all.
	input := "Transaction Date,Transaction Type,Extended Text\n06/23/2026,POS,COFFEE\n"
	if _, err := ParseReader(strings.NewReader(input)); err == nil {
		t.Fatal("expected error for missing required columns")
	}
}

func TestParseReaderSkipsHeaderAndBlankLines(t *testing.T) {
	input := header +
		row("06/23/2026", "POS", "12.34", "", "A") +
		"\n" +
		row("06/24/2026", "DEP", "", "5.00", "B")

	rows, err := ParseReader(strings.NewReader(input))
	if err != nil {
		t.Fatalf("ParseReader error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2 (header and blank line should be skipped)", len(rows))
	}
	if rows[0].RowNo != 1 || rows[1].RowNo != 2 {
		t.Errorf("row numbers = %d,%d, want 1,2", rows[0].RowNo, rows[1].RowNo)
	}
}

func TestParseReaderEmptyFile(t *testing.T) {
	if _, err := ParseReader(strings.NewReader("")); err == nil {
		t.Fatal("expected error for empty file")
	}
}

func TestParseReaderHeaderOnly(t *testing.T) {
	if _, err := ParseReader(strings.NewReader(header)); err == nil {
		t.Fatal("expected error for file with no data rows")
	}
}
