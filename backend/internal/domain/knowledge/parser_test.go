package knowledge

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParse_Markdown(t *testing.T) {
	content := "# 标题\n\n**加粗** 文本 [链接](http://x.com) `代码`"
	got, err := Parse(content, SourceManual)
	assert.NoError(t, err)
	assert.NotContains(t, got, "#")
	assert.NotContains(t, got, "**")
	assert.NotContains(t, got, "[链接](http://x.com)")
	assert.Contains(t, got, "加粗")
	assert.Contains(t, got, "链接")
	assert.Contains(t, got, "代码")
}

func TestParse_HTML(t *testing.T) {
	content := "<p>越南简介</p><div>工业园区</div>&nbsp;&amp;&lt;x&gt;"
	got, err := Parse(content, SourceURL)
	assert.NoError(t, err)
	assert.NotContains(t, got, "<p>")
	assert.NotContains(t, got, "&nbsp;")
	assert.Contains(t, got, "越南简介")
	assert.Contains(t, got, "工业园区")
	assert.Contains(t, got, "&")
	assert.Contains(t, got, "<x>")
}

func TestParse_UnknownSource(t *testing.T) {
	got, err := Parse("raw text", "unknown")
	assert.NoError(t, err)
	assert.Equal(t, "raw text", got)
}
