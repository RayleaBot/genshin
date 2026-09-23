package app

import (
	"math/big"
	"unicode/utf16"
)

// JSFixed is JavaScript's toFixed: the exact value rounded half up.
func JSFixed(value float64, digits int) string {
	scaled := new(big.Float).SetPrec(256).SetFloat64(value)
	scaled.Mul(scaled, new(big.Float).SetPrec(256).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(digits)), nil)))
	scaled.Add(scaled, new(big.Float).SetPrec(256).SetFloat64(0.5))
	whole, _ := scaled.Int(nil)
	text := whole.String()
	if digits == 0 {
		return text
	}
	for len(text) <= digits {
		text = "0" + text
	}
	return text[:len(text)-digits] + "." + text[len(text)-digits:]
}

// jsLength is a string's JavaScript length, in UTF-16 code units.
func jsLength(text string) int {
	return len(utf16.Encode([]rune(text)))
}
