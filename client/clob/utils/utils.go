package utils

import (
	"fmt"
	"math/big"

	"github.com/shopspring/decimal"
)

func RoundDown(x decimal.Decimal, sigDigits int) decimal.Decimal {
	multiplier := decimal.NewFromInt(10).Pow(decimal.NewFromInt(int64(sigDigits)))
	return x.Mul(multiplier).Floor().Div(multiplier)
}

func RoundNormal(x decimal.Decimal, sigDigits int) decimal.Decimal {
	multiplier := decimal.NewFromInt(10).Pow(decimal.NewFromInt(int64(sigDigits)))
	return x.Mul(multiplier).Round(0).Div(multiplier)
}

func RoundUp(x decimal.Decimal, sigDigits int) decimal.Decimal {
	multiplier := decimal.NewFromInt(10).Pow(decimal.NewFromInt(int64(sigDigits)))
	return x.Mul(multiplier).Ceil().Div(multiplier)
}

func DecimalPlaces(x decimal.Decimal) int {
	exp := x.Exponent()
	if exp >= 0 {
		return 0
	}
	return int(-exp)
}

func ToTokenDecimals(x decimal.Decimal) string {
	multiplier := decimal.NewFromInt(10).Pow(decimal.NewFromInt(6))
	f := x.Mul(multiplier)

	if DecimalPlaces(f) > 0 {
		f = RoundNormal(f, 0)
	}

	return fmt.Sprintf("%d", f.IntPart())
}

func parseBigInt(s string, base int) (*big.Int, error) {
	i := new(big.Int)
	_, ok := i.SetString(s, base)
	if !ok {
		return nil, fmt.Errorf("invalid numeric string: %s", s)
	}
	return i, nil
}

func MustBigInt(s string) (*big.Int, error) {
	return parseBigInt(s, 10)
}

func Prepend0x(s string) string {
	if len(s) >= 2 && s[:2] == "0x" {
		return s
	}
	return "0x" + s
}
