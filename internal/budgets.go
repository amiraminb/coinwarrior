package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/amiraminb/coinwarrior/internal/model"
)

const (
	budgetMonthLayout = "2006-01"

	budgetRolloverStatusCarried = "carried"
	budgetRolloverStatusSkipped = "skipped"
)

type BudgetSummary struct {
	Budget      model.Budget
	SpentMinor  int64
	LeftMinor   int64
	Status      string
	PeriodStart time.Time
	PeriodEnd   time.Time
}

type BudgetCarryoverCandidate struct {
	SourceBudget model.Budget
	TargetMonth  string
	LeftMinor    int64
}

func LoadBudgetsFile(path string) (model.BudgetsFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return model.BudgetsFile{SchemaVersion: 1, Budgets: []model.Budget{}}, nil
		}
		return model.BudgetsFile{}, err
	}

	var budgets model.BudgetsFile
	if err := json.Unmarshal(data, &budgets); err != nil {
		return model.BudgetsFile{}, err
	}
	if budgets.Budgets == nil {
		budgets.Budgets = []model.Budget{}
	}

	return budgets, nil
}

func SaveBudgetsFile(path string, budgets model.BudgetsFile) error {
	if budgets.Budgets == nil {
		budgets.Budgets = []model.Budget{}
	}

	data, err := json.MarshalIndent(budgets, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}

func SetMonthlyBudget(monthInput, currency, amountInput string) (model.Budget, error) {
	return setMonthlyBudgetWithNow(monthInput, currency, amountInput, nil, time.Now())
}

func SetMonthlyBudgetWithCarryover(monthInput, currency, amountInput string, carryover bool) (model.Budget, error) {
	return setMonthlyBudgetWithNow(monthInput, currency, amountInput, &carryover, time.Now())
}

func setMonthlyBudgetWithNow(monthInput, currency, amountInput string, carryoverDecision *bool, now time.Time) (model.Budget, error) {
	month, err := ParseBudgetMonth(monthInput, now)
	if err != nil {
		return model.Budget{}, err
	}

	cur := strings.ToUpper(strings.TrimSpace(currency))
	if cur == "" {
		return model.Budget{}, fmt.Errorf("currency is required")
	}

	amountMinor, err := ParseAmount(amountInput)
	if err != nil {
		return model.Budget{}, err
	}
	if amountMinor <= 0 {
		return model.Budget{}, fmt.Errorf("budget amount must be greater than zero")
	}

	path, err := FilePath(BudgetsFileName)
	if err != nil {
		return model.Budget{}, err
	}

	budgetsFile, err := LoadBudgetsFile(path)
	if err != nil {
		return model.Budget{}, err
	}
	transactionsPath, err := FilePath(TransactionsFileName)
	if err != nil {
		return model.Budget{}, err
	}
	transactionsFile, err := LoadTransactions(transactionsPath)
	if err != nil {
		return model.Budget{}, err
	}

	monthKey := FormatBudgetMonth(month)
	carryover, sourceIndex, err := budgetCarryoverCandidate(budgetsFile.Budgets, transactionsFile.Transactions, month, cur, now)
	if err != nil {
		return model.Budget{}, err
	}
	nowUTC := now.UTC().Format(time.RFC3339)
	targetIndex := -1
	for i := range budgetsFile.Budgets {
		if budgetsFile.Budgets[i].Month == monthKey && strings.EqualFold(budgetsFile.Budgets[i].Currency, cur) {
			targetIndex = i
			break
		}
	}

	if targetIndex == -1 {
		budgetsFile.Budgets = append(budgetsFile.Budgets, model.Budget{
			Month:       monthKey,
			Currency:    cur,
			AmountMinor: amountMinor,
			UpdatedAt:   nowUTC,
		})
		targetIndex = len(budgetsFile.Budgets) - 1
	} else {
		budgetsFile.Budgets[targetIndex].Currency = cur
		budgetsFile.Budgets[targetIndex].AmountMinor = amountMinor
		budgetsFile.Budgets[targetIndex].UpdatedAt = nowUTC
	}

	if carryover != nil && carryoverDecision != nil {
		if *carryoverDecision {
			if from := strings.TrimSpace(budgetsFile.Budgets[targetIndex].RolloverFromMonth); from != "" && from != carryover.SourceBudget.Month {
				return model.Budget{}, fmt.Errorf("budget for %s %s already has rollover from %s", monthKey, cur, from)
			}
			budgetsFile.Budgets[targetIndex].RolloverMinor = carryover.LeftMinor
			budgetsFile.Budgets[targetIndex].RolloverFromMonth = carryover.SourceBudget.Month
			budgetsFile.Budgets[targetIndex].UpdatedAt = nowUTC

			budgetsFile.Budgets[sourceIndex].RolloverStatus = budgetRolloverStatusCarried
			budgetsFile.Budgets[sourceIndex].RolledOverMinor = carryover.LeftMinor
			budgetsFile.Budgets[sourceIndex].RolledOverIntoMonth = monthKey
			budgetsFile.Budgets[sourceIndex].RolledOverAt = nowUTC
			budgetsFile.Budgets[sourceIndex].UpdatedAt = nowUTC
		} else {
			budgetsFile.Budgets[sourceIndex].RolloverStatus = budgetRolloverStatusSkipped
			budgetsFile.Budgets[sourceIndex].RolledOverMinor = 0
			budgetsFile.Budgets[sourceIndex].RolledOverIntoMonth = ""
			budgetsFile.Budgets[sourceIndex].RolledOverAt = nowUTC
			budgetsFile.Budgets[sourceIndex].UpdatedAt = nowUTC
		}
	}

	if err := SaveBudgetsFile(path, budgetsFile); err != nil {
		return model.Budget{}, err
	}

	return budgetsFile.Budgets[targetIndex], nil
}

func GetBudgetCarryoverCandidate(monthInput, currency string, now time.Time) (*BudgetCarryoverCandidate, error) {
	month, err := ParseBudgetMonth(monthInput, now)
	if err != nil {
		return nil, err
	}

	cur := strings.ToUpper(strings.TrimSpace(currency))
	if cur == "" {
		return nil, fmt.Errorf("currency is required")
	}

	budgetsPath, err := FilePath(BudgetsFileName)
	if err != nil {
		return nil, err
	}
	transactionsPath, err := FilePath(TransactionsFileName)
	if err != nil {
		return nil, err
	}

	budgetsFile, err := LoadBudgetsFile(budgetsPath)
	if err != nil {
		return nil, err
	}
	transactionsFile, err := LoadTransactions(transactionsPath)
	if err != nil {
		return nil, err
	}

	candidate, _, err := budgetCarryoverCandidate(budgetsFile.Budgets, transactionsFile.Transactions, month, cur, now)
	if err != nil {
		return nil, err
	}
	return candidate, nil
}

func GetMonthlyBudgetSummaries(monthInput string, now time.Time) ([]BudgetSummary, error) {
	month, err := ParseBudgetMonth(monthInput, now)
	if err != nil {
		return nil, err
	}

	budgetsPath, err := FilePath(BudgetsFileName)
	if err != nil {
		return nil, err
	}
	transactionsPath, err := FilePath(TransactionsFileName)
	if err != nil {
		return nil, err
	}

	budgetsFile, err := LoadBudgetsFile(budgetsPath)
	if err != nil {
		return nil, err
	}
	transactionsFile, err := LoadTransactions(transactionsPath)
	if err != nil {
		return nil, err
	}

	return summarizeBudgetsForMonth(budgetsFile.Budgets, transactionsFile.Transactions, month, now)
}

func GetPendingBudgetRollovers(targetMonthInput string, now time.Time) ([]BudgetSummary, error) {
	budgetsPath, err := FilePath(BudgetsFileName)
	if err != nil {
		return nil, err
	}
	transactionsPath, err := FilePath(TransactionsFileName)
	if err != nil {
		return nil, err
	}

	budgetsFile, err := LoadBudgetsFile(budgetsPath)
	if err != nil {
		return nil, err
	}
	transactionsFile, err := LoadTransactions(transactionsPath)
	if err != nil {
		return nil, err
	}

	var targetMonth string
	if strings.TrimSpace(targetMonthInput) != "" {
		month, err := ParseBudgetMonth(targetMonthInput, now)
		if err != nil {
			return nil, err
		}
		targetMonth = FormatBudgetMonth(month)
	}

	summaries := make([]BudgetSummary, 0)
	today := dateOnly(now)
	for _, budget := range budgetsFile.Budgets {
		if targetMonth != "" && budget.Month != targetMonth {
			continue
		}
		if strings.TrimSpace(budget.RolloverStatus) != "" {
			continue
		}

		month, err := ParseBudgetMonth(budget.Month, now)
		if err != nil {
			return nil, err
		}
		_, end := budgetMonthBounds(month)
		if !end.Before(today) {
			continue
		}

		spent, err := expensesForBudgetMonth(transactionsFile.Transactions, budget, month)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, BudgetSummary{
			Budget:      budget,
			SpentMinor:  spent,
			LeftMinor:   budget.AmountMinor + budget.RolloverMinor - spent,
			Status:      "pending",
			PeriodStart: month,
			PeriodEnd:   end,
		})
	}

	sortBudgetSummaries(summaries)
	return summaries, nil
}

