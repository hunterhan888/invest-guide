# 前端 MVP 任务拆分

> 落点：`docs/frontend/task.md`
> 生成日期：2026-08-02
> 依据：[ARCHITECTURE.md](../../ARCHITECTURE.md) 第 708–752 行（前端架构章节）+ [AGENT.md](../../AGENT.md)

## 1. 背景与目标

仓库当前为空（只有 `ARCHITECTURE.md`、`AGENT.md`、空 `docs/`），尚无任何前端代码与提交。
本文档把交付一个**可独立运行、可演示**的前端 MVP 所需工作拆解为 Phase → Task → 子任务。

**MVP 范围**：登录鉴权 + 会话列表/详情 + 问答与 SSE 流式回答。
所有后端 API 当前未实现，采用 **Mock 先行 + 契约驱动**：以 `ARCHITECTURE.md` 的 API 约定为唯一契约，
在前端 API 客户端与真实 `fetch` 之间插一层 Mock 拦截器，前端独立可跑；后端就绪后关闭 Mock 即可切换。

## 2. 拆分原则

- **混合组织**：Phase 0 为共享基建（一次性打底），Phase 1+ 为特性纵切（每片自含 api/store/hook/page）。
- **任务粒度**：Task → 子任务。Task 是 1–2 天工作量单元；子任务是原子可执行步骤。
- **依赖标注**：仅标 Task 间前置依赖，不做工时估算。
- **验收驱动**：每个 Task 给出可勾验的验收标准。面向用户文案一律走 i18n key，禁止裸字符串。
- **门禁**：Phase 0 即建立 `bun run lint`、`bunx tsc --noEmit`、`bun run test` 三件套；任何 Task 收尾**前两项必须通过**；从 Phase 3（引入单测）起全部三项必须通过。注意测试命令是 `bun run test`（vitest），**不是** `bun test`（bun 原生测试器无法处理 jsdom/DOM）。

## 3. MVP 范围

**✅ 包含**

- 登录 / 注册（JWT）
- 会话列表（新建 / 选择 / 删除）
- 单会话消息历史（含 sources 引用展示）
- 问答输入 + SSE 流式输出（增量渲染、断线重连、心跳、错误重试）

**❌ 不在 MVP 范围**

- 知识文档管理（`KnowledgePage`、上传、状态轮询、重试）
- 暗黑模式切换 UI（仅 CSS 钩子留好：`body[data-theme=dark]`）
- `en-US` 资源（仅 zh-CN 上线，en-US key 预留占位不翻译）
- 系统页（健康检查、版本、模型列表）
- 微信等其他渠道

## 4. Mock 与契约策略

- **单向契约**：以 `ARCHITECTURE.md` "API 约定 / 路由总表 / 统一响应 / 流式 SSE" 为唯一真源。
- **契约类型层**：`frontend/src/api/types.ts` 统一存放请求/响应/SSE 事件类型，所有 mock 与 UI 共用。
- **Mock 拦截层**：`frontend/src/api/mock/` 下按领域分文件，导出 `installMocks(client)` 在 dev 模式注入。
- **SSE Mock**：用 `ReadableStream` 模拟服务端按 token 推送的 `heartbeat`/`sources`/`message`/`done`/`error` 事件，
  覆盖正常流、流中被取消、错误终止、断线重连四条路径。
- **切换开关**：`VITE_USE_MOCK=true`（默认 dev 开启）经环境变量切换，生产构建关闭。

## 5. 任务树

### Phase 0 — 工程基础设施（共享层）

一次性打底，后续特性切片均依赖此处产物。

#### Task 0.1 — Vite + React 19 + TS 脚手架

**子任务**

- 0.1.1 `frontend/` 初始化（`package.json`、`tsconfig.json` 严格模式、`vite.config.ts`、`index.html`）
- 0.1.2 安装核心依赖：`react@19`、`react-dom@19`、`react-router-dom`、`antd@5`、`@ant-design/icons`
- 0.1.3 安装数据/状态依赖：`swr`、`zustand`、`i18next`、`react-i18next`
- 0.1.4 安装样式依赖：`tailwindcss@4`、`@tailwindcss/vite`
- 0.1.5 安装开发工具：`vitest`、`@testing-library/react`、eslint、prettier
- 0.1.6 配置路径别名 `@/*` → `frontend/src/*`
- 0.1.7 `Makefile` 增加 `make frontend-dev` / `make frontend-build` 便捷入口

**验收**

