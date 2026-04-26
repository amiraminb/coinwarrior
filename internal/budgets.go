package internal

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/amiraminb/coinwarrior/internal/domain"
	"github.com/amiraminb/coinwarrior/internal/repository"
)

const (
	budgetMonthLayout = "2006-01"

	budgetRolloverStatusCarried = "carried"
	budgetRolloverStatusSkipped = "skipped"
)

type BudgetSummary struct {
	Budget      domain.Budget
	SpentMinor  int64
	LeftMinor   int64
	Status      string
	PeriodStart time.Time
	PeriodEnd   time.Time
}

type BudgetCarryoverCandidate struct {
	SourceBudget domain.Budget
	TargetMonth  string
	LeftMinor    int64
}

func SetMonthlyBudget(monthInput, currency, amountInput string) (domain.Budget, error) {
	return setMonthlyBudgetWithNow(monthInput, currency, amountInput, nil, time.Now())
}

func SetMonthlyBudgetWithCarryover(monthInput, currency, amountInput string, carryover bool) (domain.Budget, error) {
	return setMonthlyBudgetWithNow(monthInput, currency, amountInput, &carryover, time.Now())
}

func setMonthlyBudgetWithNow(monthInput, currency, amountInput string, carryoverDecision *bool, now time.Time) (domain.Budget, error) {
	month, err := ParseBudgetMonth(monthInput, now)
	if err != nil {
		return domain.Budget{}, err
	}

	cur := strings.ToUpper(strings.TrimSpace(currency))
	if cur == "" {
		return domain.Budget{}, fmt.Errorf("currency is required")
	}

	amountMinor, err := ParseAmount(amountInput)
	if err != nil {
		return domain.Budget{}, err
	}
	if amountMinor <= 0 {
		return domain.Budget{}, fmt.Errorf("budget amount must be greater than zero")
	}

	budgets, err := repository.FRepository.LoadBudgets()
	if err != nil {
		return domain.Budget{}, err
	}
	transactions, err := repository.FRepository.LoadTransactions()
	if err != nil {
		return domain.Budget{}, err
	}

	monthKey := FormatBudgetMonth(month)
	carryover, sourceIndex, err := budgetCarryoverCandidate(budgets, transactions, month, cur, now)
	if err != nil {
		return domain.Budget{}, err
	}
	nowUTC := now.UTC().Format(time.RFC3339)
	targetIndex := -1
	for i := range budgets {
		if budgets[i].Month == monthKey && strings.EqualFold(budgets[i].Currency, cur) {
			targetIndex = i
			break
		}
	}

	if targetIndex == -1 {
		budgets = append(budgets, domain.Budget{
			Month:       monthKey,
			Currency:    cur,
			AmountMinor: amountMinor,
			UpdatedAt:   nowUTC,
		})
		targetIndex = len(budgets) - 1
	} else {
		budgets[targetIndex].Currency = cur
		budgets[targetIndex].AmountMinor = amountMinor
		budgets[targetIndex].UpdatedAt = nowUTC
	}

	if carryover != nil && carryoverDecision != nil {
		if *carryoverDecision {
			if from := strings.TrimSpace(budgets[targetIndex].RolloverFromMonth); from != "" && from != carryover.SourceBudget.Month {
				return domain.Budget{}, fmt.Errorf("budget for %s %s already has rollover from %s", monthKey, cur, from)
			}
			budgets[targetIndex].RolloverMinor = carryover.LeftMinor
			budgets[targetIndex].RolloverFromMonth = carryover.SourceBudget.Month
			budgets[targetIndex].UpdatedAt = nowUTC

			budgets[sourceIndex].RolloverStatus = budgetRolloverStatusCarried
			budgets[sourceIndex].RolledOverMinor = carryover.LeftMinor
			budgets[sourceIndex].RolledOverIntoMonth = monthKey
			budgets[sourceIndex].RolledOverAt = nowUTC
			budgets[sourceIndex].UpdatedAt = nowUTC
		} else {
			budgets[sourceIndex].RolloverStatus = budgetRolloverStatusSkipped
			budgets[sourceIndex].RolledOverMinor = 0
			budgets[sourceIndex].RolledOverIntoMonth = ""
			budgets[sourceIndex].RolledOverAt = nowUTC
			budgets[sourceIndex].UpdatedAt = nowUTC
		}
	}

	if err := repository.FRepository.SaveBudgets(budgets); err != nil {
		return domain.Budget{}, err
	}

	return budgets[targetIndex], nil
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

	budgets, err := repository.FRepository.LoadBudgets()
	if err != nil {
		return nil, err
	}
	transactions, err := repository.FRepository.LoadTransactions()
	if err != nil {
		return nil, err
	}

	candidate, _, err := budgetCarryoverCandidate(budgets, transactions, month, cur, now)
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

	budgets, err := repository.FRepository.LoadBudgets()
	if err != nil {
		return nil, err
	}
	transactions, err := repository.FRepository.LoadTransactions()
	if err != nil {
		return nil, err
	}

	return summarizeBudgetsForMonth(budgets, transactions, month, now)
}

