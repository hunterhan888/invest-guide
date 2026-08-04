#!/usr/bin/env bash
# SiliconFlow 原生流式首字延迟测试（绕过 Invest Guide 后端，直接打上游）
#
# 目的：区分 SSE 首字延迟是「上游 SiliconFlow 慢」还是「我们后端消费逻辑慢」。
#
# 用法：
#   bash scripts/test_llm_stream_ttft.sh
#
# 前置：.env 中已配置 LLM_API_KEY / LLM_MODEL。

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
API_KEY="$(grep '^LLM_API_KEY=' "$ROOT/.env" | cut -d= -f2-)"
MODEL="$(grep '^LLM_MODEL=' "$ROOT/.env" | cut -d= -f2-)"
BASE_URL="https://api.siliconflow.cn/v1/chat/completions"
QUESTION="${1:-越南的企业所得税率是多少？}"

if [ -z "$API_KEY" ] || [ "$API_KEY" = "sk-..." ]; then
  echo "❌ .env 未配置有效的 LLM_API_KEY"
  exit 1
fi

echo "模型: $MODEL"
echo "问题: $QUESTION"
echo "请求发出: $(date +%H:%M:%S.%3N)"

T0=$(date +%s%N)

# 用 Python 实时解析 SSE 流，计算首个 data 的时间
curl -s -N --max-time 120 \
  -X POST "$BASE_URL" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d "{\"model\":\"$MODEL\",\"stream\":true,\"messages\":[{\"role\":\"user\",\"content\":\"$QUESTION\"}]}" \
  2>/dev/null | python3 -u -c "
import sys, time

t0 = int('$T0') / 1e9
first_data_at = None
chunk_count = 0
total_chars = 0

def now():
    return time.time()

for line in sys.stdin:
    line = line.rstrip('\n')
    if not line.startswith('data: '):
        continue
    data = line[6:]
    ts = now()
    elapsed = ts - t0
    if first_data_at is None:
        first_data_at = ts
    chunk_count += 1
    if '[DONE]' in data:
        print(f'[{elapsed:6.2f}s] [DONE]')
        break
    # 提取 content delta（粗略）
    import json
    try:
        obj = json.loads(data)
        for choice in obj.get('choices', []):
            delta = choice.get('delta', {}).get('content', '')
            if delta:
                total_chars += len(delta)
                if chunk_count <= 3:
                    print(f'[{elapsed:6.2f}s] delta: {delta!r}')
    except Exception:
        pass

print()
if first_data_at:
    print(f'✅ SiliconFlow 原生流式首字延迟 (TTFT): {first_data_at - t0:.2f} 秒')
    print(f'   chunks: {chunk_count} 个, 共 {total_chars} 字符')
    print(f'   （对比：Invest Guide 后端实测 27.93 秒）')
else:
    print('⚠️ 未收到任何流式数据（请求失败或超时）')
"
