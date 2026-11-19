package models

import (
	"fmt"
	"strconv"
	"strings"
)

// DiceNotation represents parsed dice notation like "2d6+1d4+3"
type DiceNotation struct {
	Terms []DiceTerm
}

// DiceTerm is a single component of dice notation
type DiceTerm struct {
	Count int // Number of dice (or 0 for flat bonus)
	Size  int // Die size (d4, d6, etc.) or 0 for flat bonus
}

var validDiceSizes = map[int]bool{4: true, 6: true, 8: true, 10: true, 12: true, 20: true}

// ParseDiceNotation parses strings like "2d6+2", "1d6+1d6+6", "3d8"
func ParseDiceNotation(notation string) (*DiceNotation, error) {
	if notation == "" {
		return &DiceNotation{Terms: []DiceTerm{}}, nil
	}

	notation = strings.ReplaceAll(notation, " ", "")
	notation = strings.ToLower(notation)

	// Split by + and -
	parts := []string{}
	current := ""
	for i, char := range notation {
		if (char == '+' || char == '-') && i > 0 {
			parts = append(parts, current)
			if char == '-' {
				current = "-"
			} else {
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}

	terms := []DiceTerm{}

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Check if it's a dice roll (contains 'd')
		if strings.Contains(part, "d") {
			pieces := strings.Split(part, "d")
			if len(pieces) != 2 {
				return nil, fmt.Errorf("invalid dice notation: %s", part)
			}

			count := 1
			if pieces[0] != "" && pieces[0] != "-" {
				var err error
				count, err = strconv.Atoi(pieces[0])
				if err != nil {
					return nil, fmt.Errorf("invalid dice count: %s", pieces[0])
				}
			} else if pieces[0] == "-" {
				count = -1
			}

			size, err := strconv.Atoi(pieces[1])
			if err != nil {
				return nil, fmt.Errorf("invalid die size: %s", pieces[1])
			}

			if !validDiceSizes[size] {
				return nil, fmt.Errorf("unsupported die size: d%d (valid: d4, d6, d8, d10, d12, d20)", size)
			}

			terms = append(terms, DiceTerm{Count: count, Size: size})
		} else {
			// Flat bonus
			bonus, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid flat bonus: %s", part)
			}
			terms = append(terms, DiceTerm{Count: 0, Size: bonus})
		}
	}

	return &DiceNotation{Terms: terms}, nil
}

// String returns the dice notation as a human-readable string
func (dn *DiceNotation) String() string {
	if len(dn.Terms) == 0 {
		return "0"
	}

	parts := []string{}
	for _, term := range dn.Terms {
		if term.Count == 0 {
			// Flat bonus
			parts = append(parts, fmt.Sprintf("%+d", term.Size))
		} else {
			parts = append(parts, fmt.Sprintf("+%dd%d", term.Count, term.Size))
		}
	}

	return strings.TrimPrefix(strings.Join(parts, ""), "+")
}

// DiceRoller is the subset of the engine's roller the models need
type DiceRoller interface {
	RollDice(diceCount, diceSize, flatBonus int) int
}

// Roll rolls every term and returns the total
func (dn *DiceNotation) Roll(roller DiceRoller) int {
	total := 0
	for _, term := range dn.Terms {
		if term.Count == 0 {
			total += term.Size
		} else {
			total += roller.RollDice(term.Count, term.Size, 0)
		}
	}
	return total
}

// MaxDice is the highest total the dice terms can roll, ignoring flat bonuses.
// SRD critical damage adds this on top of the normal roll.
func (dn *DiceNotation) MaxDice() int {
	total := 0
	for _, term := range dn.Terms {
		if term.Count > 0 {
			total += term.Count * term.Size
		}
	}
	return total
}
