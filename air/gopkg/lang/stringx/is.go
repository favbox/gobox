package stringx

import "unicode"

// IsAlpha 检查字符串是否只包含unicode字母。
func IsAlpha(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

// IsAlphaNumber 检查字符串是否只包含unicode字母或数字。
func IsAlphaNumber(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !isAlphanumeric(r) {
			return false
		}
	}
	return true
}

// IsNumeric 检查字符串是否只包含数字。
func IsNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func isAlphanumeric(r rune) bool {
	return unicode.IsDigit(r) || unicode.IsLetter(r)
}