- `bun install` 无错；`bun run dev` 启动 5173 端口，浏览器看到空白 antd 根
- `bunx tsc --noEmit` 通过；`bun run lint` 通过
- `import('@/main')` 解析正常；`@/` 别名可解析

**依赖**：无（所有后续 Task 前置）

---

#### Task 0.2 — 样式系统

**子任务**

- 0.2.1 `frontend/src/styles/main.css` 作为 Tailwind v4 入口，`@import "tailwindcss"`，禁用 preflight（保留轻量 base reset）
- 0.2.2 `@theme` 块定义语义化色 token，桥接 antd `--ant-*` cssVar（如 `--color-bg`、`--color-fg`、`--color-primary`、`--color-border`）
- 0.2.3 `antd-override.css` 占位（仅放深度选择器覆盖，当前为空文件 + 注释说明用途）
- 0.2.4 antd `ConfigProvider` 启 `cssVar: true`、`theme.token` 注入语义 token
- 0.2.5 `body[data-theme=dark]` 钩子接入：Tailwind `dark:` 变体绑定该属性（暂不提供切换 UI）
- 0.2.6 验证原子类 `bg-bg`/`text-fg` 可用；禁止硬编码色值

**验收**

- `main.css` 引入后页面默认浅色 token 生效；切 `body[data-theme=dark]` 后 token 切换
- 任意组件使用 `className="bg-bg text-fg"` 渲染正确颜色
- 仓库内无 `#xxxxxx` / `rgb(...)` 硬编码色值

**依赖**：0.1

---

#### Task 0.3 — 全局骨架（main / App / router / i18n / ErrorBoundary / SWR）

**子任务**

- 0.3.1 `main.tsx`：挂载 React，注入 antd `ConfigProvider`（locale=zh-CN / algorithm=light）、`SWRConfig`、`ErrorBoundary`
- 0.3.2 `router.tsx`：声明路由表（`/login`、`/`（受守卫）、`/conversations/:id`、`* → 404`），实现 `<RequireAuth>` 守卫骨架（暂只跳转 `/login`）
- 0.3.3 `App.tsx`：根布局（`<Outlet>`），放置 `<Layout>` 容器与全局 `<message>` holder
- 0.3.4 `frontend/src/i18n/config.ts`：i18next 初始化，默认 `zh-CN`；`zh-CN.json` 与 `en-US.json`（en-US 为占位、key 入口对齐）资源文件就位
- 0.3.5 `i18n` 与 antd `ConfigProvider.locale` 联动：语言变更时 antd 本地化同步切换（MVP 内只有 zh-CN）
- 0.3.6 `components/ErrorBoundary.tsx`：捕获渲染错误，显示友好回退 UI + 重载按钮
- 0.3.7 SWR 全局 `onError`：401 → 清 auth 并跳 `/login`；其他 → `message.error(i18n key)` Toast
- 0.3.8 antd `message` / `Modal` 静态方法的 `App` 上下文消费（用 `App.useApp`，避免静态调用丢主题）

**验收**

- 访问 `/` 未登录 → 跳 `/login`
- 故意拼写错误组件 → ErrorBoundary 显示回退而非白屏
- `useTranslation('common')` 在任意组件可用；文案无裸字符串
- SWR `onError` 在 mock 返回 401 时触发跳转

**依赖**：0.1（0.2 提供样式后视觉更完整，可并行但建议先 0.2）

---

#### Task 0.4 — API 客户端基础设施与 Mock 拦截

**子任务**

- 0.4.1 `frontend/src/api/client.ts`：`request<T>(method, path, body)` 统一封装
  - 自动注入 `Authorization: Bearer <token>`（从 authStore 读）
  - 解析统一响应 `{ success, data, error, code }`；`success=false` 抛业务错误（带 code）
  - 401 触发 authStore.logout + 跳 `/login`（与 SWR onError 一致）
- 0.4.2 `frontend/src/api/types.ts`：统一响应封装类型、分页类型 `Paginated<T>`、错误码枚举（与后端 HTTP 状态码映射对齐）
- 0.4.3 Mock 总入口 `frontend/src/api/mock/index.ts`：`installMocks()` 在 `VITE_USE_MOCK=true` 时注册；按 domain 分文件挂载
- 0.4.4 Mock 工具：`jsonOk(data)` / `jsonFail(code, message, status)` / `delay(ms)` / `sseStream(events)` 工厂
- 0.4.5 `vite.config.ts` 读 `.env` / `.env.local` 的 `VITE_USE_MOCK`，`src/api/index.ts` 按需激活

**验收**

