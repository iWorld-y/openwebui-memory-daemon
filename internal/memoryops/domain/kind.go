package domain

import "strings"

type Kind int

const (
	KindNone Kind = iota
	KindDaily
	KindWeekly
	KindMonthly
)

func KindFromContent(content string) Kind {
	s := strings.TrimSpace(content)
	switch {
	case strings.HasPrefix(s, "📋 "):
		return KindDaily
	case strings.HasPrefix(s, "📦 "):
		return KindWeekly
	case strings.HasPrefix(s, "📅 "):
		return KindMonthly
	default:
		return KindNone
	}
}
