package importer

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/amiraminb/coinwarrior/internal/model"
)

// csvDateLayout is the date format used by the bank export (MM/DD/YYYY).
const csvDateLayout = "01/02/2006"

// expectedColumns is the minimum column count of a supported export row:
// transaction date, transaction type, debit amount, credit amount, extended text.
const expectedColumns = 5

// ParsedRow is one transaction read from the CSV, normalized into the fields the
// import flow needs. AmountInput stays a string so the canonical money.Parse rules
// apply later at save time rather than being duplicated here. A row that fails to
// normalize is still returned with ParseErr set so the caller can let the user fix
// it instead of silently dropping data.
type ParsedRow struct {
	// RowNo is the 1-based position of this row among data rows (header excluded),
	// used for user-facing "row N of M" messaging.
	RowNo       int
	Date        string
	Type        string
	AmountInput string
	Note        string
	ParseErr    error
}

// ParseFile reads and parses a bank-export CSV at path.
func ParseFile(path string) ([]ParsedRow, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return ParseReader(file)
}

// ParseReader parses a bank-export CSV from r. The first record is treated as a
// header and discarded. The returned error covers only failures that make the
// whole file unusable; per-row problems are reported via ParsedRow.ParseErr.
func ParseReader(r io.Reader) ([]ParsedRow, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	if _, err := reader.Read(); err == io.EOF {
		return nil, fmt.Errorf("csv file is empty")
	} else if err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}

	var rows []ParsedRow
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading row %d: %w", len(rows)+1, err)
		}
		if isBlankRecord(record) {
			continue
		}
		rows = append(rows, parseRecord(record, len(rows)+1))
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("csv file has no data rows")
	}

	return rows, nil
}

func parseRecord(record []string, rowNo int) ParsedRow {
	row := ParsedRow{RowNo: rowNo}

	if len(record) < expectedColumns {
		row.ParseErr = fmt.Errorf("expected %d columns, got %d", expectedColumns, len(record))
		return row
	}

	rawDate := strings.TrimSpace(record[0])
	bankType := strings.TrimSpace(record[1])
	debit := strings.TrimSpace(record[2])
	credit := strings.TrimSpace(record[3])
	extended := strings.TrimSpace(record[4])

	row.Note = composeNote(bankType, extended)

	switch {
	case debit != "" && credit != "":
		row.ParseErr = fmt.Errorf("row has both debit (%s) and credit (%s) amounts", debit, credit)
	case debit != "":
		row.Type = model.TransactionTypeExpense
		row.AmountInput = debit
	case credit != "":
		row.Type = model.TransactionTypeIncome
		row.AmountInput = credit
	default:
		row.ParseErr = fmt.Errorf("row has no debit or credit amount")
	}

	if parsed, err := time.Parse(csvDateLayout, rawDate); err != nil {
		row.Date = rawDate
		if row.ParseErr == nil {
			row.ParseErr = fmt.Errorf("invalid date %q (expected MM/DD/YYYY)", rawDate)
		}
	} else {
		row.Date = parsed.Format("2006-01-02")
	}

	return row
}

// composeNote folds the bank's transaction-type string and the extended text into
// a single note, keeping whichever side is present.
func composeNote(bankType, extended string) string {
	switch {
	case bankType != "" && extended != "":
		return bankType + ": " + extended
	case bankType != "":
		return bankType
	default:
		return extended
	}
}

func isBlankRecord(record []string) bool {
	for _, field := range record {
		if strings.TrimSpace(field) != "" {
			return false
		}
	}
	return true
}
