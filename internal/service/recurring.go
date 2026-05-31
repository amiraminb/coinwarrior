package service

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/amiraminb/coinwarrior/internal/daterange"
	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/amiraminb/coinwarrior/internal/money"
)

const (
	recurringDayOfMonthMin = 1
	recurringDayOfMonthMax = 28
)

type RecurringRuleInput struct {
	Type        string
	AmountInput string
	Currency    string
	Category    string
	Account     string
	ToAccount   string
	Note        string
	DayOfMonth  int
	StartDate   string
	EndDate     string
}

type RecurringRuleEdits struct {
	Type        *string
	AmountInput *string
	Currency    *string
	Category    *string
	Account     *string
	ToAccount   *string
	Note        *string
	DayOfMonth  *int
	StartDate   *string
	EndDate     *string
}

type GenerationResult struct {
	Generated  []model.Transaction
	RuleCounts map[string]int
}

func newRecurringRuleID(now time.Time) string {
	return fmt.Sprintf("rec_%d", now.UnixNano())
}

func findRecurringRuleIndex(rules []model.RecurringRule, id string) int {
	return slices.IndexFunc(rules, func(r model.RecurringRule) bool {
		return r.ID == id
	})
}

func (s *Service) LoadRecurringRules() ([]model.RecurringRule, error) {
	rules, err := s.repo.LoadRecurringRules()
	if err != nil {
		return nil, err
	}
	out := make([]model.RecurringRule, len(rules))
	copy(out, rules)
	return out, nil
}

func (s *Service) AddRecurringRule(input RecurringRuleInput) (model.RecurringRule, error) {
	now := time.Now()
	rule, err := buildRecurringRule(input, now)
	if err != nil {
		return model.RecurringRule{}, err
	}

	rules, err := s.repo.LoadRecurringRules()
	if err != nil {
		return model.RecurringRule{}, err
	}
	rules = append(rules, rule)
	if err := s.repo.SaveRecurringRules(rules); err != nil {
		return model.RecurringRule{}, err
	}
	return rule, nil
}

func (s *Service) EditRecurringRule(id string, edits RecurringRuleEdits) (model.RecurringRule, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return model.RecurringRule{}, fmt.Errorf("rule id is required")
	}
	rules, err := s.repo.LoadRecurringRules()
	if err != nil {
		return model.RecurringRule{}, err
	}
	idx := findRecurringRuleIndex(rules, id)
	if idx == -1 {
		return model.RecurringRule{}, fmt.Errorf("recurring rule '%s' not found", id)
	}

	now := time.Now()
	updated, changed, err := applyRecurringRuleEdits(rules[idx], edits, now)
	if err != nil {
		return model.RecurringRule{}, err
	}
	if !changed {
		return rules[idx], nil
	}
	rules[idx] = updated
	if err := s.repo.SaveRecurringRules(rules); err != nil {
		return model.RecurringRule{}, err
	}
	return updated, nil
}

func (s *Service) DeleteRecurringRule(id string) (model.RecurringRule, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return model.RecurringRule{}, fmt.Errorf("rule id is required")
	}
	rules, err := s.repo.LoadRecurringRules()
	if err != nil {
		return model.RecurringRule{}, err
	}
	idx := findRecurringRuleIndex(rules, id)
	if idx == -1 {
		return model.RecurringRule{}, fmt.Errorf("recurring rule '%s' not found", id)
	}
	deleted := rules[idx]
	rules = append(rules[:idx], rules[idx+1:]...)
	if err := s.repo.SaveRecurringRules(rules); err != nil {
		return model.RecurringRule{}, err
	}
	return deleted, nil
}

