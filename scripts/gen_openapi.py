#!/usr/bin/env python3
"""Generate OpenAPI 3.0 spec for InvestGuide API (backend)."""
from pathlib import Path
import yaml

S = "application/json"
BEARER = [{"bearerAuth": []}]

def jprop(**kw):
    return kw

def page(item_ref):
    return {"type": "object", "properties": {
        "items": {"type": "array", "items": {"$ref": item_ref}},
        "total": {"type": "integer"},
        "hasMore": {"type": "boolean"},
    }}

def ok(data_schema, extra_props=None):
    props = {"success": {"type": "boolean", "example": True}}
    if data_schema is not None:
        props["data"] = data_schema
    if extra_props:
        props.update(extra_props)
    return {"type": "object", "properties": props}

def res(schema):
    return {"description": "OK", "content": {S: {"schema": schema}}}

def err_res(name, desc):
    return {"description": desc, "content": {S: {"schema": {"$ref": "#/components/schemas/Error"}}}}

doc = {
    "openapi": "3.0.3",
    "info": {
        "title": "InvestGuide API",
        "description": (
            "InvestGuide 是一个面向国别投资指南的 Web 端 AI 问答平台。用户通过自然语言提问，"
            "了解各国投资相关内容，回答基于精选知识库通过检索增强生成（RAG）得到。\n\n"
            "鉴权：除 /auth/* 与 /system/* 外，所有端点需要 JWT Bearer Token（有效期 24h）。\n"
            "所有响应（除 SSE 流式端点）统一封装为 { success, data?, message? } 或 "
            "{ success, error, code }。5xx 错误的 error 字段固定为 internal error。"
        ),
        "version": "0.0.1",
    },
    "servers": [{"url": "http://localhost:8080", "description": "本地开发"}],
    "tags": [
        {"name": "System", "description": "系统信息（健康检查、版本、模型列表）— 无需鉴权"},
        {"name": "Auth", "description": "用户注册、登录 — 无需鉴权"},
        {"name": "Knowledge", "description": "国别投资指南知识文档管理 — 需要鉴权"},
        {"name": "Conversations", "description": "问答会话与消息 — 需要鉴权"},
    ],
    "paths": {
        "/api/v1/system/health": {
            "get": {
                "tags": ["System"], "summary": "健康检查", "operationId": "getHealth",
                "responses": {"200": res(ok({"type": "object", "properties": {"status": {"type": "string", "example": "ok"}}}))},
            }
        },
        "/api/v1/system/version": {
            "get": {
                "tags": ["System"], "summary": "版本信息", "operationId": "getVersion",
                "responses": {"200": res(ok({"type": "object", "properties": {
                    "version": {"type": "string", "example": "0.0.1-dev"},
                    "goVersion": {"type": "string", "example": "go1.26.5"},
                }}))},
            }
        },
        "/api/v1/system/models": {
            "get": {
                "tags": ["System"], "summary": "可用模型列表",
                "description": "返回当前配置使用的 LLM 与 Embedding 模型",
                "operationId": "getModels",
                "responses": {"200": res(ok({"type": "object", "properties": {
                    "llm": {"type": "string", "example": "Qwen/Qwen3.5-27B"},
                    "embedding": {"type": "string", "example": "Qwen/Qwen3-Embedding-0.6B"},
                }}))},
            }
        },
        "/api/v1/auth/register": {
            "post": {
                "tags": ["Auth"], "summary": "用户注册",
                "description": "创建新用户并签发 JWT。邮箱冲突返回 409。",
                "operationId": "register",
                "requestBody": {"required": True, "content": {S: {"schema": {"$ref": "#/components/schemas/RegisterRequest"}}}},
                "responses": {
                    "201": res(ok({"$ref": "#/components/schemas/AuthResponse"})),
                    "400": err_res("BadRequest", "请求格式错误 / 校验失败"),
                    "409": err_res("Conflict", "邮箱已注册"),
                },
            }
        },
        "/api/v1/auth/login": {
            "post": {
                "tags": ["Auth"], "summary": "用户登录",
                "description": "校验邮箱密码并签发 JWT。邮箱不存在与密码错误统一返回 401。",
                "operationId": "login",
                "requestBody": {"required": True, "content": {S: {"schema": {"$ref": "#/components/schemas/LoginRequest"}}}},
                "responses": {
                    "200": res(ok({"$ref": "#/components/schemas/AuthResponse"})),
                    "400": err_res("BadRequest", "请求格式错误 / 校验失败"),
                    "401": err_res("Unauthorized", "邮箱不存在或密码错误"),
                },
            }
        },
        "/api/v1/knowledge-docs": {
            "get": {
                "tags": ["Knowledge"], "summary": "知识文档列表",
                "description": "返回当前用户的文档列表，按创建时间倒序、分页。",
                "operationId": "listKnowledgeDocs", "security": BEARER,
                "parameters": [
                    {"name": "page", "in": "query", "schema": {"type": "integer", "default": 1}},
                    {"name": "pageSize", "in": "query", "schema": {"type": "integer", "default": 20}},
                    {"name": "country", "in": "query", "description": "按国别过滤（精确匹配）", "schema": {"type": "string"}},
                ],
                "responses": {
                    "200": res(ok(page("#/components/schemas/KnowledgeDoc"))),
                    "401": err_res("Unauthorized", "未鉴权"),
                },
            },
            "post": {
                "tags": ["Knowledge"], "summary": "上传知识文档",
                "description": (
                    "创建文档并异步入库（解析 → 分块 → 向量化）。返回 202 + 文档 ID，"
                    "客户端通过 GET /api/v1/knowledge-docs/{id} 轮询 status。"
                    "原始 PDF 需先经外部脚本预处理为文本后再上传。"
                ),
                "operationId": "createKnowledgeDoc", "security": BEARER,
                "requestBody": {"required": True, "content": {S: {"schema": {"$ref": "#/components/schemas/CreateDocRequest"}}}},
                "responses": {
                    "202": res(ok({"$ref": "#/components/schemas/KnowledgeDoc"})),
                    "400": err_res("BadRequest", "请求格式错误 / 校验失败"),
                    "401": err_res("Unauthorized", "未鉴权"),
                    "429": err_res("RateLimited", "触发限流"),
                },
            },
        },
        "/api/v1/knowledge-docs/{id}": {
            "get": {
                "tags": ["Knowledge"], "summary": "知识文档详情/状态",
                "description": "获取单个文档，主要用于轮询入库状态（pending/processing/ready/failed）。",
                "operationId": "getKnowledgeDoc", "security": BEARER,
                "parameters": [{"name": "id", "in": "path", "required": True, "schema": {"type": "string"}}],
                "responses": {
                    "200": res(ok({"$ref": "#/components/schemas/KnowledgeDoc"})),
                    "401": err_res("Unauthorized", "未鉴权"),
                    "404": err_res("NotFound", "文档不存在"),
                },
            },
            "delete": {
                "tags": ["Knowledge"], "summary": "删除知识文档",
                "description": "删除文档及其全部分块（级联删除）。",
                "operationId": "deleteKnowledgeDoc", "security": BEARER,
                "parameters": [{"name": "id", "in": "path", "required": True, "schema": {"type": "string"}}],
                "responses": {
                    "204": {"description": "删除成功，无内容"},
                    "401": err_res("Unauthorized", "未鉴权"),
                    "404": err_res("NotFound", "文档不存在"),
                },
            },
        },
        "/api/v1/knowledge-docs/{id}/retry": {
            "post": {
                "tags": ["Knowledge"], "summary": "重试文档入库",
                "description": "把 status 从 failed 改回 pending 并重新入队执行入库流水线。仅 failed 状态可重试。",
                "operationId": "retryKnowledgeDoc", "security": BEARER,
                "parameters": [{"name": "id", "in": "path", "required": True, "schema": {"type": "string"}}],
                "responses": {
                    "202": res(ok(None)),
                    "401": err_res("Unauthorized", "未鉴权"),
                    "404": err_res("NotFound", "文档不存在"),
                    "409": err_res("Conflict", "非 failed 状态不可重试"),
                },
            }
        },
        "/api/v1/conversations": {
            "get": {
                "tags": ["Conversations"], "summary": "会话列表",
                "description": "返回当前用户的会话列表，按最后活跃时间倒序、分页。",
                "operationId": "listConversations", "security": BEARER,
                "parameters": [
                    {"name": "page", "in": "query", "schema": {"type": "integer", "default": 1}},
                    {"name": "pageSize", "in": "query", "schema": {"type": "integer", "default": 20}},
                ],
                "responses": {
                    "200": res(ok(page("#/components/schemas/Conversation"))),
                    "401": err_res("Unauthorized", "未鉴权"),
                },
            },
            "post": {
                "tags": ["Conversations"], "summary": "新建会话",
                "description": "创建新会话。title 留空则默认为 新会话。",
                "operationId": "createConversation", "security": BEARER,
                "requestBody": {"required": True, "content": {S: {"schema": {"$ref": "#/components/schemas/CreateConversationRequest"}}}},
                "responses": {
                    "201": res(ok({"$ref": "#/components/schemas/Conversation"})),
                    "400": err_res("BadRequest", "字段长度超出限制"),
                    "401": err_res("Unauthorized", "未鉴权"),
                },
            },
        },
        "/api/v1/conversations/{id}": {
            "get": {
                "tags": ["Conversations"], "summary": "会话详情",
                "operationId": "getConversation", "security": BEARER,
                "parameters": [{"name": "id", "in": "path", "required": True, "schema": {"type": "string"}}],
                "responses": {
                    "200": res(ok({"$ref": "#/components/schemas/Conversation"})),
                    "401": err_res("Unauthorized", "未鉴权"),
                    "404": err_res("NotFound", "会话不存在"),
                },
            },
            "delete": {
                "tags": ["Conversations"], "summary": "删除会话",
                "description": "删除会话及其全部消息（级联删除）。",
                "operationId": "deleteConversation", "security": BEARER,
                "parameters": [{"name": "id", "in": "path", "required": True, "schema": {"type": "string"}}],
                "responses": {
                    "204": {"description": "删除成功，无内容"},
                    "401": err_res("Unauthorized", "未鉴权"),
                    "404": err_res("NotFound", "会话不存在"),
                },
            },
        },
        "/api/v1/conversations/{id}/messages": {
            "get": {
                "tags": ["Conversations"], "summary": "消息历史",
                "description": "返回会话的消息历史，按时间正序、分页。",
                "operationId": "listMessages", "security": BEARER,
                "parameters": [
                    {"name": "id", "in": "path", "required": True, "schema": {"type": "string"}},
                    {"name": "page", "in": "query", "schema": {"type": "integer", "default": 1}},
                    {"name": "pageSize", "in": "query", "schema": {"type": "integer", "default": 50}},
                ],
                "responses": {
                    "200": res(ok(page("#/components/schemas/Message"))),
                    "401": err_res("Unauthorized", "未鉴权"),
                    "404": err_res("NotFound", "会话不存在"),
                },
            },
            "post": {
                "tags": ["Conversations"], "summary": "发送提问",
                "description": "创建一条 user 消息与一条占位 assistant 消息，返回 assistant messageId。回答内容需通过 SSE 流式端点订阅获取。",
                "operationId": "postMessage", "security": BEARER,
                "parameters": [{"name": "id", "in": "path", "required": True, "schema": {"type": "string"}}],
                "requestBody": {"required": True, "content": {S: {"schema": {"$ref": "#/components/schemas/PostMessageRequest"}}}},
                "responses": {
                    "201": res(ok({"type": "object", "properties": {"messageId": {"type": "string", "example": "msg-assistant-1"}}})),
                    "400": err_res("BadRequest", "content 缺失/空/超长"),
                    "401": err_res("Unauthorized", "未鉴权"),
                    "404": err_res("NotFound", "会话不存在"),
                    "429": err_res("RateLimited", "触发限流"),
                },
            },
        },
        "/api/v1/conversations/{id}/messages/{messageId}/stream": {
            "get": {
                "tags": ["Conversations"], "summary": "流式回答 (SSE)",
                "description": (
                    "订阅指定 assistant 消息的 SSE 流式回答。\n\n"
                    "事件顺序（正常流程）：heartbeat → sources → message* → done\n"
                    "错误流程：error（出现后连接关闭，不再发送 done）\n\n"
                    "事件类型：\n"
                    "- heartbeat: 每 15 秒心跳 (data: {})\n"
                    "- sources: 检索引用的知识块（在首次 message 之前发送）\n"
                    "- message: 增量文本 (data: {\"delta\": \"文本片段\"})\n"
                    "- done: 流结束 (data: {\"messageId\": \"...\", \"tokensUsed\": 350})\n"
                    "- error: 终止性失败 (data: {\"code\": \"LLM_ERROR\", \"message\": \"...\"})"
                ),
                "operationId": "streamAnswer", "security": BEARER,
                "parameters": [
                    {"name": "id", "in": "path", "required": True, "schema": {"type": "string"}},
                    {"name": "messageId", "in": "path", "required": True, "description": "由 POST /messages 返回的 assistant message ID", "schema": {"type": "string"}},
                ],
                "responses": {
                    "200": {"description": "SSE 流（text/event-stream）", "content": {"text/event-stream": {"schema": {"type": "string"}}}},
                    "401": err_res("Unauthorized", "未鉴权"),
                    "404": err_res("NotFound", "会话/消息不存在"),
                    "502": err_res("BadGateway", "LLM 上游不可用"),
                },
            }
        },
    },
    "components": {
        "securitySchemes": {
            "bearerAuth": {
                "type": "http", "scheme": "bearer", "bearerFormat": "JWT",
                "description": "登录/注册后返回的 JWT token，有效期 24h",
            }
        },
        "schemas": {
            "Error": {
                "type": "object",
                "properties": {
                    "success": {"type": "boolean", "example": False},
                    "error": {"type": "string"},
                    "code": {"type": "string", "description": "错误码（INVALID_INPUT / UNAUTHORIZED / FORBIDDEN / NOT_FOUND / CONFLICT / RATE_LIMITED / INTERNAL_ERROR / BAD_GATEWAY / GATEWAY_TIMEOUT）"},
                },
            },
            "RegisterRequest": {
                "type": "object",
                "required": ["email", "password", "displayName"],
                "properties": {
                    "email": {"type": "string", "format": "email", "description": "登录凭证，全库唯一"},
                    "password": {"type": "string", "minLength": 8, "maxLength": 72, "description": "明文密码（须走 HTTPS）"},
                    "displayName": {"type": "string", "minLength": 1, "maxLength": 50},
                },
            },
            "LoginRequest": {
                "type": "object",
                "required": ["email", "password"],
                "properties": {
                    "email": {"type": "string", "format": "email"},
                    "password": {"type": "string"},
                },
            },
            "AuthResponse": {
                "type": "object",
                "properties": {
                    "token": {"type": "string", "description": "JWT，有效期 24h；后续请求放 Authorization: Bearer"},
                    "user": {"$ref": "#/components/schemas/User"},
                },
            },
            "User": {
                "type": "object",
                "properties": {
                    "id": {"type": "string"},
                    "email": {"type": "string"},
                    "displayName": {"type": "string"},
                },
            },
            "CreateDocRequest": {
                "type": "object",
                "required": ["title", "country", "sourceType"],
                "properties": {
                    "title": {"type": "string", "maxLength": 200},
                    "country": {"type": "string", "maxLength": 100},
                    "sourceType": {"type": "string", "enum": ["manual", "upload", "url"]},
                    "sourceUrl": {"type": "string", "format": "uri", "description": "当 sourceType=url 时必填"},
                    "content": {"type": "string", "description": "文档正文（纯文本/markdown/HTML 字符串）；manual/upload 必填"},
                },
            },
            "KnowledgeDoc": {
                "type": "object",
                "properties": {
                    "id": {"type": "string"},
                    "title": {"type": "string"},
                    "country": {"type": "string"},
                    "sourceType": {"type": "string", "enum": ["manual", "upload", "url"]},
                    "sourceUrl": {"type": "string", "nullable": True},
                    "status": {"type": "string", "enum": ["pending", "processing", "ready", "failed"]},
                    "errorMessage": {"type": "string", "nullable": True, "description": "status=failed 时的失败原因"},
                    "chunkCount": {"type": "integer"},
                    "createdAt": {"type": "string", "format": "date-time"},
                    "updatedAt": {"type": "string", "format": "date-time"},
                },
            },
            "CreateConversationRequest": {
                "type": "object",
                "properties": {
                    "title": {"type": "string", "maxLength": 200, "description": "留空则默认为 新会话"},
                    "country": {"type": "string", "maxLength": 100},
                },
            },
            "Conversation": {
                "type": "object",
                "properties": {
                    "id": {"type": "string"},
                    "title": {"type": "string"},
                    "country": {"type": "string"},
                    "createdAt": {"type": "string", "format": "date-time"},
                    "updatedAt": {"type": "string", "format": "date-time"},
                },
            },
            "PostMessageRequest": {
                "type": "object",
                "required": ["content"],
                "properties": {
                    "content": {"type": "string", "minLength": 1, "maxLength": 10000},
                },
            },
            "Message": {
                "type": "object",
                "properties": {
                    "id": {"type": "string"},
                    "role": {"type": "string", "enum": ["user", "assistant"]},
                    "content": {"type": "string"},
                    "sources": {
                        "type": "array",
                        "description": "仅 assistant 消息；检索引用的知识块",
                        "items": {
                            "type": "object",
                            "properties": {
                                "id": {"type": "string"},
                                "title": {"type": "string"},
                                "snippet": {"type": "string"},
                            },
                        },
                    },
                    "tokensUsed": {"type": "integer", "description": "LLM token 消耗（仅 assistant 消息）"},
                    "createdAt": {"type": "string", "format": "date-time"},
                },
            },
        },
    },
}

out = Path(__file__).resolve().parent.parent / "docs" / "backend" / "api" / "openapi.yaml"
out.write_text(yaml.safe_dump(doc, allow_unicode=True, sort_keys=False, default_flow_style=False), encoding="utf-8")
print(f"已生成: {out}")
