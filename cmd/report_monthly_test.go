package cmd

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/amiraminb/coinwarrior/internal/model"
)

func day(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.ParseInLocation("2006-01-02", value, time.UTC)
	if err != nil {
		t.Fatalf("day(%q): %v", value, err)
	}
	return parsed
}

func TestMonthsInRange(t *testing.T) {
	tests := []struct {
		name  string
		start string
		end   string
		want  []string
	}{
		{"one whole month", "2026-02-01", "2026-02-28", []string{"2026-02"}},
		{"three whole months", "2026-01-01", "2026-03-31", []string{"2026-01", "2026-02", "2026-03"}},
		{"partial at both ends", "2026-01-15", "2026-03-20", []string{"2026-01 (15-31)", "2026-02", "2026-03 (01-20)"}},
		{"part of a single month", "2026-02-15", "2026-02-20", []string{"2026-02 (15-20)"}},
		{"one day either side of a boundary", "2026-01-31", "2026-02-01", []string{"2026-01 (31-31)", "2026-02 (01-01)"}},
		{"across a year boundary", "2026-12-20", "2027-01-10", []string{"2026-12 (20-31)", "2027-01 (01-10)"}},
		{"a single day", "2026-02-10", "2026-02-10", []string{"2026-02 (10-10)"}},
		{"february in a leap year", "2024-02-01", "2024-02-29", []string{"2024-02"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := monthsInRange(day(t, tc.start), day(t, tc.end))
			if !slices.Equal(got, tc.want) {
				t.Errorf("monthsInRange(%s..%s) = %v, want %v", tc.start, tc.end, got, tc.want)
			}
		})
	}
}

// The per-month sections total only a partial month's covered days, so each
// span must carry bounds clipped to the range rather than the whole month.
func TestMonthSpansClipToTheRange(t *testing.T) {
	spans := monthSpansInRange(day(t, "2026-01-15"), day(t, "2026-03-20"))

	if len(spans) != 3 {
		t.Fatalf("got %d spans, want 3", len(spans))
	}
	if !spans[0].start.Equal(day(t, "2026-01-15")) || !spans[0].end.Equal(day(t, "2026-01-31")) {
		t.Errorf("january span = %s..%s, want 2026-01-15..2026-01-31", spans[0].start.Format("2006-01-02"), spans[0].end.Format("2006-01-02"))
	}
	if !spans[1].start.Equal(day(t, "2026-02-01")) || !spans[1].end.Equal(day(t, "2026-02-28")) {
		t.Errorf("february span = %s..%s, want the whole month", spans[1].start.Format("2006-01-02"), spans[1].end.Format("2006-01-02"))
	}
	if !spans[2].start.Equal(day(t, "2026-03-01")) || !spans[2].end.Equal(day(t, "2026-03-20")) {
		t.Errorf("march span = %s..%s, want 2026-03-01..2026-03-20", spans[2].start.Format("2006-01-02"), spans[2].end.Format("2006-01-02"))
	}
	if spans[0].key != "2026-01" || spans[2].key != "2026-03" {
		t.Errorf("keys = %q / %q, want the bare month keys", spans[0].key, spans[2].key)
	}
}

func TestHasReportableTransaction(t *testing.T) {
	transfer := monthlyTx(model.TransactionTypeTransfer, "2026-01-10", 900000)
	transfer.ToAccount = "Sav"

	start, end := day(t, "2026-01-01"), day(t, "2026-02-28")

	if hasReportableTransaction(nil, start, end) {
		t.Error("an empty ledger reported a transaction")
	}
	if hasReportableTransaction([]model.Transaction{transfer}, start, end) {
		t.Error("a transfer-only ledger must not count as reportable")
	}
	outside := []model.Transaction{monthlyTx(model.TransactionTypeExpense, "2025-12-31", 1000)}
	if hasReportableTransaction(outside, start, end) {
		t.Error("an out-of-range transaction must not count as reportable")
	}
	inside := []model.Transaction{monthlyTx(model.TransactionTypeExpense, "2026-01-10", 1000)}
	if !hasReportableTransaction(inside, start, end) {
		t.Error("an in-range expense should count as reportable")
	}
}

func monthlyTx(txType, date string, amountMinor int64) model.Transaction {
	return model.Transaction{
		Type:        txType,
		AmountMinor: amountMinor,
		Currency:    "CAD",
		Date:        date,
		Category:    "Groceries",
		Account:     "Chk",
	}
}