func (s *Service) GenerateDueTransactions(now time.Time) (GenerationResult, error) {
	result := GenerationResult{
		Generated:  []model.Transaction{},
		RuleCounts: make(map[string]int),
	}

	transactions, err := s.repo.LoadTransactions()
	if err != nil {
		return result, err
	}
	accounts, err := s.repo.LoadAccounts()
	if err != nil {
		return result, err
	}
	rules, err := s.repo.LoadRecurringRules()
	if err != nil {
		return result, err
	}

	originalAccounts := cloneAccounts(accounts)
	originalTransactions := cloneTransactions(transactions)
	nowUTC := now.UTC().Format(time.RFC3339)
	today := daterange.DateOnly(now)

	idCounter := 0
	for i := range rules {
		dueTxs, err := dueTransactionsForRule(rules[i], today, now, nowUTC, &idCounter)
		if err != nil {
			return GenerationResult{}, err
		}
		for _, tx := range dueTxs {
			if err := applyTransactionEffect(accounts, tx, nowUTC); err != nil {
				return GenerationResult{}, err
			}
			transactions = append(transactions, tx)
			result.Generated = append(result.Generated, tx)
			result.RuleCounts[rules[i].ID]++
		}
		if len(dueTxs) > 0 {
			lastTx := dueTxs[len(dueTxs)-1]
			rules[i].LastGeneratedMonth = monthKeyFromDate(lastTx.Date)
			rules[i].UpdatedAt = nowUTC
		}
	}

	if len(result.Generated) == 0 {
		return result, nil
	}

	// Save order matters: accounts and transactions first, then the rules whose
	// LastGeneratedMonth marks this work done. If the rules save fails, the already
	// persisted transactions must be rolled back — otherwise the next run sees
	// LastGeneratedMonth unchanged and regenerates the same transactions, double
	// applying their balance effects.
	if err := s.repo.SaveAccounts(accounts); err != nil {
		return GenerationResult{}, err
	}
	if err := s.repo.SaveTransactions(transactions); err != nil {
		return GenerationResult{}, s.rollback(err,
			func() error { return s.repo.SaveAccounts(originalAccounts) })
	}
	if err := s.repo.SaveRecurringRules(rules); err != nil {
		return GenerationResult{}, s.rollback(err,
			func() error { return s.repo.SaveTransactions(originalTransactions) },
			func() error { return s.repo.SaveAccounts(originalAccounts) })
	}

	return result, nil
}

func buildRecurringRule(input RecurringRuleInput, now time.Time) (model.RecurringRule, error) {
	txType := strings.TrimSpace(input.Type)
	if txType != model.TransactionTypeExpense && txType != model.TransactionTypeIncome && txType != model.TransactionTypeTransfer {
		return model.RecurringRule{}, fmt.Errorf("invalid transaction type: %s", txType)
	}

	amountMinor, err := money.Parse(input.AmountInput)
	if err != nil {
		return model.RecurringRule{}, err
	}
	if amountMinor <= 0 {
		return model.RecurringRule{}, fmt.Errorf("amount must be greater than zero")
	}

	currency := money.NormalizeCurrency(input.Currency)
	if currency == "" {
		return model.RecurringRule{}, fmt.Errorf("currency is required")
	}

	day := input.DayOfMonth
	if day < recurringDayOfMonthMin || day > recurringDayOfMonthMax {
		return model.RecurringRule{}, fmt.Errorf("day of month must be between %d and %d", recurringDayOfMonthMin, recurringDayOfMonthMax)
	}

	start := strings.TrimSpace(input.StartDate)
	if start == "" {
		start = now.Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", start); err != nil {
		return model.RecurringRule{}, fmt.Errorf("invalid start date: %s", input.StartDate)
	}

	end := strings.TrimSpace(input.EndDate)
	if end != "" {
		endTime, err := time.Parse("2006-01-02", end)
		if err != nil {
			return model.RecurringRule{}, fmt.Errorf("invalid end date: %s", input.EndDate)
		}
		startTime, _ := time.Parse("2006-01-02", start)
		if endTime.Before(startTime) {
			return model.RecurringRule{}, fmt.Errorf("end date is before start date")
		}
	}

	account := strings.TrimSpace(input.Account)
	toAccount := strings.TrimSpace(input.ToAccount)
	category := strings.TrimSpace(input.Category)

	if txType == model.TransactionTypeTransfer {
		if account == "" || toAccount == "" {
			return model.RecurringRule{}, fmt.Errorf("both source and destination accounts are required")
		}
		if strings.EqualFold(account, toAccount) {
			return model.RecurringRule{}, fmt.Errorf("source and destination accounts must be different")
		}
		if category == "" {
			category = model.TransferCategory
		}
	} else {
		if account == "" {
			return model.RecurringRule{}, fmt.Errorf("account is required")
		}
		toAccount = ""
	}

	utcNow := now.UTC().Format(time.RFC3339)
	return model.RecurringRule{
		ID:          newRecurringRuleID(now.UTC()),
		Type:        txType,
		AmountMinor: amountMinor,
		Currency:    currency,
		Category:    category,
		Account:     account,
		ToAccount:   toAccount,
		Note:        strings.TrimSpace(input.Note),
		DayOfMonth:  day,
		StartDate:   start,
		EndDate:     end,
		CreatedAt:   utcNow,
		UpdatedAt:   utcNow,
	}, nil
}

