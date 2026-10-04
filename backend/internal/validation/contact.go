package validation

import (
	"regexp"
	"strings"
)

var emailPattern = regexp.MustCompile(`^[^<>\s@]+@[^<>\s@]+\.[^<>\s@]+$`)
var phonePattern = regexp.MustCompile(`^\+?[0-9]{8,15}$`)

func Email(value string) bool { return emailPattern.MatchString(strings.TrimSpace(value)) }
func Phone(value string) bool {
	value = strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == '(' || r == ')' || r == '.' || r == '-' {
			return -1
		}
		return r
	}, value)
	return phonePattern.MatchString(value)
}
