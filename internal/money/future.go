package money

import (
	"math"
	"math/big"
)

const (
	FutureValueInterestPercent = 5
	interestNumerator          = 105
	interestDenominator        = 100
)

// FutureValueHorizons are the projections shown to users.
func FutureValueHorizons() []int {
	return []int{5, 10, 15, 20}
}

// FutureValueMinor compounds an amount annually at 5%, rounding to the nearest
// minor unit only after all years have been applied.
func FutureValueMinor(amountMinor int64, years int) int64 {
	if years <= 0 || amountMinor == 0 {
		return amountMinor
	}

	negative := amountMinor < 0
	amount := new(big.Int).SetInt64(amountMinor)
	if negative {
		amount.Abs(amount)
	}

	power := new(big.Int).Exp(
		big.NewInt(interestNumerator),
		big.NewInt(int64(years)),
		nil,
	)
	numerator := new(big.Int).Mul(amount, power)
	denominator := new(big.Int).Exp(
		big.NewInt(interestDenominator),
		big.NewInt(int64(years)),
		nil,
	)

	value, remainder := new(big.Int).QuoRem(numerator, denominator, new(big.Int))
	if new(big.Int).Mul(remainder, big.NewInt(2)).Cmp(denominator) >= 0 {
		value.Add(value, big.NewInt(1))
	}
	if negative {
		value.Neg(value)
	}

	max := big.NewInt(math.MaxInt64)
	if value.Cmp(max) > 0 {
		return math.MaxInt64
	}
	min := big.NewInt(math.MinInt64)
	if value.Cmp(min) < 0 {
		return math.MinInt64
	}
	return value.Int64()
}