func applyRecurringRuleEdits(rule model.RecurringRule, edits RecurringRuleEdits, now time.Time) (model.RecurringRule, bool, error) {
	updated := rule
	if edits.Type != nil {
		updated.Type = *edits.Type
	}
	if edits.AmountInput != nil {
		amountMinor, err := money.Parse(*edits.AmountInput)
		if err != nil {
			return model.RecurringRule{}, false, err
		}
		if amountMinor <= 0 {
			return model.RecurringRule{}, false, fmt.Errorf("amount must be greater than zero")
		}
		updated.AmountMinor = amountMinor
	}
	if edits.Currency != nil {
		updated.Currency = money.NormalizeCurrency(*edits.Currency)
	}
	if edits.Category != nil {
		updated.Category = *edits.Category
	}
	if edits.Account != nil {
		updated.Account = *edits.Account
	}
	if edits.ToAccount != nil {
		updated.ToAccount = *edits.ToAccount
	}
	if edits.Note != nil {
		updated.Note = *edits.Note
	}
	if edits.DayOfMonth != nil {
		updated.DayOfMonth = *edits.DayOfMonth
	}
	if edits.StartDate != nil {
		updated.StartDate = *edits.StartDate
	}
	if edits.EndDate != nil {
		updated.EndDate = *edits.EndDate
	}

	updated.Type = strings.TrimSpace(updated.Type)
	updated.Currency = money.NormalizeCurrency(updated.Currency)
	updated.Account = strings.TrimSpace(updated.Account)
	updated.ToAccount = strings.TrimSpace(updated.ToAccount)
	updated.Category = strings.TrimSpace(updated.Category)
	updated.Note = strings.TrimSpace(updated.Note)
	updated.StartDate = strings.TrimSpace(updated.StartDate)
	updated.EndDate = strings.TrimSpace(updated.EndDate)

	if updated.Type != model.TransactionTypeExpense && updated.Type != model.TransactionTypeIncome && updated.Type != model.TransactionTypeTransfer {
		return model.RecurringRule{}, false, fmt.Errorf("invalid transaction type: %s", updated.Type)
	}
	if updated.Currency == "" {
		return model.RecurringRule{}, false, fmt.Errorf("currency is required")
	}
	if updated.DayOfMonth < recurringDayOfMonthMin || updated.DayOfMonth > recurringDayOfMonthMax {
		return model.RecurringRule{}, false, fmt.Errorf("day of month must be between %d and %d", recurringDayOfMonthMin, recurringDayOfMonthMax)
	}
	if updated.StartDate == "" {
		return model.RecurringRule{}, false, fmt.Errorf("start date is required")
	}
	if _, err := time.Parse("2006-01-02", updated.StartDate); err != nil {
		return model.RecurringRule{}, false, fmt.Errorf("invalid start date: %s", updated.StartDate)
	}
	if updated.EndDate != "" {
		endTime, err := time.Parse("2006-01-02", updated.EndDate)
		if err != nil {
			return model.RecurringRule{}, false, fmt.Errorf("invalid end date: %s", updated.EndDate)
		}
		startTime, _ := time.Parse("2006-01-02", updated.StartDate)
		if endTime.Before(startTime) {
			return model.RecurringRule{}, false, fmt.Errorf("end date is before start date")
		}
	}

	if updated.Type == model.TransactionTypeTransfer {
		if updated.Account == "" || updated.ToAccount == "" {
			return model.RecurringRule{}, false, fmt.Errorf("both source and destination accounts are required")
		}
		if strings.EqualFold(updated.Account, updated.ToAccount) {
			return model.RecurringRule{}, false, fmt.Errorf("source and destination accounts must be different")
		}
		if updated.Category == "" {
			updated.Category = model.TransferCategory
		}
	} else {
		if updated.Account == "" {
			return model.RecurringRule{}, false, fmt.Errorf("account is required")
		}
		updated.ToAccount = ""
	}

	changed := updated.Type != rule.Type ||
		updated.AmountMinor != rule.AmountMinor ||
		updated.Currency != rule.Currency ||
		updated.Category != rule.Category ||
		updated.Account != rule.Account ||
		updated.ToAccount != rule.ToAccount ||
		updated.Note != rule.Note ||
		updated.DayOfMonth != rule.DayOfMonth ||
		updated.StartDate != rule.StartDate ||
		updated.EndDate != rule.EndDate
	if !changed {
		return rule, false, nil
	}

	updated.UpdatedAt = now.UTC().Format(time.RFC3339)
	return updated, true, nil
}

