package web

import (
	"strings"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// slugify chuyển tên hiển thị thành slug an toàn cho filesystem.
func slugify(title string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	s, _, _ := transform.String(t, strings.ToLower(strings.TrimSpace(title)))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// uniqueSlug trả về slug chưa có trong used.
func uniqueSlug(title string, used map[string]bool) string {
	base := slugify(title)
	if base == "" {
		base = "novel"
	}
	candidate := base
	for n := 2; used[candidate]; n++ {
		candidate = base + "-" + itoa(n)
	}
	used[candidate] = true
	return candidate
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	for n > 0 {
		i--
		digits[i] = byte('0' + n%10)
		n /= 10
	}
	return string(digits[i:])
}