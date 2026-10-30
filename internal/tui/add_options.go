package tui

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/amiraminb/coinwarrior/internal/model"
)

// AddTransactionOptions supplies fields to skip in the entry form. A nil field
// opens its prompt; a non-nil Note may be empty to explicitly omit the note.
type AddTransactionOptions struct {
	Type     *string
	Amount   *string
	Date     *string
	Category *string
	Account  *string
	Note     *string
}

var addPromptSteps = []addStep{
	stepType, stepAmount, stepDate, stepCategorySelect, stepAccountSelect, stepNote,
}

func newAddModelWithOptions(categories, accounts []string, options AddTransactionOptions) (addModel, error) {
	m := newAddModel(categories, accounts)
	m.preset = make(map[addStep]bool)
	for _, field := range []struct {
		name   string
		value  *string
		target *string
		step   addStep
	}{
		{"type", options.Type, &m.selected, stepType},
		{"amount", options.Amount, &m.amountInput, stepAmount},
		{"date", options.Date, &m.dateInput, stepDate},
		{"category", options.Category, &m.categoryInput, stepCategorySelect},
		{"account", options.Account, &m.accountInput, stepAccountSelect},
		{"note", options.Note, &m.noteInput, stepNote},
	} {
		if field.value == nil {
			continue
		}
		value := strings.TrimSpace(*field.value)
		if value == "" && field.step != stepNote {
			return addModel{}, fmt.Errorf("--%s cannot be empty", field.name)
		}
		switch field.step {
		case stepType:
			value = strings.ToLower(value)
			if value != model.TransactionTypeExpense && value != model.TransactionTypeIncome {
				return addModel{}, fmt.Errorf("invalid --type %q (use expense or income)", value)
			}
		case stepAmount:
			if err := validateAmount(value); err != nil {
				return addModel{}, fmt.Errorf("invalid --amount: %w", err)
			}
		case stepDate:
			if _, err := time.Parse("2006-01-02", value); err != nil {
				return addModel{}, fmt.Errorf("invalid --date %q (use YYYY-MM-DD)", value)
			}
		case stepCategorySelect, stepAccountSelect:
			names := categories
			if field.step == stepAccountSelect {
				names = accounts
			}
			i := slices.IndexFunc(names, func(name string) bool { return strings.EqualFold(name, value) })
			if i < 0 {
				return addModel{}, fmt.Errorf("unknown %s %q (available: %s)", field.name, value, strings.Join(names, ", "))
			}
			value = names[i]
		}
		*field.target = value
		m.preset[field.step] = true
	}
	m.skipPresetSteps(stepType)
	return m, nil
}

// Skip supplied fields in either direction, including when Esc goes back. If
// every earlier field was supplied, stay at the current prompt.
func (m *addModel) skipPresetSteps(previous addStep) {
	direction := 1
	if m.step < previous {
		direction = -1
	}
	for m.preset[m.step] {
		i := slices.Index(addPromptSteps, m.step) + direction
		if i < 0 {
			m.step = previous
			return
		}
		if i == len(addPromptSteps) {
			m.step = stepDone
			return
		}
		m.step = addPromptSteps[i]
	}
}
