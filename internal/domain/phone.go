package domain

import (
	"errors"
	"strings"
)

var ErrInvalidPhone = errors.New("invalid Brazilian mobile phone")

var validBrazilianAreaCodes = map[string]struct{}{
	"11": {}, "12": {}, "13": {}, "14": {}, "15": {}, "16": {}, "17": {}, "18": {}, "19": {},
	"21": {}, "22": {}, "24": {}, "27": {}, "28": {},
	"31": {}, "32": {}, "33": {}, "34": {}, "35": {}, "37": {}, "38": {},
	"41": {}, "42": {}, "43": {}, "44": {}, "45": {}, "46": {}, "47": {}, "48": {}, "49": {},
	"51": {}, "53": {}, "54": {}, "55": {},
	"61": {}, "62": {}, "63": {}, "64": {}, "65": {}, "66": {}, "67": {}, "68": {}, "69": {},
	"71": {}, "73": {}, "74": {}, "75": {}, "77": {}, "79": {},
	"81": {}, "82": {}, "83": {}, "84": {}, "85": {}, "86": {}, "87": {}, "88": {}, "89": {},
	"91": {}, "92": {}, "93": {}, "94": {}, "95": {}, "96": {}, "97": {}, "98": {}, "99": {},
}

func NormalizeBrazilianPhone(value string) string {
	var digits strings.Builder
	digits.Grow(len(value))
	for _, character := range value {
		if character >= '0' && character <= '9' {
			digits.WriteRune(character)
		}
	}
	return digits.String()
}

func IsValidBrazilianMobilePhone(value string) bool {
	digits := NormalizeBrazilianPhone(value)
	if len(digits) != 11 || digits[2] != '9' {
		return false
	}
	_, exists := validBrazilianAreaCodes[digits[:2]]
	return exists
}
