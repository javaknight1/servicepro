package pdf

import (
	"encoding/hex"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"

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

// RGBToHex converts RGB values to a hex color string
func RGBToHex(r, g, b int) string {
	return fmt.Sprintf("#%02X%02X%02X", r, g, b)
}

// DarkenColor darkens a hex color by a percentage (0-100)
func DarkenColor(hexColor string, percent float64) string {
	r, g, b := hexToRGB(hexColor)
	factor := 1 - (percent / 100)
	return RGBToHex(
		int(float64(r)*factor),
		int(float64(g)*factor),
		int(float64(b)*factor),
	)
}

// LightenColor lightens a hex color by a percentage (0-100)
func LightenColor(hexColor string, percent float64) string {
	r, g, b := hexToRGB(hexColor)
	factor := percent / 100
	return RGBToHex(
		int(float64(r)+(255-float64(r))*factor),
		int(float64(g)+(255-float64(g))*factor),
		int(float64(b)+(255-float64(b))*factor),
	)
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

// FormatCurrencyWithCode formats a number with currency code
func FormatCurrencyWithCode(amount float64, currencyCode string) string {
	p := message.NewPrinter(language.English)
	symbol := getCurrencySymbol(currencyCode)
	if amount < 0 {
		return p.Sprintf("-%s%.2f", symbol, -amount)
	}
	return p.Sprintf("%s%.2f", symbol, amount)
}

// getCurrencySymbol returns the symbol for a currency code
func getCurrencySymbol(code string) string {
	symbols := map[string]string{
		"USD": "$",
		"EUR": "€",
		"GBP": "£",
		"JPY": "¥",
		"CAD": "CA$",
		"AUD": "A$",
		"CHF": "CHF ",
		"CNY": "¥",
		"INR": "₹",
		"MXN": "MX$",
		"BRL": "R$",
	}
	if symbol, ok := symbols[strings.ToUpper(code)]; ok {
		return symbol
	}
	return code + " "
}

// FormatNumber formats a number with thousands separators
func FormatNumber(n float64) string {
	p := message.NewPrinter(language.English)
	return p.Sprintf("%.2f", n)
}

// FormatInteger formats an integer with thousands separators
func FormatInteger(n int64) string {
	p := message.NewPrinter(language.English)
	return p.Sprintf("%d", n)
}

// =============================================================================
// Date Formatting
// =============================================================================

// formatDate formats a date in the default format
func formatDate(t time.Time) string {
	return t.Format("January 2, 2006")
}

// FormatDateShort formats a date in short format
func FormatDateShort(t time.Time) string {
	return t.Format("Jan 2, 2006")
}

// FormatDateTime formats a date and time
func FormatDateTime(t time.Time) string {
	return t.Format("January 2, 2006 at 3:04 PM")
}

// FormatDateISO formats a date in ISO format
func FormatDateISO(t time.Time) string {
	return t.Format("2006-01-02")
}

// FormatTime formats a time
func FormatTime(t time.Time) string {
	return t.Format("3:04 PM")
}

// FormatTime24 formats a time in 24-hour format
func FormatTime24(t time.Time) string {
	return t.Format("15:04")
}

// FormatDuration formats a duration in human-readable format
func FormatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	if hours > 0 {
		if minutes > 0 {
			return fmt.Sprintf("%dh %dm", hours, minutes)
		}
		return fmt.Sprintf("%d hour(s)", hours)
	}
	return fmt.Sprintf("%d minute(s)", minutes)
}

// =============================================================================
// Text Utilities
// =============================================================================

// TruncateText truncates text to a maximum length with ellipsis
func TruncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	if maxLen <= 3 {
		return text[:maxLen]
	}
	return text[:maxLen-3] + "..."
}

// WrapText wraps text to a maximum line width
func WrapText(text string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	currentLine := words[0]
	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) > maxWidth {
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			currentLine += " " + word
		}
	}
	lines = append(lines, currentLine)

	return lines
}

// SanitizeFilename removes invalid characters from a filename
func SanitizeFilename(filename string) string {
	// Remove or replace invalid characters
	invalid := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)
	safe := invalid.ReplaceAllString(filename, "_")

	// Trim spaces and dots from start and end
	safe = strings.Trim(safe, " .")

	// Limit length
	if len(safe) > 200 {
		safe = safe[:200]
	}

	if safe == "" {
		safe = "document"
	}

	return safe
}

// ToTitleCase converts a string to title case
func ToTitleCase(s string) string {
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			runes := []rune(word)
			runes[0] = unicode.ToUpper(runes[0])
			words[i] = string(runes)
		}
	}
	return strings.Join(words, " ")
}

