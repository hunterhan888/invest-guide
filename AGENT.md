# Invest Guide — 项目指南

所有贡献者（人类与 AI）在修改结构前必须先阅读 [ARCHITECTURE.md](ARCHITECTURE.md)，
并保证每次代码变更都同步更新设计文档（参见 [设计文档同步](#设计文档同步强制要求)）。

## 代码规范

### 文件与目录结构

- **目录容量限制**：每个目录的直接子项建议 ≤ **10** 个；新建或大幅重组的目录必须满足此要求。
- **Monorepo 布局** — 代码只放在两个顶层目录下，根目录不放代码：

| 目录 | 内容 |
|------|------|
| `backend/`  | Go + Gin API 服务 |
| `frontend/` | React + TypeScript Web 应用（Vite） |

### 命名

**前端（TypeScript）**

- 组件：PascalCase（`Button.tsx`、`QaPanel.tsx`）
- 页面：PascalCase 并加 `Page` 后缀（`HomePage.tsx`、`ConversationPage.tsx`）
- 工具函数：camelCase（`formatDate.ts`）
- Hooks：camelCase 并加 `use` 前缀（`useConversation.ts`）
- 常量文件：camelCase（`constants.ts`）；文件内常量值用 `UPPER_SNAKE_CASE`
- 类型文件：camelCase（`types.ts`）
- 样式文件：kebab-case 或 `ComponentName.module.css`
- 未使用的参数：加 `_` 前缀

**后端（Go）**

- 文件名：`snake_case.go`
- 导出标识符：PascalCase；非导出：camelCase（标准 Go 规范）
- 接口：描述性命名，不加 `I` 前缀（`ConversationRepository`、`LLMProvider`）
- Handler 函数：`handleXxx`（`handleCreateConversation`）
- 错误变量：`ErrXxx`（`ErrNotFound`、`ErrConflict`）

### UI 库与图标（仅前端）

- 组件：`antd`（Ant Design v5，启 `cssVar: true`）— 不使用原生交互 HTML
  （`<button>`、`<input>`、`<select>` 等）
- 图标：`@ant-design/icons`
- antd 主题与 locale 经根组件 `<ConfigProvider>` 注入；组件级定制优先用
  `theme.token` / `theme.components`，必要时在 `frontend/src/styles/antd-override.css`
  做深度选择器覆盖

### CSS（前端）

- 优先使用 **Tailwind v4 原子类**；复杂样式使用 **CSS Modules**
  （`ComponentName.module.css`，内部可用 `@apply` 引用 token）
- 颜色必须使用 `frontend/src/styles/main.css` 中 `@theme` 定义的语义化 token
  （桥接自 antd 的 `--ant-*` cssVar）或 CSS 变量 — 禁止硬编码色值
- 全局样式只放在 `frontend/src/styles/`（`main.css` 为 Tailwind 入口，
  `antd-override.css` 用于 antd 深度覆盖）
- Preflight 已禁用以避免破坏 antd 注入的组件样式；轻量 base reset 仅在 `main.css` 内
- 暗黑模式：`body[data-theme=dark]` 是 CSS 侧的唯一切换点；
  Tailwind 的 `dark:` 变体绑定到该属性，antd 由 `ConfigProvider.algorithm` 驱动

### TypeScript

- 开启严格模式 — 禁止 `any`、禁止隐式返回
- 路径别名：`@/*` → `frontend/src/*`
- 优先使用 `type` 而非 `interface`
- 代码注释用英文；公开函数使用 JSDoc
- API 契约类型放在 `frontend/src/api/`（与对应 API 客户端同模块），必须与后端 API 契约保持一致
  （见 [ARCHITECTURE.md](ARCHITECTURE.md#api-约定)）

### Go

- 格式化：强制 `gofmt`；`go vet` 必须无任何告警
- 错误即值 — 必须显式处理；请求路径中禁止 `panic`
- 使用 `errors.Is` / `errors.As` 匹配错误；用 `%w` 包装
- 全程传递 `context.Context`；禁止全局可变状态；依赖通过注入获得
- SQL 只用参数化查询（GORM 或 `database/sql`）— 禁止字符串拼接
- 注释、日志、提交信息使用中文

### 国际化（i18n）

新增或修改面向用户的文案必须使用 i18n key；禁止硬编码字符串。
语言在 `frontend/src/i18n/config.ts` 配置（默认 `zh-CN`，并支持 `en-US`）。
antd 组件自身的文案（DatePicker、Modal 默认按钮等）由根组件 `<ConfigProvider>`
按 `i18n.language` 切换 locale，与应用 i18n 同步切换。

## 架构

**边界规则**：前端与后端**只**通过 HTTP API（`/api/v1`）通信，其他任何形式都不得越过此边界。

| 层 | 路径 | 限制 |
|----|------|------|
| 后端 | `backend/internal/` | 不含前端/DOM 概念；全部业务逻辑都在此层（`platform/` 基础层 + `domain/` 领域层） |
| 前端 | `frontend/src/` | 不直接访问数据库或 LLM；所有数据经 `frontend/src/api/` |

- Web 前端是首个客户端渠道。后续渠道（微信等）通过同一 HTTP API 接入 —
  不得为其新建独立接口。
- 鉴权、限流、CORS、日志均为后端中间件职责。
- 模块边界与依赖规则见 [ARCHITECTURE.md](ARCHITECTURE.md)。

## 测试

**后端** — `go test`，优先使用表驱动测试；`internal/` 覆盖率目标 ≥ 70%；
新增行为应附带聚焦测试。

```bash
cd backend && go test ./...           # 运行全部测试
cd backend && go test ./... -cover    # 附带覆盖率报告
```

**前端** — Vitest。

```bash
cd frontend && bun run test              # 运行全部测试（vitest）
cd frontend && bun run test:coverage     # 附带覆盖率报告
```

> 注意：Vitest 依赖 jsdom 环境与 `vite.config.ts` 配置，**必须用 `bun run test`**（映射到 `vitest run`）。直接 `bun test` 会走 bun 原生测试器，无法解析 `localStorage`/DOM，且失败时退出码为 0，不可作为门禁。

测试文件就近放在被测代码旁，命名为 `*.test.ts(x)`；仅当一个特性需要跨文件集成验证时
才在 `frontend/src/<scope>/__tests__/` 下建目录。根目录不设顶层 `tests/`。

依赖 LLM 的测试必须使用输出确定的 fake `LLMProvider` — 测试中禁止调用真实模型。

## 工作流

### 范围与执行

- **硬性阻断项**：后端编译错误、测试失败、`go vet` 告警、TypeScript 错误、
  新增/修改的面向用户文案缺失 i18n、新 UI 中出现原生交互 HTML，以及
  **设计文档漂移**（改动架构或 API 却未同步更新 AGENT.md / ARCHITECTURE.md）。
- **本次变更要求**：命名、CSS、文件归属、测试、文档、目录容量规则适用于本次
  新建或实质性修改的文件。
- **棘轮规则**：既有违规在常规开发中无需清理，但本次变更不得使其恶化。
- **禁止范围扩张**：计划与评审不得擅自新增清理任务，除非用户明确要求该范围。

### 开发过程中

```bash
# 后端
cd backend && gofmt -l .          # 列出需格式化的文件
cd backend && go vet ./...        # 静态检查
cd backend && go test ./...       # 运行测试

# 前端
cd frontend && bun run lint:fix   # 自动修复 lint 问题
cd frontend && bun run format     # 自动格式化全部文件
cd frontend && bunx tsc --noEmit  # 校验无类型错误
```

### 设计文档同步（强制要求）

设计文档是活文档。在引入代码变更的**同一次变更**中：

- 架构 / 模块边界 / API / 数据模型变更 → 更新 **ARCHITECTURE.md**
- 规范或工作流变更 → 更新 **AGENT.md**

不得将文档更新推迟到后续清理。

### 推送前

AI 未经明确要求不得推送。推送前运行：

```bash
cd backend && gofmt -l . && go vet ./... && go test ./...
cd frontend && bun run lint && bun run format && bunx tsc --noEmit && bun run test
```

任一步失败即中止推送。修复、提交后重试。

### 提交与 PR 格式

提交信息与 PR 标题遵循 Conventional Commits：

```text
<type>(<scope>): <subject>
```

允许的 type：`feat`、`fix`、`perf`、`refactor`、`docs`、`style`、`chore`、`test`、
`ci`、`build`。

**禁止添加 AI 签名**（Co-Authored-By、"Generated with" 等）。

## 参考文档

| 文档 | 用途 |
|------|------|
| `AGENT.md` | 本仓库的贡献者与 Agent 规范（本文件） |
| `ARCHITECTURE.md` | 系统架构、模块职责、API 约定、安全与测试策略 |
| `docs/backend/api/openapi.yaml` | API 契约唯一事实来源（OpenAPI 3.0），前后端联调与测试均以此为准 |
| `docs/mcp.md` | MCP server 接入说明（Agent 如何通过 MCP 调用后端能力） |
