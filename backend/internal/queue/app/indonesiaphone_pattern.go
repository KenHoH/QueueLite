package app

import (
	"QueueLite/internal/apperror"
	"regexp"
	"strings"
)

func NormalizeIndonesiaPhone(phone string) (string, error) {
	normalized := strings.TrimSpace(phone)
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "-", "")

	if strings.HasPrefix(normalized, "+62") {
		normalized = "62" + strings.TrimPrefix(normalized, "+62")
	} else if strings.HasPrefix(normalized, "08") {
		normalized = "62" + strings.TrimPrefix(normalized, "0")
	}

	if !IndonesiaPhonePattern.MatchString(normalized) {
		return "", apperror.New(apperror.KindInvalid, "INVALID_PHONE_NUMBER", "phone number must be a valid Indonesian mobile number")
	}
	return normalized, nil
}

var IndonesiaPhonePattern = regexp.MustCompile(`^628[0-9]{8,11}$`)