- 关闭 Mock 时，对真实后端（不存在）的请求原样经 fetch（请求失败的错误由 SWR/`request` 兜底）
- 开启 Mock 时，未注册路由返回 `404 NOT_FOUND`
- `request` 自动附加 token；401 响应触发 logout + 跳转
- `sseStream` 能在测试中按顺序推事件

**依赖**：0.1（authStore 在 0.4 实现以读 token；完整 authStore 在 Phase 1 完善，此处先放读 token 的最小壳）

### Phase 1 — 鉴权切片

#### Task 1.1 — Auth API 契约 + Mock

**子任务**

- 1.1.1 `api/auth/types.ts`：`LoginRequest`、`RegisterRequest`、`AuthResponse { token, user }`、`User` 类型
- 1.1.2 `api/auth/auth.ts`：`login`、`register` 客户端方法（后端无 `/auth/me`，刷新后信任本地 token，非法 token 由首个 API 调用的 401 兜底）
- 1.1.3 `api/mock/auth.ts`：注册（200 返回 token + user，重复邮箱 409）、登录（成功 / 失败凭证 401）
- 1.1.4 错误码覆盖：`UNAUTHORIZED`、`CONFLICT`、`INVALID_INPUT`

**验收**

- mock 在 dev 模式响应三条路径；契约类型被 store 与页面复用
- 重复注册得到 409；错误登录得到 401 → 不会误登

**依赖**：0.4

---

#### Task 1.2 — authStore

**子任务**

- 1.2.1 `stores/authStore.ts`（Zustand）：state `{ token, user }`、action `{ login, logout, setUser }`
- 1.2.2 token 持久化到 `localStorage`（key: `investguide.token`），初始化时读回
- 1.2.3 `logout` 清 token + user；由 SWR/`request` 的 401 路径统一调用（避免重复实现）
- 1.2.4 提供 `useAuth()` 选择器 hook；服务端友好（避免 SSR 水合问题，本项目无 SSR 但预留）

**验收**

- 刷新页面 token 不丢
- logout 后 token / user 清空，访问 `/` 自动跳 `/login`
- 401 触发的 logout 与显式登出走同一函数

**依赖**：0.3、0.4、1.1

---

#### Task 1.3 — LoginPage 与路由守卫接入

**子任务**

- 1.3.1 `pages/LoginPage.tsx`：antd `Form`，包含登录/注册双 tab；校验（email/required、密码长度）
- 1.3.2 提交流程：调 `authStore.login` → 成功后 `navigate('/')`；失败 inline 表单错误（不 Toast）
- 1.3.3 文案全部走 i18n（`auth.login.*`、`auth.register.*`、`common.*`）
- 1.3.4 `RequireAuth` 守卫：无 token → `<Navigate to="/login" state={{from}} />`；有 token 直接放行（不调 getMe——后端无此端点；非法 token 由首个 API 请求的 401 兜底 → logout + 跳登录）
- 1.3.5 已登录访问 `/login` 自动跳 `/`

**验收**

- tab 切换登录/注册不丢已填字段
- 登录成功跳首页；失败有明确 inline 错误
- 未登录访问受保护路由 → 登录后回到 `from`
- 表单不出现原生 `<input>` / `<button>`

**依赖**：1.2

### Phase 2 — 会话管理切片

#### Task 2.1 — Conversation API 契约 + Mock + 消息历史

**子任务**

- 2.1.1 `api/conversation/types.ts`：`Conversation`、`Message`、`MessageRole`、`KnowledgeChunkRef`（sources）、`CreateConversationRequest`、`SendMessageRequest`
- 2.1.2 `api/conversation/conversation.ts`：`listConversations(page)`、`getConversation(id)`、`createConversation`、`deleteConversation`、`listMessages(convId)`、`sendMessage(convId, content)` 返回 `{ messageId }`
- 2.1.3 Mock：内存维护 conversations/messages；首次 `createConversation` 默认标题取首条消息摘要；分页符合 `{ items, total, hasMore }`
- 2.1.4 跨用户隔离（mock 内单用户即可，但 API 形状按后端契约）

**验收**

- list 分页参数正确；`hasMore` 在边界正确
- `sendMessage` 返回 `messageId`，并能随后取到该 user 消息
- types 与后端 camelCase 字段对齐

**依赖**：0.4

---

#### Task 2.2 — conversationStore + SWR hooks

**子任务**

