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

// Column header names the parser looks up (case-insensitively). Columns are matched
// by header name rather than fixed position so the export's column order and any
// extra columns do not matter.
const (
	colDate     = "transaction date"
	colDebit    = "debit amount"
	colCredit   = "credit amount"
	colExtended = "extended text"
)

// requiredColumns must be present in the header for the file to be usable.
var requiredColumns = []string{colDate, colDebit, colCredit}

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

	header, err := reader.Read()
	if err == io.EOF {
		return nil, fmt.Errorf("csv file is empty")
	} else if err != nil {
		return nil, fmt.Errorf("reading header: %w", err)
	}

	columns, err := indexColumns(header)
	if err != nil {
		return nil, err
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
		rows = append(rows, parseRecord(record, len(rows)+1, columns))
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("csv file has no data rows")
	}

	return rows, nil
}

// indexColumns maps each required and optional column name to its position in the
// header, matching case-insensitively. It errors only when a required column is
// missing, since the whole file is then unusable.
func indexColumns(header []string) (map[string]int, error) {
	columns := make(map[string]int, len(header))
	for i, name := range header {
		// Strip a UTF-8 BOM the export may prepend to the first header cell,
		// otherwise its name won't match and a required column looks missing.
		name = strings.TrimPrefix(name, "\uFEFF")
		columns[strings.ToLower(strings.TrimSpace(name))] = i
	}

	var missing []string
	for _, name := range requiredColumns {
		if _, ok := columns[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("csv is missing required column(s): %s", strings.Join(missing, ", "))
	}

	return columns, nil
}

func parseRecord(record []string, rowNo int, columns map[string]int) ParsedRow {
	row := ParsedRow{RowNo: rowNo}

	rawDate := field(record, columns, colDate)
	debit := field(record, columns, colDebit)
	credit := field(record, columns, colCredit)

	row.Note = field(record, columns, colExtended)

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

// field returns the trimmed value of the named column for a record, or "" when the
// column is absent from the header or the record is too short to include it.
func field(record []string, columns map[string]int, name string) string {
	idx, ok := columns[name]
	if !ok || idx >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[idx])
}

func isBlankRecord(record []string) bool {
	for _, field := range record {
		if strings.TrimSpace(field) != "" {
			return false
		}
	}
	return true
}