func GetPendingBudgetRollovers(targetMonthInput string, now time.Time) ([]BudgetSummary, error) {
	budgets, err := repository.FRepository.LoadBudgets()
	if err != nil {
		return nil, err
	}
	transactions, err := repository.FRepository.LoadTransactions()
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
	for _, budget := range budgets {
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

		spent, err := expensesForBudgetMonth(transactions, budget, month)
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

func ApplyMonthlyBudgetRollover(monthInput, currency string, carry bool, now time.Time) (domain.Budget, *domain.Budget, error) {
	month, err := ParseBudgetMonth(monthInput, now)
	if err != nil {
		return domain.Budget{}, nil, err
	}
	monthKey := FormatBudgetMonth(month)
	cur := strings.ToUpper(strings.TrimSpace(currency))
	if cur == "" {
		return domain.Budget{}, nil, fmt.Errorf("currency is required")
	}

	budgets, err := repository.FRepository.LoadBudgets()
	if err != nil {
		return domain.Budget{}, nil, err
	}
	transactions, err := repository.FRepository.LoadTransactions()
	if err != nil {
		return domain.Budget{}, nil, err
	}

	index := -1
	for i := range budgets {
		if budgets[i].Month == monthKey && strings.EqualFold(budgets[i].Currency, cur) {
			index = i
			break
		}
	}
	if index == -1 {
		return domain.Budget{}, nil, fmt.Errorf("budget for %s %s not found", monthKey, cur)
	}

	_, end := budgetMonthBounds(month)
	if !end.Before(dateOnly(now)) {
		return domain.Budget{}, nil, fmt.Errorf("budget period %s is still open", monthKey)
	}
	if strings.TrimSpace(budgets[index].RolloverStatus) != "" {
		return domain.Budget{}, nil, fmt.Errorf("budget for %s %s already has rollover decision '%s'", monthKey, cur, budgets[index].RolloverStatus)
	}

	left, err := budgetLeftForMonth(transactions, budgets[index], month)
	if err != nil {
		return domain.Budget{}, nil, err
	}

	nowUTC := now.UTC().Format(time.RFC3339)
	if !carry {
		budgets[index].RolloverStatus = budgetRolloverStatusSkipped
		budgets[index].RolledOverMinor = 0
		budgets[index].RolledOverIntoMonth = ""
		budgets[index].RolledOverAt = nowUTC
		budgets[index].UpdatedAt = nowUTC
		if err := repository.FRepository.SaveBudgets(budgets); err != nil {
			return domain.Budget{}, nil, err
		}
		return budgets[index], nil, nil
	}

	nextMonth := month.AddDate(0, 1, 0)
	nextMonthKey := FormatBudgetMonth(nextMonth)
	destIndex := -1
	for i := range budgets {
		if budgets[i].Month == nextMonthKey && strings.EqualFold(budgets[i].Currency, cur) {
			destIndex = i
			break
		}
	}

	if destIndex == -1 {
		budgets = append(budgets, domain.Budget{
			Month:             nextMonthKey,
			Currency:          cur,
			AmountMinor:       0,
			RolloverMinor:     left,
			RolloverFromMonth: monthKey,
			UpdatedAt:         nowUTC,
		})
		destIndex = len(budgets) - 1
	} else {
		if from := strings.TrimSpace(budgets[destIndex].RolloverFromMonth); from != "" && from != monthKey {
			return domain.Budget{}, nil, fmt.Errorf("budget for %s %s already has rollover from %s", nextMonthKey, cur, from)
		}
		budgets[destIndex].RolloverMinor = left
		budgets[destIndex].RolloverFromMonth = monthKey
		budgets[destIndex].UpdatedAt = nowUTC
	}

	budgets[index].RolloverStatus = budgetRolloverStatusCarried
	budgets[index].RolledOverMinor = left
	budgets[index].RolledOverIntoMonth = nextMonthKey
	budgets[index].RolledOverAt = nowUTC
	budgets[index].UpdatedAt = nowUTC

	if err := repository.FRepository.SaveBudgets(budgets); err != nil {
		return domain.Budget{}, nil, err
	}

	source := budgets[index]
	destination := budgets[destIndex]
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

func summarizeBudgetsForMonth(budgets []domain.Budget, transactions []domain.Transaction, month time.Time, now time.Time) ([]BudgetSummary, error) {
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

func budgetLeftForMonth(transactions []domain.Transaction, budget domain.Budget, month time.Time) (int64, error) {
	spent, err := expensesForBudgetMonth(transactions, budget, month)
	if err != nil {
		return 0, err
	}
	return budget.AmountMinor + budget.RolloverMinor - spent, nil
}

func expensesForBudgetMonth(transactions []domain.Transaction, budget domain.Budget, month time.Time) (int64, error) {
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

func budgetSummaryStatus(budget domain.Budget, periodEnd, now time.Time) string {
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

func budgetCarryoverCandidate(budgets []domain.Budget, transactions []domain.Transaction, targetMonth time.Time, currency string, now time.Time) (*BudgetCarryoverCandidate, int, error) {
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
