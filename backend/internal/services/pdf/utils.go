package pdf

import (
	"encoding/hex"
	"strings"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// =============================================================================
// Color Utilities
// =============================================================================

// hexToRGB converts a hex color string to RGB values
func hexToRGB(hexColor string) (int, int, int) {
	hexColor = strings.TrimPrefix(hexColor, "#")

	if len(hexColor) == 3 {
		// Short hex format (#RGB)
		hexColor = string(hexColor[0]) + string(hexColor[0]) +
			string(hexColor[1]) + string(hexColor[1]) +
			string(hexColor[2]) + string(hexColor[2])
	}

	if len(hexColor) != 6 {
		return 0, 0, 0
	}

	bytes, err := hex.DecodeString(hexColor)
	if err != nil || len(bytes) != 3 {
		return 0, 0, 0
	}

	return int(bytes[0]), int(bytes[1]), int(bytes[2])
}

// =============================================================================
// Currency Formatting
// =============================================================================

// formatCurrency formats a number as currency
func formatCurrency(amount float64) string {
	p := message.NewPrinter(language.English)
	if amount < 0 {
		return p.Sprintf("-$%.2f", -amount)
	}
	return p.Sprintf("$%.2f", amount)
}

// =============================================================================
// Date Formatting
// =============================================================================

// formatDate formats a date in the default format
func formatDate(t time.Time) string {
	return t.Format("January 2, 2006")
}