func ApplyMonthlyBudgetRollover(monthInput, currency string, carry bool, now time.Time) (model.Budget, *model.Budget, error) {
	month, err := ParseBudgetMonth(monthInput, now)
	if err != nil {
		return model.Budget{}, nil, err
	}
	monthKey := FormatBudgetMonth(month)
	cur := strings.ToUpper(strings.TrimSpace(currency))
	if cur == "" {
		return model.Budget{}, nil, fmt.Errorf("currency is required")
	}

	budgetsPath, err := FilePath(BudgetsFileName)
	if err != nil {
		return model.Budget{}, nil, err
	}
	transactionsPath, err := FilePath(TransactionsFileName)
	if err != nil {
		return model.Budget{}, nil, err
	}

	budgetsFile, err := LoadBudgetsFile(budgetsPath)
	if err != nil {
		return model.Budget{}, nil, err
	}
	transactionsFile, err := LoadTransactions(transactionsPath)
	if err != nil {
		return model.Budget{}, nil, err
	}

	index := -1
	for i := range budgetsFile.Budgets {
		if budgetsFile.Budgets[i].Month == monthKey && strings.EqualFold(budgetsFile.Budgets[i].Currency, cur) {
			index = i
			break
		}
	}
	if index == -1 {
		return model.Budget{}, nil, fmt.Errorf("budget for %s %s not found", monthKey, cur)
	}

	_, end := budgetMonthBounds(month)
	if !end.Before(dateOnly(now)) {
		return model.Budget{}, nil, fmt.Errorf("budget period %s is still open", monthKey)
	}
	if strings.TrimSpace(budgetsFile.Budgets[index].RolloverStatus) != "" {
		return model.Budget{}, nil, fmt.Errorf("budget for %s %s already has rollover decision '%s'", monthKey, cur, budgetsFile.Budgets[index].RolloverStatus)
	}

	left, err := budgetLeftForMonth(transactionsFile.Transactions, budgetsFile.Budgets[index], month)
	if err != nil {
		return model.Budget{}, nil, err
	}

	nowUTC := now.UTC().Format(time.RFC3339)
	if !carry {
		budgetsFile.Budgets[index].RolloverStatus = budgetRolloverStatusSkipped
		budgetsFile.Budgets[index].RolledOverMinor = 0
		budgetsFile.Budgets[index].RolledOverIntoMonth = ""
		budgetsFile.Budgets[index].RolledOverAt = nowUTC
		budgetsFile.Budgets[index].UpdatedAt = nowUTC
		if err := SaveBudgetsFile(budgetsPath, budgetsFile); err != nil {
			return model.Budget{}, nil, err
		}
		return budgetsFile.Budgets[index], nil, nil
	}

	nextMonth := month.AddDate(0, 1, 0)
	nextMonthKey := FormatBudgetMonth(nextMonth)
	destIndex := -1
	for i := range budgetsFile.Budgets {
		if budgetsFile.Budgets[i].Month == nextMonthKey && strings.EqualFold(budgetsFile.Budgets[i].Currency, cur) {
			destIndex = i
			break
		}
	}

	if destIndex == -1 {
		budgetsFile.Budgets = append(budgetsFile.Budgets, model.Budget{
			Month:             nextMonthKey,
			Currency:          cur,
			AmountMinor:       0,
			RolloverMinor:     left,
			RolloverFromMonth: monthKey,
			UpdatedAt:         nowUTC,
		})
		destIndex = len(budgetsFile.Budgets) - 1
	} else {
		if from := strings.TrimSpace(budgetsFile.Budgets[destIndex].RolloverFromMonth); from != "" && from != monthKey {
			return model.Budget{}, nil, fmt.Errorf("budget for %s %s already has rollover from %s", nextMonthKey, cur, from)
		}
		budgetsFile.Budgets[destIndex].RolloverMinor = left
		budgetsFile.Budgets[destIndex].RolloverFromMonth = monthKey
		budgetsFile.Budgets[destIndex].UpdatedAt = nowUTC
	}

	budgetsFile.Budgets[index].RolloverStatus = budgetRolloverStatusCarried
	budgetsFile.Budgets[index].RolledOverMinor = left
	budgetsFile.Budgets[index].RolledOverIntoMonth = nextMonthKey
	budgetsFile.Budgets[index].RolledOverAt = nowUTC
	budgetsFile.Budgets[index].UpdatedAt = nowUTC

	if err := SaveBudgetsFile(budgetsPath, budgetsFile); err != nil {
		return model.Budget{}, nil, err
	}

	source := budgetsFile.Budgets[index]
	destination := budgetsFile.Budgets[destIndex]
	return source, &destination, nil
}

