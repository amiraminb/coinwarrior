package domain

const (
	TransactionTypeExpense  = "expense"
	TransactionTypeIncome   = "income"
	TransactionTypeTransfer = "transfer"
)

const (
	TransferCategory        = "Transfer"
	TransactionSourceManual = "manual"
)

const (
	BudgetSummaryStatusOpen    = "open"
	BudgetSummaryStatusPending = "pending"
)

const (
	BudgetRolloverStatusCarried = "carried"
	BudgetRolloverStatusSkipped = "skipped"
)
