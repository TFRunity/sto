// Package vehicle implements vehicle registration and card lookup.
package vehicle

import (
	"strings"
	"unicode"
)

var plateLetters = strings.NewReplacer(
	"A", "А", "B", "В", "E", "Е", "K", "К", "M", "М", "H", "Н",
	"O", "О", "P", "Р", "C", "С", "T", "Т", "Y", "У", "X", "Х",
)

func normalizePlate(number string) string {
	number = strings.Map(func(char rune) rune {
		if unicode.IsSpace(char) {
			return -1
		}
		return unicode.ToUpper(char)
	}, number)
	return plateLetters.Replace(number)
}