func dueTransactionsForRule(rule model.RecurringRule, today, now time.Time, nowUTC string, idCounter *int) ([]model.Transaction, error) {
	startTime, err := time.Parse("2006-01-02", rule.StartDate)
	if err != nil {
		return nil, fmt.Errorf("rule %s: invalid start date: %s", rule.ID, rule.StartDate)
	}
	startMonth := time.Date(startTime.Year(), startTime.Month(), 1, 0, 0, 0, 0, today.Location())

	cursor := startMonth
	if rule.LastGeneratedMonth != "" {
		last, err := time.ParseInLocation("2006-01", rule.LastGeneratedMonth, today.Location())
		if err != nil {
			return nil, fmt.Errorf("rule %s: invalid last_generated_month: %s", rule.ID, rule.LastGeneratedMonth)
		}
		next := last.AddDate(0, 1, 0)
		if next.After(cursor) {
			cursor = next
		}
	}

	var endLimit *time.Time
	if rule.EndDate != "" {
		endTime, err := time.Parse("2006-01-02", rule.EndDate)
		if err != nil {
			return nil, fmt.Errorf("rule %s: invalid end date: %s", rule.ID, rule.EndDate)
		}
		endLimit = &endTime
	}

	var due []model.Transaction
	currentMonth := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, today.Location())
	for !cursor.After(currentMonth) {
		txDate := time.Date(cursor.Year(), cursor.Month(), rule.DayOfMonth, 0, 0, 0, 0, today.Location())
		if txDate.After(today) {
			break
		}
		if endLimit != nil && txDate.After(*endLimit) {
			break
		}
		startBoundary, _ := time.Parse("2006-01-02", rule.StartDate)
		if txDate.Before(startBoundary) {
			cursor = cursor.AddDate(0, 1, 0)
			continue
		}

		txTime := now.Add(time.Duration(*idCounter) * time.Nanosecond)
		*idCounter++
		tx := model.Transaction{
			ID:          NewTransactionID(txTime.UTC()),
			Type:        rule.Type,
			AmountMinor: rule.AmountMinor,
			Currency:    rule.Currency,
			Date:        txDate.Format("2006-01-02"),
			Category:    rule.Category,
			Account:     rule.Account,
			ToAccount:   rule.ToAccount,
			Note:        rule.Note,
			CreatedAt:   nowUTC,
			UpdatedAt:   nowUTC,
			Source:      model.TransactionSourceRecurring,
		}
		due = append(due, tx)
		cursor = cursor.AddDate(0, 1, 0)
	}

	return due, nil
}

func monthKeyFromDate(date string) string {
	if len(date) >= 7 {
		return date[:7]
	}
	return date
}
