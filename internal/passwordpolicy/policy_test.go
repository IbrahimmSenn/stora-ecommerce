package passwordpolicy

import (
	"errors"
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	cases := []struct {
		name string
		pw   string
		ok   bool
	}{
		{"all rules met", "Abcdef1!", true},
		{"too short", "Ab1!", false},
		{"no uppercase", "abcdef1!", false},
		{"no lowercase", "ABCDEF1!", false},
		{"no digit", "Abcdefg!", false},
		{"no symbol", "Abcdefg1", false},
		{"empty", "", false},
		{"too long", strings.Repeat("Ab1!", 20), false}, // 80 > 72
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := Validate(c.pw)
			if c.ok && err != nil {
				t.Fatalf("expected ok, got %v", err)
			}
			if !c.ok && !errors.Is(err, ErrWeak) {
				t.Fatalf("expected ErrWeak, got %v", err)
			}
		})
	}
}
