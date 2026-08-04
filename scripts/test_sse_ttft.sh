#!/usr/bin/env bash
# Invest Guide SSE 流式回答首字延迟测试
#
# 用法：
#   bash scripts/test_sse_ttft.sh                # 用 smoke@test.com 测试
#   bash scripts/test_sse_ttft.sh <email> <密码> # 指定账号
#
# 前置：后端服务已在运行（make backend-dev），知识库已有数据。
# 输出：每个关键事件的时间戳 + 首字延迟。

set -euo pipefail

BASE_URL="http://localhost:8080"
EMAIL="${1:-smoke@test.com}"
if [ -z "${2:-}" ]; then
  echo "用法: bash scripts/test_sse_ttft.sh <email> <密码> [问题]"
  echo "示例: bash scripts/test_sse_ttft.sh user@example.com 'mypassword'"
  exit 1
fi
PASSWORD="$2"
QUESTION="${3:-越南的企业所得税率是多少？}"

echo "=== 1. 登录获取 token ==="
LOGIN_RESP=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"email\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
TOKEN=$(echo "$LOGIN_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['token'])" 2>/dev/null || echo "")
if [ -z "$TOKEN" ]; then
  echo "❌ 登录失败: $LOGIN_RESP"
  exit 1
fi
echo "✅ token 获取成功 (${#TOKEN} 字符)"

echo "=== 2. 创建会话 ==="
CONV=$(curl -s -X POST "$BASE_URL/api/v1/conversations" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"title\":\"SSE-TTFT-测试\",\"country\":\"越南\"}" \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['id'])" 2>/dev/null || echo "")
[ -n "$CONV" ] || { echo "❌ 创建会话失败"; exit 1; }
echo "✅ 会话: $CONV"

echo "=== 3. 发送问题 ==="
MSG=$(curl -s -X POST "$BASE_URL/api/v1/conversations/$CONV/messages" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"content\":\"$QUESTION\"}" \
  | python3 -c "import sys,json; print(json.load(sys.stdin)['data']['messageId'])" 2>/dev/null || echo "")
[ -n "$MSG" ] || { echo "❌ 发送消息失败"; exit 1; }
echo "✅ 消息: $MSG"

echo "=== 4. 订阅 SSE 并计时（首字延迟）==="
echo "开始时间: $(date +%H:%M:%S.%3N)"
echo "---- SSE 事件流 ----"

# 记录请求发出的时间戳（纳秒）
T0=$(date +%s%N)

# 用 awk 解析 SSE 帧，实时计算时间
curl -s -N --max-time 60 \
  "$BASE_URL/api/v1/conversations/$CONV/messages/$MSG/stream" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Accept: text/event-stream" \
  2>/dev/null | python3 -u -c "
import sys, time

t0 = int('$T0') / 1e9  # 请求发出时刻
first_delta_at = None

def now():
    return time.time()

for line in sys.stdin:
    line = line.rstrip('\n')
    if line.startswith('event: '):
        evt = line[7:]
        # 预读下一行 data
        try:
            data = next(sys.stdin).rstrip('\n')
        except StopIteration:
            data = ''
        ts = now()
        elapsed = ts - t0
        if evt == 'message' and first_delta_at is None:
            first_delta_at = ts
        if evt == 'sources':
            print(f'[{elapsed:6.2f}s] event: sources ({len(data)} 字节)')
        elif evt == 'message':
            print(f'[{elapsed:6.2f}s] event: message ({len(data)} 字节)')
        elif evt == 'done':
            print(f'[{elapsed:6.2f}s] event: done')
        elif evt == 'error':
            print(f'[{elapsed:6.2f}s] event: error')
        elif evt == 'heartbeat':
            print(f'[{elapsed:6.2f}s] event: heartbeat')
    if first_delta_at is not None and line.startswith('event: done'):
        break

print()
if first_delta_at:
    print(f'✅ 首字延迟 (TTFT): {first_delta_at - t0:.2f} 秒')
    print(f'✅ 总耗时: {time.time() - t0:.2f} 秒')
else:
    print('⚠️ 未收到 message 事件（可能 LLM 调用失败或超时）')
"
