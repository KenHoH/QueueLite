package app

import (
	"QueueLite/internal/apperror"
	"QueueLite/internal/validation"
	"regexp"
)

func NormalizeIndonesiaPhone(phone string) (string, error) {
	normalized, ok := validation.NormalizeIndonesiaPhone(phone)
	if !ok {
		return "", apperror.New(apperror.KindInvalid, "INVALID_PHONE_NUMBER", "phone number must be a valid Indonesian mobile number")
	}
	return normalized, nil
}

var IndonesiaPhonePattern = regexp.MustCompile(`^628[0-9]{8,11}$`)
