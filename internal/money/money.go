package money

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/amiraminb/coinwarrior/internal/model"
)

func Parse(input string) (int64, error) {
	amount := strings.TrimSpace(input)
	if amount == "" {
		return 0, fmt.Errorf("amount cannot be empty")
	}

	negative := false
	if strings.HasPrefix(amount, "-") {
		negative = true
		amount = strings.TrimSpace(strings.TrimPrefix(amount, "-"))
		if amount == "" {
			return 0, fmt.Errorf("invalid amount: %s", input)
		}
	}

	parts := strings.Split(amount, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid amount format: %s", input)
	}

	whole := parts[0]
	if whole == "" {
		whole = "0"
	}

	wholeValue, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount: %s", input)
	}

	frac := "00"
	if len(parts) == 2 {
		if len(parts[1]) > 2 {
			return 0, fmt.Errorf("amount supports max 2 decimals: %s", input)
		}
		frac = parts[1]
		if len(frac) == 1 {
			frac += "0"
		}
		if len(frac) == 0 {
			frac = "00"
		}
	}

	fracValue, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount: %s", input)
	}

	result := wholeValue*100 + fracValue
	if negative {
		result = -result
	}

	return result, nil
}

func Format(amountMinor int64) string {
	negative := amountMinor < 0
	if negative {
		amountMinor = -amountMinor
	}

	whole := amountMinor / 100
	fraction := amountMinor % 100
	wholeFormatted := formatWithCommas(whole)

	if negative {
		return fmt.Sprintf("-%s.%02d", wholeFormatted, fraction)
	}

	return fmt.Sprintf("%s.%02d", wholeFormatted, fraction)
}

func FormatTransaction(tx model.Transaction) string {
	amount := Format(tx.AmountMinor)
	if tx.Type == model.TransactionTypeExpense {
		amount = "-" + amount
	}
	return amount
}

func NormalizeCurrency(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

func formatWithCommas(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}

	first := len(s) % 3
	if first == 0 {
		first = 3
	}

	result := s[:first]
	for i := first; i < len(s); i += 3 {
		result += "," + s[i:i+3]
	}

	return result
}
