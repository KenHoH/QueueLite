package validation

import (
	"regexp"
	"strings"
)

var emailPattern = regexp.MustCompile(`^[^<>\s@]+@[^<>\s@]+\.[^<>\s@]+$`)
var indonesiaMobilePattern = regexp.MustCompile(`^628[0-9]{8,11}$`)

func Email(value string) bool { return emailPattern.MatchString(strings.TrimSpace(value)) }

// NormalizeIndonesiaPhone normalizes an Indonesian mobile phone number into 628... format.
func NormalizeIndonesiaPhone(value string) (string, bool) {
	normalized := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '(' || r == ')' || r == '.' || r == '-' {
			return -1
		}
		return r
	}, strings.TrimSpace(value))

	if strings.HasPrefix(normalized, "+62") {
		normalized = "62" + strings.TrimPrefix(normalized, "+62")
	} else if strings.HasPrefix(normalized, "08") {
		normalized = "62" + strings.TrimPrefix(normalized, "0")
	}

	if !indonesiaMobilePattern.MatchString(normalized) {
		return "", false
	}
	return normalized, true
}

func Phone(value string) bool {
	_, ok := NormalizeIndonesiaPhone(value)
	return ok
}