// StripHTML removes HTML tags from a string
func StripHTML(s string) string {
	re := regexp.MustCompile(`<[^>]*>`)
	return re.ReplaceAllString(s, "")
}

// NormalizeWhitespace replaces multiple whitespace with single space
func NormalizeWhitespace(s string) string {
	re := regexp.MustCompile(`\s+`)
	return strings.TrimSpace(re.ReplaceAllString(s, " "))
}

// =============================================================================
// Unit Conversion
// =============================================================================

// Measurement units
const (
	UnitMM = "mm"
	UnitCM = "cm"
	UnitIN = "in"
	UnitPT = "pt"
)

// MMToInch converts millimeters to inches
func MMToInch(mm float64) float64 {
	return mm / 25.4
}

// InchToMM converts inches to millimeters
func InchToMM(inch float64) float64 {
	return inch * 25.4
}

// MMToPoint converts millimeters to points
func MMToPoint(mm float64) float64 {
	return mm * 2.83465
}

// PointToMM converts points to millimeters
func PointToMM(pt float64) float64 {
	return pt / 2.83465
}

// InchToPoint converts inches to points
func InchToPoint(inch float64) float64 {
	return inch * 72
}

// PointToInch converts points to inches
func PointToInch(pt float64) float64 {
	return pt / 72
}

// ConvertUnit converts a value from one unit to another
func ConvertUnit(value float64, fromUnit, toUnit string) float64 {
	if fromUnit == toUnit {
		return value
	}

	// Convert to points first
	var points float64
	switch fromUnit {
	case UnitMM:
		points = MMToPoint(value)
	case UnitCM:
		points = MMToPoint(value * 10)
	case UnitIN:
		points = InchToPoint(value)
	case UnitPT:
		points = value
	default:
		return value
	}

	// Convert from points to target unit
	switch toUnit {
	case UnitMM:
		return PointToMM(points)
	case UnitCM:
		return PointToMM(points) / 10
	case UnitIN:
		return PointToInch(points)
	case UnitPT:
		return points
	default:
		return value
	}
}

// =============================================================================
// Page Size Utilities
// =============================================================================

// PageSize represents standard page dimensions
type PageSize struct {
	Width  float64
	Height float64
	Unit   string
}

// Standard page sizes in millimeters
var PageSizes = map[string]PageSize{
	"A4":      {Width: 210, Height: 297, Unit: UnitMM},
	"A3":      {Width: 297, Height: 420, Unit: UnitMM},
	"A5":      {Width: 148, Height: 210, Unit: UnitMM},
	"Letter":  {Width: 215.9, Height: 279.4, Unit: UnitMM},
	"Legal":   {Width: 215.9, Height: 355.6, Unit: UnitMM},
	"Tabloid": {Width: 279.4, Height: 431.8, Unit: UnitMM},
}

// GetPageSize returns the dimensions for a page size name
func GetPageSize(name string) (PageSize, bool) {
	size, ok := PageSizes[name]
	return size, ok
}

// GetContentArea returns the printable content area given margins
func GetContentArea(pageSize PageSize, marginLeft, marginRight, marginTop, marginBottom float64) (width, height float64) {
	width = pageSize.Width - marginLeft - marginRight
	height = pageSize.Height - marginTop - marginBottom
	return
}

// =============================================================================
// Number Formatting
// =============================================================================

// FormatPercent formats a decimal as a percentage
func FormatPercent(value float64, decimals int) string {
	format := fmt.Sprintf("%%.%df%%%%", decimals)
	return fmt.Sprintf(format, value*100)
}

// FormatDecimal formats a decimal with specified precision
func FormatDecimal(value float64, decimals int) string {
	format := fmt.Sprintf("%%.%df", decimals)
	return fmt.Sprintf(format, value)
}

// RoundTo rounds a number to the specified decimal places
func RoundTo(value float64, decimals int) float64 {
	multiplier := math.Pow(10, float64(decimals))
	return math.Round(value*multiplier) / multiplier
}

// =============================================================================
// Address Formatting
// =============================================================================

