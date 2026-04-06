package model

type Transaction struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	AmountMinor int64    `json:"amount_minor"`
	Currency    string   `json:"currency"`
	Date        string   `json:"date"`
	Category    string   `json:"category"`
	Account     string   `json:"account"`
	Tags        []string `json:"tags,omitempty"`
	Note        string   `json:"note,omitempty"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	Source      string   `json:"source"`
}

type TransactionsFile struct {
	SchemaVersion int           `json:"schema_version"`
	Transactions  []Transaction `json:"transactions"`
}
