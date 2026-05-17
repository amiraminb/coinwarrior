package service

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/amiraminb/coinwarrior/internal/daterange"
	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/amiraminb/coinwarrior/internal/money"
)

func findBudgetIndex(budgets []model.Budget, monthKey, currency string) int {
	return slices.IndexFunc(budgets, func(b model.Budget) bool {
		return b.Month == monthKey && strings.EqualFold(b.Currency, currency)
	})
}

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

func (s *Service) SetMonthlyBudget(monthInput, currency, amountInput string) (model.Budget, error) {
	return s.setMonthlyBudgetWithNow(monthInput, currency, amountInput, nil, time.Now())
}

func (s *Service) SetMonthlyBudgetWithCarryover(monthInput, currency, amountInput string, carryover bool) (model.Budget, error) {
	return s.setMonthlyBudgetWithNow(monthInput, currency, amountInput, &carryover, time.Now())
}

func (s *Service) setMonthlyBudgetWithNow(monthInput, currency, amountInput string, carryoverDecision *bool, now time.Time) (model.Budget, error) {
	month, err := daterange.ParseMonth(monthInput, now)
	if err != nil {
		return model.Budget{}, err
	}

	cur := money.NormalizeCurrency(currency)
	if cur == "" {
		return model.Budget{}, fmt.Errorf("currency is required")
	}

	amountMinor, err := money.Parse(amountInput)
	if err != nil {
		return model.Budget{}, err
	}
	if amountMinor <= 0 {
		return model.Budget{}, fmt.Errorf("budget amount must be greater than zero")
	}

	budgets, err := s.repo.LoadBudgets()
	if err != nil {
		return model.Budget{}, err
	}
	transactions, err := s.repo.LoadTransactions()
	if err != nil {
		return model.Budget{}, err
	}

	monthKey := daterange.FormatMonth(month)
	carryover, sourceIndex, err := budgetCarryoverCandidate(budgets, transactions, month, cur, now)
	if err != nil {
		return model.Budget{}, err
	}
	nowUTC := now.UTC().Format(time.RFC3339)
	targetIndex := findBudgetIndex(budgets, monthKey, cur)

	if targetIndex == -1 {
		budgets = append(budgets, model.Budget{
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
				return model.Budget{}, fmt.Errorf("budget for %s %s already has rollover from %s", monthKey, cur, from)
			}
			budgets[targetIndex].RolloverMinor = carryover.LeftMinor
			budgets[targetIndex].RolloverFromMonth = carryover.SourceBudget.Month
			budgets[targetIndex].UpdatedAt = nowUTC

			budgets[sourceIndex].RolloverStatus = model.BudgetRolloverStatusCarried
			budgets[sourceIndex].RolledOverMinor = carryover.LeftMinor
			budgets[sourceIndex].RolledOverIntoMonth = monthKey
			budgets[sourceIndex].RolledOverAt = nowUTC
			budgets[sourceIndex].UpdatedAt = nowUTC
		} else {
			budgets[sourceIndex].RolloverStatus = model.BudgetRolloverStatusSkipped
			budgets[sourceIndex].RolledOverMinor = 0
			budgets[sourceIndex].RolledOverIntoMonth = ""
			budgets[sourceIndex].RolledOverAt = nowUTC
			budgets[sourceIndex].UpdatedAt = nowUTC
		}
	}

	if err := s.repo.SaveBudgets(budgets); err != nil {
		return model.Budget{}, err
	}

	return budgets[targetIndex], nil
}

func (s *Service) GetBudgetCarryoverCandidate(monthInput, currency string, now time.Time) (*BudgetCarryoverCandidate, error) {
	month, err := daterange.ParseMonth(monthInput, now)
	if err != nil {
		return nil, err
	}

	cur := money.NormalizeCurrency(currency)
	if cur == "" {
		return nil, fmt.Errorf("currency is required")
	}

	budgets, err := s.repo.LoadBudgets()
	if err != nil {
		return nil, err
	}
	transactions, err := s.repo.LoadTransactions()
	if err != nil {
		return nil, err
	}

	candidate, _, err := budgetCarryoverCandidate(budgets, transactions, month, cur, now)
	if err != nil {
		return nil, err
	}
	return candidate, nil
}

func (s *Service) GetMonthlyBudgetSummaries(monthInput string, now time.Time) ([]BudgetSummary, error) {
	month, err := daterange.ParseMonth(monthInput, now)
	if err != nil {
		return nil, err
	}

	budgets, err := s.repo.LoadBudgets()
	if err != nil {
		return nil, err
	}
	transactions, err := s.repo.LoadTransactions()
	if err != nil {
		return nil, err
	}

	return summarizeBudgetsForMonth(budgets, transactions, month, now)
}

func (s *Service) GetPendingBudgetRollovers(targetMonthInput string, now time.Time) ([]BudgetSummary, error) {
	budgets, err := s.repo.LoadBudgets()
	if err != nil {
		return nil, err
	}
	transactions, err := s.repo.LoadTransactions()
	if err != nil {
		return nil, err
	}

	var targetMonth string
	if strings.TrimSpace(targetMonthInput) != "" {
		month, err := daterange.ParseMonth(targetMonthInput, now)
		if err != nil {
			return nil, err
		}
		targetMonth = daterange.FormatMonth(month)
	}

	summaries := make([]BudgetSummary, 0)
	today := daterange.DateOnly(now)
	for _, budget := range budgets {
		if targetMonth != "" && budget.Month != targetMonth {
			continue
		}
		if strings.TrimSpace(budget.RolloverStatus) != "" {
			continue
		}

		month, err := daterange.ParseMonth(budget.Month, now)
		if err != nil {
			return nil, err
		}
		_, end := daterange.MonthBounds(month)
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
			Status:      model.BudgetSummaryStatusPending,
			PeriodStart: month,
			PeriodEnd:   end,
		})
	}

	sortBudgetSummaries(summaries)
	return summaries, nil
}

