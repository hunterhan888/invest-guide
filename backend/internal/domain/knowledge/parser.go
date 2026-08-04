package knowledge

import (
	"regexp"
	"strings"
)

// Parse 把原始文档内容（按 sourceType 不同格式）转为纯文本
// manual/upload 按 markdown/纯文本处理；url 按 HTML 字符串处理
func Parse(content string, sourceType string) (string, error) {
	switch sourceType {
	case SourceManual, SourceUpload:
		return stripMarkdown(content), nil
	case SourceURL:
		return stripHTML(content), nil
	default:
		return content, nil
	}
}

// stripMarkdown 移除 markdown 语法符号，保留纯文本
var (
	mdCodeRe   = regexp.MustCompile("`{1,3}([^`]*)`{1,3}")
	mdLinkRe   = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	mdHeaderRe = regexp.MustCompile(`^#+\s*`)
	mdEmphRe   = regexp.MustCompile(`[*_]+([^*_]+)[*_]+`)
)

func stripMarkdown(s string) string {
	s = mdCodeRe.ReplaceAllString(s, "$1")
	s = mdLinkRe.ReplaceAllString(s, "$1")
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = mdHeaderRe.ReplaceAllString(line, "")
	}
	s = strings.Join(lines, "\n")
	s = mdEmphRe.ReplaceAllString(s, "$1")
	return s
}

// stripHTML 移除 HTML 标签与常见实体
var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

func stripHTML(s string) string {
	s = htmlTagRe.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	return s
}