// FormatAddress formats address components into a single string
func FormatAddress(street, city, state, zip, country string) string {
	var parts []string

	if street != "" {
		parts = append(parts, street)
	}

	cityStateZip := ""
	if city != "" {
		cityStateZip = city
	}
	if state != "" {
		if cityStateZip != "" {
			cityStateZip += ", "
		}
		cityStateZip += state
	}
	if zip != "" {
		if cityStateZip != "" {
			cityStateZip += " "
		}
		cityStateZip += zip
	}
	if cityStateZip != "" {
		parts = append(parts, cityStateZip)
	}

	if country != "" && country != "USA" && country != "US" && country != "United States" {
		parts = append(parts, country)
	}

	return strings.Join(parts, "\n")
}

// =============================================================================
// Phone Formatting
// =============================================================================

// FormatPhone formats a phone number
func FormatPhone(phone string) string {
	// Remove all non-numeric characters
	digits := regexp.MustCompile(`\D`).ReplaceAllString(phone, "")

	switch len(digits) {
	case 10:
		return fmt.Sprintf("(%s) %s-%s", digits[0:3], digits[3:6], digits[6:10])
	case 11:
		if digits[0] == '1' {
			return fmt.Sprintf("+1 (%s) %s-%s", digits[1:4], digits[4:7], digits[7:11])
		}
	}

	return phone
}

// =============================================================================
// Validation Utilities
// =============================================================================

// IsValidEmail checks if an email address is valid
func IsValidEmail(email string) bool {
	re := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return re.MatchString(email)
}

// IsValidPhone checks if a phone number is valid
func IsValidPhone(phone string) bool {
	digits := regexp.MustCompile(`\D`).ReplaceAllString(phone, "")
	return len(digits) >= 10 && len(digits) <= 15
}

// IsValidHexColor checks if a string is a valid hex color
func IsValidHexColor(color string) bool {
	re := regexp.MustCompile(`^#?([0-9A-Fa-f]{3}|[0-9A-Fa-f]{6})$`)
	return re.MatchString(color)
}

// =============================================================================
// Barcode/QR Code Utilities
// =============================================================================

// GenerateInvoiceReference generates a unique invoice reference
func GenerateInvoiceReference(invoiceNumber string, date time.Time) string {
	return fmt.Sprintf("%s-%s", invoiceNumber, date.Format("20060102"))
}

// ParseInvoiceReference parses an invoice reference
func ParseInvoiceReference(reference string) (invoiceNumber string, date time.Time, err error) {
	parts := strings.Split(reference, "-")
	if len(parts) < 2 {
		return "", time.Time{}, fmt.Errorf("invalid reference format")
	}

	invoiceNumber = parts[0]
	dateStr := parts[len(parts)-1]

	date, err = time.Parse("20060102", dateStr)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("invalid date in reference: %w", err)
	}

	return invoiceNumber, date, nil
}

// =============================================================================
// Calculation Utilities
// =============================================================================

// CalculateLineTotal calculates the total for a line item
func CalculateLineTotal(quantity, unitPrice, discount, taxRate float64) float64 {
	subtotal := quantity * unitPrice
	discountAmount := subtotal * (discount / 100)
	afterDiscount := subtotal - discountAmount
	taxAmount := afterDiscount * (taxRate / 100)
	return afterDiscount + taxAmount
}

// CalculateSubtotal calculates subtotal from line items
func CalculateSubtotal(lineAmounts []float64) float64 {
	var total float64
	for _, amount := range lineAmounts {
		total += amount
	}
	return total
}

// CalculateTax calculates tax amount
func CalculateTax(subtotal, taxRate float64) float64 {
	return subtotal * (taxRate / 100)
}

// CalculateDiscount calculates discount amount
func CalculateDiscount(subtotal, discountRate float64) float64 {
	return subtotal * (discountRate / 100)
}

// =============================================================================
// File Utilities
// =============================================================================

// GetFileExtension returns the extension from a filename
func GetFileExtension(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) > 1 {
		return strings.ToLower(parts[len(parts)-1])
	}
	return ""
}

// IsImageFile checks if a filename is an image
func IsImageFile(filename string) bool {
	ext := GetFileExtension(filename)
	imageExts := map[string]bool{
		"jpg": true, "jpeg": true, "png": true,
		"gif": true, "bmp": true, "tiff": true,
	}
	return imageExts[ext]
}

// FormatFileSize formats a file size in bytes to human-readable format
func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// =============================================================================
// String Parsing Utilities
// =============================================================================

// ParseFloat safely parses a string to float64
func ParseFloat(s string) float64 {
	// Remove currency symbols and commas
	s = regexp.MustCompile(`[$,€£¥]`).ReplaceAllString(s, "")
	s = strings.TrimSpace(s)

	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// ParseInt safely parses a string to int64
func ParseInt(s string) int64 {
	// Remove commas
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)

	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}