func (s *Service) ApplyMonthlyBudgetRollover(monthInput, currency string, carry bool, now time.Time) (model.Budget, *model.Budget, error) {
	month, err := daterange.ParseMonth(monthInput, now)
	if err != nil {
		return model.Budget{}, nil, err
	}
	monthKey := daterange.FormatMonth(month)
	cur := money.NormalizeCurrency(currency)
	if cur == "" {
		return model.Budget{}, nil, fmt.Errorf("currency is required")
	}

	budgets, err := s.repo.LoadBudgets()
	if err != nil {
		return model.Budget{}, nil, err
	}
	transactions, err := s.repo.LoadTransactions()
	if err != nil {
		return model.Budget{}, nil, err
	}

	index := findBudgetIndex(budgets, monthKey, cur)
	if index == -1 {
		return model.Budget{}, nil, fmt.Errorf("budget for %s %s not found", monthKey, cur)
	}

	_, end := daterange.MonthBounds(month)
	if !end.Before(daterange.DateOnly(now)) {
		return model.Budget{}, nil, fmt.Errorf("budget period %s is still open", monthKey)
	}
	if strings.TrimSpace(budgets[index].RolloverStatus) != "" {
		return model.Budget{}, nil, fmt.Errorf("budget for %s %s already has rollover decision '%s'", monthKey, cur, budgets[index].RolloverStatus)
	}

	left, err := budgetLeftForMonth(transactions, budgets[index], month)
	if err != nil {
		return model.Budget{}, nil, err
	}

	nowUTC := now.UTC().Format(time.RFC3339)
	if !carry {
		budgets[index].RolloverStatus = model.BudgetRolloverStatusSkipped
		budgets[index].RolledOverMinor = 0
		budgets[index].RolledOverIntoMonth = ""
		budgets[index].RolledOverAt = nowUTC
		budgets[index].UpdatedAt = nowUTC
		if err := s.repo.SaveBudgets(budgets); err != nil {
			return model.Budget{}, nil, err
		}
		return budgets[index], nil, nil
	}

	nextMonth := month.AddDate(0, 1, 0)
	nextMonthKey := daterange.FormatMonth(nextMonth)
	destIndex := findBudgetIndex(budgets, nextMonthKey, cur)

	if destIndex == -1 {
		budgets = append(budgets, model.Budget{
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
			return model.Budget{}, nil, fmt.Errorf("budget for %s %s already has rollover from %s", nextMonthKey, cur, from)
		}
		budgets[destIndex].RolloverMinor = left
		budgets[destIndex].RolloverFromMonth = monthKey
		budgets[destIndex].UpdatedAt = nowUTC
	}

	budgets[index].RolloverStatus = model.BudgetRolloverStatusCarried
	budgets[index].RolledOverMinor = left
	budgets[index].RolledOverIntoMonth = nextMonthKey
	budgets[index].RolledOverAt = nowUTC
	budgets[index].UpdatedAt = nowUTC

	if err := s.repo.SaveBudgets(budgets); err != nil {
		return model.Budget{}, nil, err
	}

	source := budgets[index]
	destination := budgets[destIndex]
	return source, &destination, nil
}

func summarizeBudgetsForMonth(budgets []model.Budget, transactions []model.Transaction, month time.Time, now time.Time) ([]BudgetSummary, error) {
	start, end := daterange.MonthBounds(month)
	monthKey := daterange.FormatMonth(month)
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
	start, end := daterange.MonthBounds(month)
	spent := int64(0)
	for _, tx := range transactions {
		if tx.Type != model.TransactionTypeExpense {
			continue
		}
		if !strings.EqualFold(tx.Currency, budget.Currency) {
			continue
		}
		inRange, err := daterange.Contains(tx.Date, start, end)
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
	if periodEnd.Before(daterange.DateOnly(now)) {
		return model.BudgetSummaryStatusPending
	}
	return model.BudgetSummaryStatusOpen
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
	previousMonthKey := daterange.FormatMonth(previousMonth)

	sourceIndex := findBudgetIndex(budgets, previousMonthKey, currency)
	if sourceIndex == -1 {
		return nil, -1, nil
	}
	if strings.TrimSpace(budgets[sourceIndex].RolloverStatus) != "" {
		return nil, -1, nil
	}

	_, end := daterange.MonthBounds(previousMonth)
	if !end.Before(daterange.DateOnly(now)) {
		return nil, -1, nil
	}

	left, err := budgetLeftForMonth(transactions, budgets[sourceIndex], previousMonth)
	if err != nil {
		return nil, -1, err
	}

	return &BudgetCarryoverCandidate{
		SourceBudget: budgets[sourceIndex],
		TargetMonth:  daterange.FormatMonth(targetMonth),
		LeftMinor:    left,
	}, sourceIndex, nil
}