func ParseBudgetMonth(input string, now time.Time) (time.Time, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()), nil
	}

	month, err := time.ParseInLocation(budgetMonthLayout, trimmed, now.Location())
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid month format: %s", input)
	}

	return time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, now.Location()), nil
}

func FormatBudgetMonth(month time.Time) string {
	return time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, month.Location()).Format(budgetMonthLayout)
}

func summarizeBudgetsForMonth(budgets []model.Budget, transactions []model.Transaction, month time.Time, now time.Time) ([]BudgetSummary, error) {
	start, end := budgetMonthBounds(month)
	monthKey := FormatBudgetMonth(month)
	summaries := make([]BudgetSummary, 0)

	for _, budget := range budgets {
		if budget.Month != monthKey {
			continue
		}

		spent, err := expensesForBudgetMonth(transactions, budget, month)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, BudgetSummary{
			Budget:      budget,
			SpentMinor:  spent,
			LeftMinor:   budget.AmountMinor + budget.RolloverMinor - spent,
			Status:      budgetSummaryStatus(budget, end, now),
			PeriodStart: start,
			PeriodEnd:   end,
		})
	}

	sortBudgetSummaries(summaries)
	return summaries, nil
}

func budgetLeftForMonth(transactions []model.Transaction, budget model.Budget, month time.Time) (int64, error) {
	spent, err := expensesForBudgetMonth(transactions, budget, month)
	if err != nil {
		return 0, err
	}
	return budget.AmountMinor + budget.RolloverMinor - spent, nil
}

