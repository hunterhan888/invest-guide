package assistant

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildSystemPrompt_WithSources(t *testing.T) {
	sources := []ContextSource{
		{Title: "越南指南", Snippet: "河内是首都"},
		{Title: "泰国指南", Snippet: "曼谷是首都"},
	}
	p := BuildSystemPrompt(sources)
	assert.Contains(t, p, "越南指南")
	assert.Contains(t, p, "河内是首都")
	assert.Contains(t, p, "泰国指南")
	assert.Contains(t, p, "[1]")
	assert.Contains(t, p, "[2]")
}

func TestBuildSystemPrompt_EmptySources(t *testing.T) {
	p := BuildSystemPrompt(nil)
	assert.True(t, strings.Contains(p, "无可用知识库片段"))
	assert.NotContains(t, p, "[1]")
}
