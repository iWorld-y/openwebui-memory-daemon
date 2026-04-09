package task

import (
	"fmt"
	"strings"
)

func DailyPrompt(dateYYYYMMDD string, chatsText string) string {
	return strings.TrimSpace(fmt.Sprintf(`
以下是用户在 %s 的所有对话内容，请提取关键信息生成总结。

硬性要求：
- 使用绝对时间表达（例如“%s 讨论了...”、“4月9日至4月11日期间...”）
- 禁止使用相对时间（例如“昨天”、“最近”、“上周”、“前几天”）
- 输出一段自然语言中文总结，信息密度高，避免冗余

对话内容：
%s
`, dateYYYYMMDD, dateYYYYMMDD, chatsText))
}

func WeeklyPrompt(weekKey string, dailySummariesText string) string {
	return strings.TrimSpace(fmt.Sprintf(`
以下是 %s 的每日对话总结，请压缩合并为一条周总结。

硬性要求：
- 使用绝对时间表达（禁止“昨天/最近/这周”等相对时间）
- 保留关键信息，消除重复与冗余
- 输出一段自然语言中文总结，信息密度高

每日总结：
%s
`, weekKey, dailySummariesText))
}

func MonthlyPrompt(monthKey string, weeklySummariesText string) string {
	return strings.TrimSpace(fmt.Sprintf(`
以下是 %s 的每周总结，请压缩合并为一条月总结。

硬性要求：
- 使用绝对时间表达（禁止“昨天/最近/上个月”等相对时间）
- 保留关键信息，消除重复与冗余
- 输出一段自然语言中文总结，信息密度高

每周总结：
%s
`, monthKey, weeklySummariesText))
}