func expensesForBudgetMonth(transactions []model.Transaction, budget model.Budget, month time.Time) (int64, error) {
	start, end := budgetMonthBounds(month)
	spent := int64(0)
	for _, tx := range transactions {
		if tx.Type != TransactionTypeExpense {
			continue
		}
		if !strings.EqualFold(tx.Currency, budget.Currency) {
			continue
		}
		inRange, err := TransactionInRange(tx.Date, start, end)
		if err != nil {
			return 0, err
		}
		if inRange {
			spent += tx.AmountMinor
		}
	}

	return spent, nil
}

func budgetSummaryStatus(budget model.Budget, periodEnd, now time.Time) string {
	if strings.TrimSpace(budget.RolloverStatus) != "" {
		return budget.RolloverStatus
	}
	if periodEnd.Before(dateOnly(now)) {
		return "pending"
	}
	return "open"
}

func budgetMonthBounds(month time.Time) (time.Time, time.Time) {
	start := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, month.Location())
	end := start.AddDate(0, 1, -1)
	return start, end
}

func sortBudgetSummaries(summaries []BudgetSummary) {
	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Budget.Month == summaries[j].Budget.Month {
			return summaries[i].Budget.Currency < summaries[j].Budget.Currency
		}
		return summaries[i].Budget.Month < summaries[j].Budget.Month
	})
}

func budgetCarryoverCandidate(budgets []model.Budget, transactions []model.Transaction, targetMonth time.Time, currency string, now time.Time) (*BudgetCarryoverCandidate, int, error) {
	previousMonth := targetMonth.AddDate(0, -1, 0)
	previousMonthKey := FormatBudgetMonth(previousMonth)

	sourceIndex := -1
	for i := range budgets {
		if budgets[i].Month == previousMonthKey && strings.EqualFold(budgets[i].Currency, currency) {
			sourceIndex = i
			break
		}
	}
	if sourceIndex == -1 {
		return nil, -1, nil
	}
	if strings.TrimSpace(budgets[sourceIndex].RolloverStatus) != "" {
		return nil, -1, nil
	}

	_, end := budgetMonthBounds(previousMonth)
	if !end.Before(dateOnly(now)) {
		return nil, -1, nil
	}

	left, err := budgetLeftForMonth(transactions, budgets[sourceIndex], previousMonth)
	if err != nil {
		return nil, -1, err
	}

	return &BudgetCarryoverCandidate{
		SourceBudget: budgets[sourceIndex],
		TargetMonth:  FormatBudgetMonth(targetMonth),
		LeftMinor:    left,
	}, sourceIndex, nil
}
