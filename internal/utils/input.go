package utils

import (
	"strings"
	"unicode"
)

// CleanInputText xóa các ký tự điều khiển không có ý nghĩa nghiệp vụ trong đầu vào terminal, giữ lại văn bản hiển thị cho người dùng.
// Trong trường hợp nhập một dòng, ký tự xuống dòng và tab trong văn bản dán vào sẽ được chuẩn hóa thành khoảng trắng.
func CleanInputText(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return ' '
		}
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

// CleanInputLine làm sạch đầu vào thủ công một dòng và loại bỏ khoảng trắng đầu/cuối.
func CleanInputLine(s string) string {
	return strings.TrimSpace(CleanInputText(s))
}

func CleanInputRunes(runes []rune) string {
	var b strings.Builder
	for _, r := range runes {
		if r == '\n' || r == '\r' || r == '\t' {
			b.WriteByte(' ')
			continue
		}
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// CollapseBlankLines chuẩn hóa line ending (CRLF → LF) rồi rút gọn
// các dòng trống liên tiếp (≥ 2) xuống còn tối đa 1 dòng trống.
// Dùng để chuẩn hóa prompt dán vào hoặc đọc từ file.
func CollapseBlankLines(s string) string {
	// Normalize CRLF → LF trước để tránh \r gây ra visual blank row thừa.
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	blankRun := 0
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			blankRun++
			if blankRun <= 1 {
				out = append(out, "")
			}
		} else {
			blankRun = 0
			out = append(out, l)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

// HasExcessBlankLines báo true khi chuỗi cần normalize:
// có CRLF (paste từ Windows), hoặc ≥ 2 dòng trống liên tiếp.
func HasExcessBlankLines(s string) bool {
	return strings.Contains(s, "\r") || strings.Contains(s, "\n\n\n")
}

func ContainsControl(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}
