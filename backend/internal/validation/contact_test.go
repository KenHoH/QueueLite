package validation

import "testing"

func TestContactFormats(t *testing.T) {
	for _, phone := range []string{"081234567890", "+62 812-3456-7890", "6281234567890", "(021) 1234-5678"} {
		if !Phone(phone) {
			t.Errorf("valid phone %q", phone)
		}
	}
	for _, phone := range []string{"", "   ", "abc", "08", "1234567890123456", "+62<script>"} {
		if Phone(phone) {
			t.Errorf("invalid phone %q", phone)
		}
	}
	for _, email := range []string{"ayu@example.test", "ayu+clinic@example.test", " ayu@example.test "} {
		if !Email(email) {
			t.Errorf("valid email %q", email)
		}
	}
	for _, email := range []string{"", " ", "bad", "a@b", "a b@example.test", "<a@example.test>"} {
		if Email(email) {
			t.Errorf("invalid email %q", email)
		}
	}
}