- 2.2.1 `stores/conversationStore.ts`：state `{ activeId }`、action `setActive`/`clearActive`（activeId 不持久化，刷新清空后回首页）
- 2.2.2 `stores/uiStore.ts`：`sidebarCollapsed` + `toggleSidebar`（localStorage 持久化）
- 2.2.3 `hooks/useConversations.ts`：SWR `useSWR` 列表（key 含 token 状态）
- 2.2.4 `hooks/useConversation(id)`：详情 + 消息历史 SWR fetcher，`messageId` 映射保证消息顺序
- 2.2.5 列表 mutate：新建后插入并选中；删除后选中下一条或置空
- 2.2.6 消息 mutate hook：`mutateMessages(convId)` 暴露给 Phase 3 乐观更新

**验收**

- 新建会话后列表立即更新且自动选中
- 删除当前会话后路由跳转合理（回首页或下一条）
- 刷新后 activeId 不残留，URL 与 activeId 一致

**依赖**：1.2、2.1

---

#### Task 2.3 — 会话布局与列表侧边栏

**子任务**

- 2.3.1 `components/layout/AppLayout.tsx`：antd `Layout`（Sider + Content），Sider 可折叠（uiStore）
- 2.3.2 `components/layout/Sidebar.tsx`：会话列表 `List`，新建按钮、删除（`Popconfirm`），当前项高亮
- 2.3.3 `components/layout/UserMenu.tsx`：用户名 + 登出（antd `Dropdown`）
- 2.3.4 路由整合：`/` 默认重定向到最近会话或空白引导页；`/conversations/:id` 渲染 `ConversationPage` 占位（Phase 3 填充）
- 2.3.5 空态：无会话时显示"新建会话"引导；无 activeId 时显示欢迎卡片
- 2.3.6 i18n：`sidebar.*`、`conversation.empty.*` 全部走 key

**验收**

- 切换会话 URL 与高亮一致
- 折叠状态持久化
- 空态文案不出现裸字符串
- 不使用原生 `<input>` / `<button>` / `<select>`

**依赖**：2.2

### Phase 3 — 问答与 SSE 流式切片

#### Task 3.1 — useSSEStream hook + Mock SSE

**子任务**

- 3.1.1 `hooks/useSSEStream.ts`：`useSSEStream({ convId, messageId, onEvent, onDone, onError })`
  - 用 `fetch` + `ReadableStream` 打开 `GET /conversations/{id}/messages/{messageId}/stream`（需 JWT header，故不用 EventSource）
  - 解析 SSE 帧：`event:` / `data:` 多行拼接、空行分帧
  - 事件分发：`heartbeat`（刷新心跳时间戳）/`sources`/`message`/`done`/`error`
- 3.1.2 心跳超时检测：超过 30s（15s 心跳 + 容差）未收到任意事件 → 标记 stale 并触发重连
- 3.1.3 重连：指数退避（1s/2s/4s，最多 3 次），携带 `Last-Event-ID`，仅在流未 `done` 时重连
- 3.1.4 取消：`AbortController`，组件卸载或用户"停止"主动取消；服务端 `ctx.Done` 由后端落实，前端只保证关闭连接
- 3.1.5 Mock SSE：`api/mock/sse.ts` 用 `ReadableStream` 推送事件序列；覆盖正常 / 中途 error / 心跳间隔 / 触发重连四条路径
- 3.1.6 单元测试（Vitest）：帧解析、重连退避时序、取消后不再触发回调

**验收**

- 正常流：`sources` 一次 → `message` 多次 → `done`；UI 拼接出完整回答
- 错误流：`error` 事件终止连接，不再收到 `done`
- 手动 abort 后无回调残留
- 心跳超时触发一次重连并续传

**依赖**：2.2、1.1（stream 端点需 JWT）

---

#### Task 3.2 — 消息渲染

**子任务**

- 3.2.1 `components/conversation/MessageList.tsx`：消息按 `created_at` 排序，自动滚动到底（仅当用户已在底部时跟随）
- 3.2.2 `components/conversation/MessageBubble.tsx`：区分 user/assistant 气泡（颜色 token，无硬编码），支持流式增量（pending 状态光标）
- 3.2.3 `components/conversation/SourcesCard.tsx`：展示 `sources.chunks`（title + snippet），点击展开（Phase 3 纯展示，不跳转知识库详情——超出 MVP）
- 3.2.4 `components/conversation/MarkdownRenderer.tsx`：安全渲染 Markdown——引入 `react-markdown` + `remark-gfm`；默认禁用 `raw HTML`（不传 `rehype-raw`），代码块用 `rehype-highlight`；MVP 不接受内联 HTML，从而杜绝 XSS 面
- 3.2.5 i18n：`message.sources.*`、`message.streaming.*` 走 key
- 3.2.6 测试：`MessageList` 排序、`SourcesCard` 渲染

