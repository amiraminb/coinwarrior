package domain

type Transaction struct {
	ID          string   `json:"id"`
	Type        string   `json:"type"`
	AmountMinor int64    `json:"amount_minor"`
	Currency    string   `json:"currency"`
	Date        string   `json:"date"`
	Category    string   `json:"category"`
	Account     string   `json:"account"`
	ToAccount   string   `json:"to_account,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Note        string   `json:"note,omitempty"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
	Source      string   `json:"source"`
}

type Account struct {
	Name         string `json:"name"`
	Currency     string `json:"currency"`
	BalanceMinor int64  `json:"balance_minor"`
	UpdatedAt    string `json:"updated_at"`
}

type Budget struct {
	Month               string `json:"month"`
	Currency            string `json:"currency"`
	AmountMinor         int64  `json:"amount_minor"`
	RolloverMinor       int64  `json:"rollover_minor,omitempty"`
	RolloverFromMonth   string `json:"rollover_from_month,omitempty"`
	RolloverStatus      string `json:"rollover_status,omitempty"`
	RolledOverMinor     int64  `json:"rolled_over_minor,omitempty"`
	RolledOverIntoMonth string `json:"rolled_over_into_month,omitempty"`
	RolledOverAt        string `json:"rolled_over_at,omitempty"`
	UpdatedAt           string `json:"updated_at"`
}
