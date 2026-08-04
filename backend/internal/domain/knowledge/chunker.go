package knowledge

import "strings"

// Chunk 按 chunkSize（字节近似 token）分块，相邻块之间有 overlap 字节重叠
// 优先在段落/换行/句末边界切，避免词中断
func Chunk(text string, chunkSize, overlap int) []string {
	if text == "" || chunkSize <= 0 {
		return nil
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= chunkSize {
		overlap = chunkSize / 2
	}

	paragraphs := splitParagraphs(text)
	var chunks []string
	var cur strings.Builder
	curLen := 0

	flush := func() {
		if curLen > 0 {
			chunks = append(chunks, cur.String())
			cur.Reset()
			curLen = 0
		}
	}

	for _, p := range paragraphs {
		if len(p) > chunkSize {
			flush()
			for _, c := range splitBySize(p, chunkSize, overlap) {
				chunks = append(chunks, c)
			}
			continue
		}
		if curLen > 0 && curLen+len(p)+2 > chunkSize {
			flush()
		}
		if curLen > 0 {
			cur.WriteString("\n\n")
			curLen += 2
		}
		cur.WriteString(p)
		curLen += len(p)
	}
	flush()
	return chunks
}

func splitParagraphs(text string) []string {
	var out []string
	cur := ""
	for _, r := range text {
		if r == '\n' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func splitBySize(s string, chunkSize, overlap int) []string {
	var out []string
	step := chunkSize - overlap
	if step <= 0 {
		step = chunkSize
	}
	for i := 0; i < len(s); i += step {
		end := i + chunkSize
		if end > len(s) {
			end = len(s)
		}
		out = append(out, s[i:end])
		if end == len(s) {
			break
		}
	}
	return out
}
