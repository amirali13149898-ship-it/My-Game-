package game

import (
	"strconv"
	"strings"
	"unicode"
)

var faDigits = []rune("۰۱۲۳۴۵۶۷۸۹")

// FaS converts ASCII digits in s to Persian digits.
func FaS(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(faDigits[r-'0'])
		} else if r == '.' {
			b.WriteRune('٫')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Fa formats an integer with Persian digits and thousands separators.
func Fa(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	var out []byte
	for i := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, s[i])
	}
	res := string(out)
	res = strings.ReplaceAll(res, ",", "٬")
	res = FaS(res)
	if neg {
		return "−" + res
	}
	return res
}

// FaI is Fa for int.
func FaI(n int) string { return Fa(int64(n)) }

// FaF formats a float with at most dec decimals (trailing zeros trimmed).
func FaF(f float64, dec int) string {
	s := strconv.FormatFloat(f, 'f', dec, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	return FaS(s)
}

// Norm normalizes user text so that ability names match regardless of
// Arabic/Persian letter variants, spaces, ZWNJ and diacritics.
func Norm(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(s) {
		switch {
		case r == 'ي' || r == 'ى':
			b.WriteRune('ی')
		case r == 'ك':
			b.WriteRune('ک')
		case r == 'ۀ' || r == 'ة':
			b.WriteRune('ه')
		case r == '‌' || r == '‍' || r == '‎' || r == '‏' || r == 'ـ':
			// skip
		case r >= 'ً' && r <= 'ٟ':
			// skip diacritics
		case unicode.IsSpace(r):
			// skip
		case r >= '٠' && r <= '٩':
			b.WriteRune(faDigits[r-'٠'])
		default:
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}
