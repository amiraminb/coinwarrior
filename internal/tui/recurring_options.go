package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/amiraminb/coinwarrior/internal/model"
	"github.com/amiraminb/coinwarrior/internal/money"
	"github.com/amiraminb/coinwarrior/internal/service"
)

// Nil fields are omitted flags; explicit empty notes and end dates clear them.
func normalizeRecurringOptions(options service.RecurringRuleEdits, categories, accounts []string) (service.RecurringRuleEdits, error) {
	for _, field := range []struct {
		name  string
		value **string
	}{
		{"type", &options.Type}, {"amount", &options.AmountInput},
		{"currency", &options.Currency}, {"category", &options.Category},
		{"account", &options.Account}, {"to-account", &options.ToAccount},
		{"start-date", &options.StartDate}, {"end-date", &options.EndDate}, {"note", &options.Note},
	} {
		if *field.value == nil {
			continue
		}
		value := strings.TrimSpace(**field.value)
		if value == "" && field.name != "note" && field.name != "end-date" {
			return options, fmt.Errorf("--%s cannot be empty", field.name)
		}
		switch field.name {
		case "type":
			value = strings.ToLower(value)
			if value != model.TransactionTypeExpense && value != model.TransactionTypeIncome && value != model.TransactionTypeTransfer {
				return options, fmt.Errorf("invalid --type %q (use expense, income, or transfer)", value)
			}
		case "amount":
			if err := validateAmount(value); err != nil {
				return options, fmt.Errorf("invalid --amount: %w", err)
			}
		case "currency":
			value = money.NormalizeCurrency(value)
		case "category", "account", "to-account":
			names := accounts
			if field.name == "category" {
				names = categories
				if options.Type != nil && *options.Type == model.TransactionTypeTransfer && strings.EqualFold(value, model.TransferCategory) {
					value = model.TransferCategory
					break
				}
			}
			i := indexOfFold(names, value)
			if i < 0 {
				return options, fmt.Errorf("unknown %s %q (available: %s)", field.name, value, strings.Join(names, ", "))
			}
			value = names[i]
		case "start-date", "end-date":
			if value != "" {
				if _, err := time.Parse("2006-01-02", value); err != nil {
					return options, fmt.Errorf("invalid --%s %q (use YYYY-MM-DD)", field.name, value)
				}
			}
		}
		*field.value = &value
	}
	if options.DayOfMonth != nil && (*options.DayOfMonth < 1 || *options.DayOfMonth > 28) {
		return options, fmt.Errorf("--day must be between 1 and 28")
	}
	if options.StartDate != nil && options.EndDate != nil && *options.EndDate != "" && *options.EndDate < *options.StartDate {
		return options, fmt.Errorf("end date is before start date")
	}
	if options.Type != nil && *options.Type != model.TransactionTypeTransfer && options.ToAccount != nil {
		return options, fmt.Errorf("--to-account requires a transfer rule")
	}
	if options.Type != nil && *options.Type == model.TransactionTypeTransfer && options.Category != nil && *options.Category != model.TransferCategory {
		return options, fmt.Errorf("transfer rules use the Transfer category")
	}
	if options.Account != nil && options.ToAccount != nil && *options.Account == *options.ToAccount {
		return options, fmt.Errorf("source and destination accounts must be different")
	}
	return options, nil
}

func newRecurringAddModelWithOptions(categories, accounts []string, options service.RecurringRuleEdits) (recurringFormModel, error) {
	m := newRecurringAddFormModel(categories, accounts)
	if options == (service.RecurringRuleEdits{}) {
		return m, nil
	}
	options, err := normalizeRecurringOptions(options, categories, accounts)
	if err != nil {
		return m, err
	}
	// Flag-based entry uses defaults for optional fields. The bare add command
	// keeps the full interactive form, including these prompts.
	m.preset = map[recurringField]bool{
		recurringFieldCurrency: true, recurringFieldStartDate: true,
		recurringFieldEndDate: true, recurringFieldNote: true,
	}
	for _, field := range []struct {
		value  *string
		target *string
		step   recurringField
	}{
		{options.AmountInput, &m.amountInput, recurringFieldAmount},
		{options.Currency, &m.currencyInput, recurringFieldCurrency},
		{options.StartDate, &m.startInput, recurringFieldStartDate},
		{options.EndDate, &m.endInput, recurringFieldEndDate},
		{options.Note, &m.noteInput, recurringFieldNote},
	} {
		if field.value != nil {
			*field.target = *field.value
			m.preset[field.step] = true
		}
	}
	for _, field := range []struct {
		value   *string
		choices []string
		cursor  *int
		step    recurringField
	}{
		{options.Type, m.typeChoices, &m.typeCursor, recurringFieldType},
		{options.Category, categories, &m.categoryCursor, recurringFieldCategory},
		{options.Account, accounts, &m.accountCursor, recurringFieldAccount},
		{options.ToAccount, accounts, &m.toAccountCursor, recurringFieldToAccount},
	} {
		if field.value != nil {
			*field.cursor = indexOfFold(field.choices, *field.value)
			m.preset[field.step] = true
		}
	}
	if options.DayOfMonth != nil {
		m.dayInput = strconv.Itoa(*options.DayOfMonth)
		m.preset[recurringFieldDayOfMonth] = true
	}
	if m.endInput != "" && m.endInput < m.startInput {
		return m, fmt.Errorf("end date is before start date")
	}
	m.skipPresetSteps(recurringFieldType)
	return m, nil
}

// Follow the form's field order in either direction, skipping flag-supplied
// values and fields that do not apply to the currently selected type.
func (m *recurringFormModel) skipPresetSteps(previous recurringField) {
	direction := recurringField(1)
	if m.step < previous {
		direction = -1
	}
	transfer := m.typeChoices[m.typeCursor] == model.TransactionTypeTransfer
	for m.step < recurringFieldConfirm && (m.preset[m.step] ||
		(transfer && m.step == recurringFieldCategory) ||
		(!transfer && m.step == recurringFieldToAccount)) {
		if m.step+direction < recurringFieldType {
			m.step = previous
			return
		}
		m.step += direction
	}
}
