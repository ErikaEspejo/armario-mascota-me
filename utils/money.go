package utils

import (
	"strconv"
	"strings"
)

// FormatCOP formats an integer amount (in COP) as a string like "$12.500".
// Uses dot as thousands separator (common in Colombia).
func FormatCOP(amount int64) string {
	neg := amount < 0
	if neg {
		amount = -amount
	}

	s := strconv.FormatInt(amount, 10)
	if len(s) <= 3 {
		if neg {
			return "-$" + s
		}
		return "$" + s
	}

	var b strings.Builder
	// Pre-allocate: digits + separators + $
	b.Grow(len(s) + len(s)/3 + 2)
	if neg {
		b.WriteString("-$")
	} else {
		b.WriteString("$")
	}

	// Insert separators from the left.
	rem := len(s) % 3
	if rem == 0 {
		rem = 3
	}
	b.WriteString(s[:rem])
	for i := rem; i < len(s); i += 3 {
		b.WriteByte('.')
		b.WriteString(s[i : i+3])
	}

	return b.String()
}
