package task

import "unicode/utf8"

// TruncateKeepHeadTail keeps head and tail to fit within maxBytes.
// It works in bytes but preserves UTF-8 integrity.
func TruncateKeepHeadTail(s string, maxBytes int) string {
	if maxBytes <= 0 {
		return ""
	}
	if len(s) <= maxBytes {
		return s
	}
	if maxBytes < 16 {
		return string([]rune(s)[:1])
	}

	keep := (maxBytes - 10) / 2
	head := s[:safeUTF8Cut(s, keep)]
	tailStart := len(s) - keep
	if tailStart < 0 {
		tailStart = 0
	}
	tail := s[tailStart:]
	tail = tail[len(tail)-safeUTF8CutReverse(tail, keep):]
	return head + "\n...\n" + tail
}

func safeUTF8Cut(s string, max int) int {
	if max >= len(s) {
		return len(s)
	}
	i := max
	for i > 0 && !utf8.ValidString(s[:i]) {
		i--
	}
	if i == 0 {
		return 0
	}
	return i
}

func safeUTF8CutReverse(s string, max int) int {
	if max >= len(s) {
		return len(s)
	}
	i := max
	for i > 0 && !utf8.ValidString(s[len(s)-i:]) {
		i--
	}
	return i
}