**验收**

- Markdown 标题/列表/代码块正确渲染；不渲染内联脚本
- 流式时显示光标，`done` 后光标消失
- 长列表滚动跟随；用户向上回看时不强制下滚

**依赖**：3.1（事件来源就位后接渲染；可并行先做静态部分）

---

#### Task 3.3 — 消息输入与乐观更新

**子任务**

- 3.3.1 `components/conversation/MessageComposer.tsx`：`Input.TextArea` + 发送按钮；Enter 发送 / Shift+Enter 换行
- 3.3.2 乐观更新：发送前立即把 user 消息插入消息列表（临时 `pending` id），同时占位一条 assistant 占位气泡（流式前）
- 3.3.3 调 `sendMessage` 拿 `messageId` 后开启 `useSSEStream`；流式中禁用输入；出现 `done` 解禁
- 3.3.4 inline 错误：`error` 事件在 assistant 气泡下方显示错误 + "重试"按钮；重试复用相同 user 消息内容
- 3.3.5 服务端取消"停止生成"按钮：abort 当前 SSE
- 3.3.6 空内容拦截；超长内容（> 阈值）提示
- 3.3.7 i18n：`composer.*`、`error.retry.*` 走 key

**验收**

- 发送后立即看到 user 气泡 + assistant 占位；流式拼接过程中不抖动
- 流式中无法重复发送
- 错误事件展示 inline 重试而非整页白屏
- 停止生成关闭流； assistant 气泡保留已收片段

**依赖**：3.1、3.2

---

#### Task 3.4 — 首页落地与端到端走查

**子任务**

- 3.4.1 `pages/HomePage.tsx`：未选会话时落地页（最近会话快捷 + 新建按钮 + 简介文案 i18n）
- 3.4.2 `pages/ConversationPage.tsx` 完整集成：`AppLayout` + `MessageList` + `MessageComposer`
- 3.4.3 端到端 mock 走查脚本：登录 → 新建 → 发问 → 看到完整回答 → 删除会话 → 登出
- 3.4.4 全量 `bun run lint && bunx tsc --noEmit && bun run test` 必过
- 3.4.5 `README.md` 增加"前端本地运行"小节（含 `VITE_USE_MOCK` 说明）
- 3.4.6 文档同步检查：若实现与 `ARCHITECTURE.md` 出现偏差，同 PR 内更新 ARCHITECTURE.md / AGENT.md（按 AGENT.md 强制要求）

**验收**

- 全程 mock 完整跑通问答闭环
- 三件套全绿
- README 新增运行说明
- 与架构文档无静默漂移

**依赖**：3.1、3.2、3.3、2.3

## 6. 任务依赖图

```
Phase 0
  0.1 ──┬──> 0.2
        ├──> 0.3 ──> (Phase 1)
        └──> 0.4 ──> (Phase 1)

Phase 1
  1.1 (auth api+mock) ──> 1.2 (authStore) ──> 1.3 (LoginPage + 守卫)

Phase 2
  2.1 (conv api+mock) ──> 2.2 (store+SWR) ──> 2.3 (布局+Sidebar)
                          ▲                      │
                         ┌┘                      │
   Phase 1 (1.3 守卫) ───┘                       │
                                                 ▼
Phase 3                                    2.3 完成
  3.1 (useSSEStream + mock SSE) ──┬──> 3.3 (MessageComposer)
                                   │      │
                                   └──> 3.2 (MessageList 等，可与 3.1 部分并行)
                                                   │
                                                   ▼
                                              3.4 (集成走查)
```

**可并行点**

- 0.2 / 0.3 / 0.4 在 0.1 后可并行
- 2.1 与 1.x 阶段无依赖（理论上可与 Phase 1 并行，但建议先让 Phase 1 打通鉴权体验）
- 3.2 的静态部分（非流式渲染）可在 3.1 开发期间并行起步

## 7. Mock 切换与收尾

- 所有切片在 dev 阶段统一走 Mock；对接真实后端时仅切换 `VITE_USE_MOCK`，无需改 UI 代码。
- Phase 3.4 收尾后，关闭 Mock 对接真实后端（即使后端尚未就绪，CI 也以 Mock 模式跑过三件套）。
- 后续阶段（知识管理、暗黑切换 UI、en-US 资源、系统页）以独立 spec 启动，不在本任务树内。
