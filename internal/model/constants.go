package model

const (
	TransactionTypeExpense  = "expense"
	TransactionTypeIncome   = "income"
	TransactionTypeTransfer = "transfer"
)

const (
	TransferCategory           = "Transfer"
	TransactionSourceManual    = "manual"
	TransactionSourceRecurring = "recurring"
)

const (
	BudgetSummaryStatusOpen    = "open"
	BudgetSummaryStatusPending = "pending"
)

const (
	BudgetRolloverStatusCarried = "carried"
	BudgetRolloverStatusSkipped = "skipped"
)
