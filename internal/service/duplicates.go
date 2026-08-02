package service

import (
	"strings"

	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/amiraminb/coinwarrior/internal/money"
)

// Transfers are never reported: they all share model.TransferCategory and
// legitimately repeat, so matching them would warn on routine activity.
func (s *Service) FindPossibleDuplicates(txType, amountInput, dateValue, category string) ([]model.Transaction, error) {
	if txType == model.TransactionTypeTransfer {
		return nil, nil
	}

	amountMinor, err := money.Parse(amountInput)
	if err != nil {
		return nil, err
	}

	date := strings.TrimSpace(dateValue)
	category = strings.TrimSpace(category)

	transactions, err := s.repo.LoadTransactions()
	if err != nil {
		return nil, err
	}

	var matches []model.Transaction
	for _, tx := range transactions {
		if tx.Type != txType || tx.AmountMinor != amountMinor || tx.Date != date {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(tx.Category), category) {
			continue
		}
		matches = append(matches, tx)
	}

	return matches, nil
}
