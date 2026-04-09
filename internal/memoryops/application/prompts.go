package application

import (
	"bytes"
	"strings"
	"text/template"
)

const PROMPT_DAILY = `你是一个对话记忆助手。以下是用户在 {{.date}} 的所有对话记录。

请从中提取值得长期记住的信息，包括但不限于：
- 做出的决定或结论
- 学到的新知识或新认知
- 提到的项目、技术、工具及其关键细节
- 表达的观点、偏好或倾向
- 待办事项或未来计划

严格规则：
1. 必须使用绝对时间（如"2026-04-09"），禁止使用"昨天""今天"" recently"等相对时间
2. 不改写用户表述的原意，不添加推断
3. 如果当天没有值得记录的内容，输出"当日无关键信息"
4. 按主题分段，每段以"· "开头

对话记录：
---
{{.conversations}}
---`

const PROMPT_WEEKLY = `你是一个记忆压缩助手。以下是用户在 {{.week_start}} 至 {{.week_end}} 的每日对话总结。

请将它们压缩合并为一条周总结：
- 合并同一主题在不同日期的讨论，保留讨论的演进脉络
- 消除跨天重复的信息
- 已被后续对话推翻或修正的早期结论，以最终结论为准
- 保留具体的事实、数据、名称，不要过度泛化

严格规则：
1. 必须使用绝对时间（如"2026-04-09"），禁止使用相对时间
2. 保留发生日期的标注，如"4月9日讨论了...4月11日调整方案为..."
3. 每个主题段落保留足够的细节级别，不丢失关键决策和事实
4. 按主题分段，每段以"· "开头

每日总结：
---
{{.daily_summaries}}
---`

const PROMPT_MONTHLY = `你是一个记忆归档助手。以下是用户在 {{.year}}-{{.month}} 的每周对话总结。

请将它们提炼为一条月度归档：
- 识别本月核心主题和重大事项
- 对于持续多周的话题，概括其整体走向和最终状态
- 一次性讨论且无后续的话题，如无重要结论可省略
- 保留所有具体的事实、数据、名称，不要过度泛化

严格规则：
1. 必须使用绝对时间（如"2026-04-09"），禁止使用相对时间
2. 重大决定和结论必须保留原文或接近原文的表述
3. 按主题分段，每段以"· "开头
4. 输出应比输入有显著压缩，但关键信息零丢失

每周总结：
---
{{.weekly_summaries}}
---`

func renderPrompt(tpl string, data map[string]string) string {
	t, err := template.New("prompt").Option("missingkey=error").Parse(tpl)
	if err != nil {
		return tpl
	}
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		return tpl
	}
	return b.String()
}

func DailyPrompt(dateYYYYMMDD string, chatsText string) string {
	return strings.TrimSpace(renderPrompt(PROMPT_DAILY, map[string]string{
		"date":          dateYYYYMMDD,
		"conversations": strings.TrimSpace(chatsText),
	}))
}

func WeeklyPrompt(weekStartYYYYMMDD string, weekEndYYYYMMDD string, dailySummariesText string) string {
	return strings.TrimSpace(renderPrompt(PROMPT_WEEKLY, map[string]string{
		"week_start":     weekStartYYYYMMDD,
		"week_end":       weekEndYYYYMMDD,
		"daily_summaries": strings.TrimSpace(dailySummariesText),
	}))
}

func MonthlyPrompt(monthKey string, weeklySummariesText string) string {
	year := monthKey
	month := ""
	if parts := strings.SplitN(monthKey, "-", 2); len(parts) == 2 {
		year, month = parts[0], parts[1]
	}
	return strings.TrimSpace(renderPrompt(PROMPT_MONTHLY, map[string]string{
		"year":            year,
		"month":           month,
		"weekly_summaries": strings.TrimSpace(weeklySummariesText),
	}))
}
