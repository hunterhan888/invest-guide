# Invest Guide API 接口文档

Invest Guide 后端 API 接口规范统一以 **OpenAPI 3.0** 形式呈现：

- **规范文件**：[`openapi.yaml`](openapi.yaml)（OpenAPI 3.0.3，可用 Swagger UI / Redoc / Bruno / Postman 等任意 OpenAPI 工具查看或导入）
- **Base URL**：`http://localhost:8080`（开发环境，见服务端 `.env` 的 `PORT`）

## 使用方式

### 用 OpenAPI 工具查看/联调

| 工具 | 用法 |
|------|------|
| Swagger UI | 将 `openapi.yaml` 加载到任意 Swagger UI 实例 |
| Redoc | 在线粘贴或本地渲染 `openapi.yaml` |
| Bruno | 导入 → 选择 OpenAPI 3.x → 选 `openapi.yaml` |
| Postman | 导入 → OpenAPI / Swagger 2.0 / 3.0 |

### 直接浏览（VS Code / 编辑器中）

安装 OpenAPI 预览插件（如 Redocly）后打开 `openapi.yaml` 即可渲染。

## 通用约定（OpenAPI 文件之外的补充说明）

以下约定已体现在 `openapi.yaml` 的组件定义中，这里做速览：

### 统一响应格式

**成功**（`data` 与 `message` 为空时省略）：

```json
{ "success": true, "data": { }, "message": "optional" }
```

分页响应的 `data` 固定为：

```json
{ "items": [ ], "total": 100, "hasMore": true }
```

请求参数：`?page=1&pageSize=20`（页码从 1 开始）。

**错误**：

```json
{ "success": false, "error": "面向人类的错误信息", "code": "ERROR_CODE" }
```

5xx 错误的 `error` 字段固定为 `"internal error"`，不泄露内部细节。

### 鉴权

除 `/auth/*` 与 `/system/*` 外，所有端点需要 JWT：

```
Authorization: Bearer <token>
```

- Token 通过 `POST /api/v1/auth/register` 或 `POST /api/v1/auth/login` 获取
- 算法 HS256，有效期 24h，载荷含 `user_id/exp/iat/iss`

### 错误码映射

| HTTP | code | 用途 |
|------|------|------|
| 400 | `INVALID_INPUT` | 请求格式错误 / 校验失败 |
| 401 | `UNAUTHORIZED` | 缺失或过期的 token |
| 403 | `FORBIDDEN` | 无权限 |
| 404 | `NOT_FOUND` | 资源不存在 |
| 409 | `CONFLICT` | 状态冲突 |
| 429 | `RATE_LIMITED` | 触发限流 |
| 500 | `INTERNAL_ERROR` | 未预期的内部错误 |
| 502 | `BAD_GATEWAY` | LLM / Embedding / 上游服务失败 |
| 504 | `GATEWAY_TIMEOUT` | LLM 响应超时 |

### 限流

| 级别 | 限制 | 时间窗 | 范围 |
|------|------|--------|------|
| 鉴权 | 5 次失败 | 15 分钟 | 登录/注册（按 IP） |
| API | 60 次请求 | 1 分钟 | 通用端点（按用户/IP） |
| 敏感 | 20 次请求 | 1 分钟 | 流式、知识上传（按用户） |

### SSE 流式协议

端点 `GET /api/v1/conversations/{id}/messages/{messageId}/stream`：

- 事件顺序：`heartbeat → sources → message* → done`（错误时 `error` 后不再 `done`）
- 断线重连：客户端携带 `Last-Event-ID` header，服务端从断点续传（仅限最近一条消息流）

## 请求追踪

每个响应都带 `X-Request-ID` header（UUID）。客户端可不传；若已带则服务端沿用。
