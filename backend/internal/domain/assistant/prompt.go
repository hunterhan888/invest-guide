package assistant

import (
	"fmt"
	"strings"
)

const systemPromptTemplate = `你是国别投资指南助手。基于以下知识库片段回答用户问题。
若上下文不足以回答，请明确说明"知识库中未涵盖该问题"，不要编造内容。
回答采用中文，结构清晰。

# 知识库片段
%s

# 指引
- 引用片段时使用「片段[N]」格式标记编号（N 为片段编号），便于前端跳转到来源
- 用户问投资相关的法律、税务、行业准入、园区、外汇等内容时优先引用片段
- 不讨论与投资无关的话题`

// BuildSystemPrompt 把检索到的 sources 拼成 system prompt
func BuildSystemPrompt(sources []ContextSource) string {
	if len(sources) == 0 {
		return strings.Replace(systemPromptTemplate, "%s", "（无可用知识库片段，请基于通用知识谨慎回答）", 1)
	}
	var b strings.Builder
	for i, s := range sources {
		fmt.Fprintf(&b, "[%d] %s\n%s\n\n", i+1, s.Title, s.Snippet)
	}
	return fmt.Sprintf(systemPromptTemplate, b.String())
}
