package importer

import (
	"strings"
	"testing"

	"github.com/amiraminb/coinwarrior/internal/model"
)

const header = "Transaction Date,Transaction Type,Debit Amount,Credit Amount,Extended Text\n"

func TestParseReader(t *testing.T) {
	tests := []struct {
		name       string
		row        string
		wantDate   string
		wantType   string
		wantAmount string
		wantNote   string
		wantErr    bool
	}{
		{
			name:       "debit becomes expense",
			row:        "06/23/2026,POS PURCHASE,12.34,,COFFEE SHOP",
			wantDate:   "2026-06-23",
			wantType:   model.TransactionTypeExpense,
			wantAmount: "12.34",
			wantNote:   "POS PURCHASE: COFFEE SHOP",
		},
		{
			name:       "credit becomes income",
			row:        "01/05/2026,DIRECT DEPOSIT,,2000.00,PAYROLL",
			wantDate:   "2026-01-05",
			wantType:   model.TransactionTypeIncome,
			wantAmount: "2000.00",
			wantNote:   "DIRECT DEPOSIT: PAYROLL",
		},
		{
			name:       "note with only extended text",
			row:        "12/31/2025,,5.00,,SOMETHING",
			wantDate:   "2025-12-31",
			wantType:   model.TransactionTypeExpense,
			wantAmount: "5.00",
			wantNote:   "SOMETHING",
		},
		{
			name:     "invalid date flagged",
			row:      "2026-06-23,POS,12.34,,COFFEE",
			wantErr:  true,
			wantType: model.TransactionTypeExpense,
		},
		{
			name:    "both debit and credit flagged",
			row:     "06/23/2026,XFER,10.00,20.00,AMBIGUOUS",
			wantErr: true,
		},
		{
			name:    "neither debit nor credit flagged",
			row:     "06/23/2026,FEE,,,NO AMOUNT",
			wantErr: true,
		},
		{
			name:    "too few columns flagged",
			row:     "06/23/2026,POS,12.34",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := ParseReader(strings.NewReader(header + tc.row + "\n"))
			if err != nil {
				t.Fatalf("ParseReader returned file error: %v", err)
			}
			if len(rows) != 1 {
				t.Fatalf("got %d rows, want 1", len(rows))
			}
			row := rows[0]

			if tc.wantErr {
				if row.ParseErr == nil {
					t.Fatalf("expected ParseErr, got none (row=%+v)", row)
				}
				return
			}
			if row.ParseErr != nil {
				t.Fatalf("unexpected ParseErr: %v", row.ParseErr)
			}
			if row.Date != tc.wantDate {
				t.Errorf("Date = %q, want %q", row.Date, tc.wantDate)
			}
			if row.Type != tc.wantType {
				t.Errorf("Type = %q, want %q", row.Type, tc.wantType)
			}
			if row.AmountInput != tc.wantAmount {
				t.Errorf("AmountInput = %q, want %q", row.AmountInput, tc.wantAmount)
			}
			if row.Note != tc.wantNote {
				t.Errorf("Note = %q, want %q", row.Note, tc.wantNote)
			}
		})
	}
}

func TestParseReaderSkipsHeaderAndBlankLines(t *testing.T) {
	input := header +
		"06/23/2026,POS,12.34,,A\n" +
		"\n" +
		"06/24/2026,POS,,5.00,B\n"

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
