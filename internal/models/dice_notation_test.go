package models

import (
	"testing"
)

func TestParseDiceNotation(t *testing.T) {
	tests := []struct {
		name      string
		notation  string
		wantTerms int
		wantErr   bool
	}{
		{"empty string", "", 0, false},
		{"simple die", "1d6", 1, false},
		{"multiple dice", "2d8", 1, false},
		{"dice plus bonus", "2d6+3", 2, false},
		{"mixed dice", "1d6+1d4+3", 3, false},
		{"whitespace ignored", " 2d6 + 3 ", 2, false},
		{"uppercase ignored", "2D6+3", 2, false},
		{"negative bonus", "1d6-2", 2, false},
		{"d4", "1d4", 1, false},
		{"d8", "1d8", 1, false},
		{"d10", "1d10", 1, false},
		{"d12", "1d12", 1, false},
		{"d20", "1d20", 1, false},
		{"unsupported die size", "1d7", 0, true},
		{"invalid format", "abc", 0, true},
		{"invalid dice count", "xd6", 0, true},
		{"invalid die size", "2dx", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDiceNotation(tt.notation)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDiceNotation(%q) error = %v, wantErr %v", tt.notation, err, tt.wantErr)
				return
			}
			if err == nil && len(got.Terms) != tt.wantTerms {
				t.Errorf("ParseDiceNotation(%q) got %d terms, want %d", tt.notation, len(got.Terms), tt.wantTerms)
			}
		})
	}
}

func TestParseDiceNotationTermValues(t *testing.T) {
	dn, err := ParseDiceNotation("2d6+1d4+3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(dn.Terms) != 3 {
		t.Fatalf("want 3 terms, got %d", len(dn.Terms))
	}

	if dn.Terms[0].Count != 2 || dn.Terms[0].Size != 6 {
		t.Errorf("term 0: want 2d6, got %dd%d", dn.Terms[0].Count, dn.Terms[0].Size)
	}
	if dn.Terms[1].Count != 1 || dn.Terms[1].Size != 4 {
		t.Errorf("term 1: want 1d4, got %dd%d", dn.Terms[1].Count, dn.Terms[1].Size)
	}
	if dn.Terms[2].Count != 0 || dn.Terms[2].Size != 3 {
		t.Errorf("term 2: want flat +3, got count=%d size=%d", dn.Terms[2].Count, dn.Terms[2].Size)
	}
}

func TestParseDiceNotationNegativeBonus(t *testing.T) {
	dn, err := ParseDiceNotation("1d6-2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(dn.Terms) != 2 {
		t.Fatalf("want 2 terms, got %d", len(dn.Terms))
	}

	if dn.Terms[1].Count != 0 || dn.Terms[1].Size != -2 {
		t.Errorf("want flat -2, got count=%d size=%d", dn.Terms[1].Count, dn.Terms[1].Size)
	}
}

func TestDiceNotationString(t *testing.T) {
	tests := []struct {
		notation string
		want     string
	}{
		{"2d6+3", "2d6+3"},
		{"1d8", "1d8"},
		{"", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.notation, func(t *testing.T) {
			dn, err := ParseDiceNotation(tt.notation)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			got := dn.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

type mockRoller struct {
	fixedRoll int
}

func (m *mockRoller) RollDamage(diceCount, diceSize, flatBonus int) int {
	return diceCount*m.fixedRoll + flatBonus
}

func (m *mockRoller) RollD6() int {
	return m.fixedRoll
}

func TestDiceNotationRoll(t *testing.T) {
	roller := &mockRoller{fixedRoll: 3}

	dn, err := ParseDiceNotation("2d6+5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := dn.Roll(roller)
	want := 6 + 5
	if got != want {
		t.Errorf("Roll() = %d, want %d (2 dice * 3 each + 5 flat)", got, want)
	}
}

func TestDiceNotationRollEmpty(t *testing.T) {
	roller := &mockRoller{fixedRoll: 3}

	dn, err := ParseDiceNotation("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := dn.Roll(roller)
	if got != 0 {
		t.Errorf("Roll() on empty notation = %d, want 0", got)
	}
}
