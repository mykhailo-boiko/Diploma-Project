package controller

import "testing"

func TestIsValidCustomerName(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect bool
	}{
		{"empty", "", false},
		{"too short single char", "A", false},
		{"plain ASCII", "John Doe", true},
		{"with apostrophe", "O'Brien", true},
		{"with hyphen", "Jean-Luc", true},
		{"with diacritic Latin-1", "Café Müller", true},
		{"with Latin Extended-A", "Łukasz Świątek", true},
		{"with Latin Extended-B", "Ǎlexander", true},
		{"with dot", "St. John", true},
		{"Cyrillic rejected", "Іван Петренко", false},
		{"Chinese rejected", "張三", false},
		{"HTML script rejected", "<script>alert(1)</script>", false},
		{"SQL injection rejected", "X'; DROP TABLE;--", false},
		{"angle brackets rejected", "John<test>Doe", false},
		{"quote rejected", "John\"Doe", false},
		{"backtick rejected", "John`Doe", false},
		{"semicolon rejected", "John;Doe", false},
		{"leading space rejected", " John Doe", false},
		{"only spaces rejected", "   ", false},
		{"too long rejected", string(make([]byte, 250)), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidCustomerName(tt.input)
			if got != tt.expect {
				t.Errorf("isValidCustomerName(%q) = %v, want %v", tt.input, got, tt.expect)
			}
		})
	}
}
