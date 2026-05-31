package money

import (
	"fmt"
	"math"
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

	// Accept thousands separators in the whole-number part so values copied from
	// formatted output (e.g. "1,234.50") round-trip through Parse. Commas in the
	// fractional part remain invalid.
	wholePart := strings.ReplaceAll(parts[0], ",", "")
	fracPart := ""
	if len(parts) == 2 {
		fracPart = parts[1]
	}

	// Each part must be digits only (an empty part is allowed, e.g. ".5" or "5."),
	// and at least one digit must be present. This rejects forms like "." or "--5"
	// that strconv would otherwise read as a silent zero or a flipped sign.
	if !isDigits(wholePart) || !isDigits(fracPart) {
		return 0, fmt.Errorf("invalid amount: %s", input)
	}
	if wholePart == "" && fracPart == "" {
		return 0, fmt.Errorf("invalid amount: %s", input)
	}
	if len(fracPart) > 2 {
		return 0, fmt.Errorf("amount supports max 2 decimals: %s", input)
	}

	whole := wholePart
	if whole == "" {
		whole = "0"
	}
	wholeValue, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount: %s", input)
	}

	frac := fracPart
	for len(frac) < 2 {
		frac += "0"
	}
	fracValue, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount: %s", input)
	}

	// Guard before computing wholeValue*100+fracValue, which would otherwise
	// silently wrap past int64 into a negative balance.
	if wholeValue > (math.MaxInt64-fracValue)/100 {
		return 0, fmt.Errorf("amount is too large: %s", input)
	}

	result := wholeValue*100 + fracValue
	if negative {
		result = -result
	}

	return result, nil
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
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
