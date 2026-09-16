package cmd

import (
	"reflect"
	"testing"

	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/charmbracelet/bubbles/table"
)

func TestFutureValueRowsAreSortedAndProjected(t *testing.T) {
	rows := futureValueRows(map[string]int64{
		"USD": 2500,
		"CAD": 10000,
	})

	want := []table.Row{
		{"CAD", "100.00", "127.63", "162.89", "207.89", "265.33"},
		{"USD", "25.00", "31.91", "40.72", "51.97", "66.33"},
	}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("futureValueRows() = %v, want %v", rows, want)
	}
}

func TestTransactionSpendingByCurrencyExcludesIncomeAndTransfers(t *testing.T) {
	income := monthlyTx(model.TransactionTypeIncome, "2026-01-01", 50000)
	transfer := monthlyTx(model.TransactionTypeTransfer, "2026-01-02", 30000)
	spending := transactionSpendingByCurrency([]model.Transaction{
		monthlyTx(model.TransactionTypeExpense, "2026-01-03", 10000),
		income,
		transfer,
	})

	if !reflect.DeepEqual(spending, map[string]int64{"CAD": 10000}) {
		t.Errorf("transactionSpendingByCurrency() = %v, want CAD expense only", spending)
	}
}
