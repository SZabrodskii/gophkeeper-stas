package model

import "testing"

func TestValidateLuhn_Valid(t *testing.T) {
	valid := []string{
		"4532015112830366",
		"4111111111111111",
		"5500000000000004",
		"3400 0000 0000 009",
		"3714-4963-5398-431",
	}
	for _, num := range valid {
		if !ValidateLuhn(num) {
			t.Errorf("ValidateLuhn(%q) = false, want true", num)
		}
	}
}

func TestValidateLuhn_Invalid(t *testing.T) {
	invalid := []string{
		"1234567890",
		"4532015112830365",
		"4532015112830367",
		"abc",
		"1",
		"",
	}
	for _, num := range invalid {
		if ValidateLuhn(num) {
			t.Errorf("ValidateLuhn(%q) = true, want false", num)
		}
	}
}

func TestValidateExpiry_Valid(t *testing.T) {
	valid := []string{"12/25", "01/30", "06/99", "01/00"}
	for _, exp := range valid {
		if !ValidateExpiry(exp) {
			t.Errorf("ValidateExpiry(%q) = false, want true", exp)
		}
	}
}

func TestValidateExpiry_Invalid(t *testing.T) {
	invalid := []string{"13/25", "00/25", "1225", "ab/cd", "1/25", "12/2", "12/abc", "/25", "12/"}
	for _, exp := range invalid {
		if ValidateExpiry(exp) {
			t.Errorf("ValidateExpiry(%q) = true, want false", exp)
		}
	}
}