func TestMonthlyBreakdownTotalsEachMonth(t *testing.T) {
	transactions := []model.Transaction{
		monthlyTx(model.TransactionTypeIncome, "2026-01-05", 500000),
		monthlyTx(model.TransactionTypeExpense, "2026-01-20", 120000),
		monthlyTx(model.TransactionTypeExpense, "2026-01-25", 80000),
		monthlyTx(model.TransactionTypeExpense, "2026-02-10", 300000),
	}

	got := monthlyBreakdown(transactions, day(t, "2026-01-01"), day(t, "2026-02-28"))

	rows, ok := got["CAD"]
	if !ok {
		t.Fatalf("no CAD rows in %v", got)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	if rows[0].incomeMinor != 500000 || rows[0].expenseMinor != 200000 {
		t.Errorf("january = income %d expense %d, want 500000 / 200000", rows[0].incomeMinor, rows[0].expenseMinor)
	}
	if rows[1].incomeMinor != 0 || rows[1].expenseMinor != 300000 {
		t.Errorf("february = income %d expense %d, want 0 / 300000", rows[1].incomeMinor, rows[1].expenseMinor)
	}
}

// A transfer is neither earned nor spent, so counting it would inflate both
// columns for money only moved between accounts.
func TestMonthlyBreakdownExcludesTransfers(t *testing.T) {
	transfer := monthlyTx(model.TransactionTypeTransfer, "2026-01-10", 900000)
	transfer.ToAccount = "Sav"
	transactions := []model.Transaction{
		monthlyTx(model.TransactionTypeExpense, "2026-01-10", 1000),
		transfer,
	}

	rows := monthlyBreakdown(transactions, day(t, "2026-01-01"), day(t, "2026-02-28"))["CAD"]

	if rows[0].expenseMinor != 1000 {
		t.Errorf("january expense = %d, want 1000 (the transfer must not count)", rows[0].expenseMinor)
	}
	if rows[0].incomeMinor != 0 {
		t.Errorf("january income = %d, want 0", rows[0].incomeMinor)
	}
}

// A currency seen only on transfers must not appear at all, rather than as a
// table of zeroes.
func TestMonthlyBreakdownOmitsATransferOnlyCurrency(t *testing.T) {
	transfer := monthlyTx(model.TransactionTypeTransfer, "2026-01-10", 900000)
	transfer.Currency = "USD"
	transfer.ToAccount = "Sav"
	transactions := []model.Transaction{
		monthlyTx(model.TransactionTypeExpense, "2026-01-10", 1000),
		transfer,
	}

	got := monthlyBreakdown(transactions, day(t, "2026-01-01"), day(t, "2026-02-28"))

	if _, ok := got["USD"]; ok {
		t.Errorf("USD appears with only a transfer behind it: %v", got["USD"])
	}
	if len(got) != 1 {
		t.Errorf("got %d currencies, want only CAD", len(got))
	}
}

func TestMonthlyBreakdownIgnoresTransactionsOutsideTheRange(t *testing.T) {
	transactions := []model.Transaction{
		monthlyTx(model.TransactionTypeExpense, "2025-12-31", 999999),
		monthlyTx(model.TransactionTypeExpense, "2026-01-15", 1000),
		monthlyTx(model.TransactionTypeExpense, "2026-03-01", 888888),
	}

	rows := monthlyBreakdown(transactions, day(t, "2026-01-01"), day(t, "2026-02-28"))["CAD"]

	total := int64(0)
	for _, row := range rows {
		total += row.expenseMinor
	}
	if total != 1000 {
		t.Errorf("total expense = %d, want 1000 (out-of-range rows must be skipped)", total)
	}
}

func TestMonthlyBreakdownSeparatesCurrencies(t *testing.T) {
	usd := monthlyTx(model.TransactionTypeExpense, "2026-01-10", 5000)
	usd.Currency = "USD"
	transactions := []model.Transaction{
		monthlyTx(model.TransactionTypeExpense, "2026-01-10", 1000),
		usd,
	}

	got := monthlyBreakdown(transactions, day(t, "2026-01-01"), day(t, "2026-02-28"))

	if len(got) != 2 {
		t.Fatalf("got %d currencies, want 2", len(got))
	}
	if got["CAD"][0].expenseMinor != 1000 {
		t.Errorf("CAD january = %d, want 1000", got["CAD"][0].expenseMinor)
	}
	if got["USD"][0].expenseMinor != 5000 {
		t.Errorf("USD january = %d, want 5000", got["USD"][0].expenseMinor)
	}
}

func TestMonthlyBreakdownLabelsRowsWithTheirMonth(t *testing.T) {
	transactions := []model.Transaction{monthlyTx(model.TransactionTypeExpense, "2026-01-20", 1000)}

	rows := monthlyBreakdown(transactions, day(t, "2026-01-15"), day(t, "2026-02-10"))["CAD"]

	if rows[0].label != "2026-01 (15-31)" || rows[1].label != "2026-02 (01-10)" {
		t.Errorf("labels = %q / %q, want the partial-month spans", rows[0].label, rows[1].label)
	}
}

func TestBarScalesToThePeak(t *testing.T) {
	full := bar(1000, 1000)
	if got := len([]rune(full)); got != monthlyBarWidth {
		t.Errorf("the peak value drew %d blocks, want %d", got, monthlyBarWidth)
	}

	half := bar(500, 1000)
	if got := len([]rune(half)); got != monthlyBarWidth/2 {
		t.Errorf("half the peak drew %d blocks, want %d", got, monthlyBarWidth/2)
	}
}

// A tiny non-zero amount must still be visible, or a month with real activity
// reads as empty.
func TestBarNeverVanishesForANonZeroAmount(t *testing.T) {
	if got := bar(1, 100000000); got == "" {
		t.Error("a non-zero amount rendered no bar at all")
	}
}

func TestBarIsEmptyForZero(t *testing.T) {
	if got := bar(0, 1000); got != "" {
		t.Errorf("bar(0, 1000) = %q, want an empty string", got)
	}
	if got := bar(1000, 0); got != "" {
		t.Errorf("bar(1000, 0) = %q, want an empty string when nothing has been spent", got)
	}
}

func TestBarCellOmitsTheSeparatorWhenThereIsNoBar(t *testing.T) {
	cell := barCell(0, 1000)

	if strings.HasPrefix(cell, " ") {
		t.Errorf("barCell(0, …) = %q, want no leading space", cell)
	}
	if cell != "0.00" {
		t.Errorf("barCell(0, …) = %q, want %q", cell, "0.00")
	}
}
