# InvestGuide 前端 UI 改版设计（DeepSeek 风格）

> 落点：`docs/frontend/design-restyle.md`
> 日期：2026-08-03
> 状态：已批准（浏览器确认 + 终端确认）
> 范围：**仅样式与布局改版，不修改任何功能逻辑/数据流/API 契约/测试行为**

## 1. 背景与目标

当前前端使用 antd v5 默认主题，未做设计定制：字号偏小（14px）、布局为传统侧栏+内容区、消息气泡为 antd 蓝色、无视觉层次。用户反馈"缺乏设计感"。

目标：借鉴 **chat.deepseek.com** 的设计语言，对浅色主题做**全面改版**：

- 克制冷灰色系、深色文字、近白底
- 更大字号（15px 基础）与舒适行高
- 居中对话列（最大宽度 680px）
- 柔和圆角、极简侧栏、细分割线
- 纯黑主按钮作为唯一强调色

**已确认决策**：
1. DeepSeek 风格（浏览器选项 A 已确认）
2. 只做浅色主题（深色模式本轮不做；保留 `body[data-theme=dark]` 钩子不被破坏）
3. 全面改版（antd token 重构 + 布局结构重写 + 组件样式）
4. 不动逻辑：路由、store、hooks、API 契约、测试行为全部保持

## 2. 设计语言（Design Tokens）

在 `frontend/src/App.tsx` 的 `ConfigProvider.theme.token` 定义，`main.css` 的 `@theme` 块桥接为 Tailwind 语义 token。

| Token | 值 | 说明 |
|-------|-----|------|
| `colorBgLayout` | `#f7f8fa` | 页面底色 |
| `colorBgContainer` | `#ffffff` | 卡片 / 输入框 / 侧栏 |
| `colorBgElevated` | `#ffffff` | 悬浮层 |
| `colorText` | `#1a1a1a` | 正文 |
| `colorTextSecondary` | `#6b7280` | 次要文本 |
| `colorTextTertiary` | `#9ca3af` | 占位符 / 弱化 |
| `colorTextQuaternary` | `#d1d5db` | 禁用弱化 |
| `colorBorder` | `#eef0f3` | 极细分隔线 |
| `colorBorderSecondary` | `#f0f1f3` | 更弱分隔 |
| `colorPrimary` | `#111111` | 纯黑主按钮 / 强调（DeepSeek 风格） |
| `colorPrimaryHover` | `#2a2a2a` | 主按钮 hover |
| `colorPrimaryActive` | `#000000` | 主按钮 active |
| `borderRadius` | `10` | 组件圆角基础 |
| `borderRadiusLG` | `16` | 大圆角（输入卡片） |
| `borderRadiusSM` | `6` | 小圆角 |
| `fontSize` | `16` | 全局基础字号（原 14） |
| `fontSizeHeading1..5` | 按 antd 相对比例 | 标题随基础放大 |
| `lineHeight` | `1.6` | 基础行高 |
| `controlHeight` | `40` | 控件高度（输入/按钮略高） |
| `controlHeightLG` | `44` | 大控件 |

### Tailwind 语义 token 桥接（`main.css` `@theme`）

```
--color-bg:        var(--ant-color-bg-container)     → bg-bg
--color-bg-layout: var(--ant-color-bg-layout)        → bg-bg-layout
--color-fg:        var(--ant-color-text)             → text-fg
--color-fg-secondary: var(--ant-color-text-secondary) → text-fg-secondary
--color-fg-tertiary:  var(--ant-color-text-tertiary)  → text-fg-tertiary
--color-primary:   var(--ant-color-primary)          → bg-primary
--color-border:    var(--ant-color-border)           → border-border
--color-bubble-user: #e8e9ee                         → 用户消息气泡底（新增，非 antd token）
```

## 3. 布局（AppLayout）

- **侧栏**：宽度 240px，浅色背景 `#ffffff`，右边框 `#eef0f3`；顶部品牌标题（16px、字重 700、`-0.2px` 字距）；`新建会话` 按钮改为白底细边框圆角（radius 12px）而非 antd 主色填充。
- **内容区**：消息列**水平居中**，内容最大宽度 **680px**。
- **顶部 header**：保留，高度 48px，下边框 `#eef0f3`，会话标题 15px 字重 600。
- 折叠态（Sider collapsible）保持可用，折叠后仅图标。

## 4. 组件改造

### MessageList
- 改为居中列：`mx-auto w-full max-w-[680px]`
- 消息间距 24px（`space-y-6`），垂直留白 24px
- 保持滚动跟随逻辑不变

### MessageBubble
- **用户消息**：灰气泡 `#e8e9ee`，圆角 18px，内边距 `12px 18px`，字号 15px 行高 1.6，文字色 `#111`（**不再是白字蓝底**），右对齐
- **assistant 消息**：**无气泡**，透明底、通栏（不设 max-width 气泡）、字号 15px 行高 1.75、文字色 `#1a1a1a`
- 流式光标保留（`message.streaming.cursor`）
- sources 区块下移

### MessageComposer
- 输入卡片：白色底、`border 1px #e5e7eb`、圆角 16px、内边距 `12px 16px`、浅阴影 `0 1px 3px rgba(0,0,0,.04)`，**居中 680px**
- 发送按钮：纯黑主色（token `colorPrimary`），与输入框同卡片底部右侧
- 流式中：Stop 按钮保留（危险色），输入禁用
- Enter 发送 / Shift+Enter 换行、乐观更新逻辑**不变**

### SourcesCard
- 由 antd Collapse 改为**细线条引用**：`📎 引用 · title`（13px、`#6b7280`），点击展开 snippet
- 实施方式明确：**保留 antd `Collapse` 组件**（维持现有展开/收起交互与测试），仅通过样式覆盖实现细线条外观（`items` label 用 `📎 引用 · title`，`ghost` 模式、紧凑间距）

### HomePage
- 居中大标题 22px 字重 700、`-0.3px`
- 圆形渐变 logo 占位（56px，`linear-gradient(135deg,#eef0f3,#e3e6eb)` 圆角 16px）
- 纯黑胶囊主按钮（radius 999px）
- 副标题与 footer 弱化文案

### LoginPage
- 居中白色圆角卡片：`#ffffff`、`border 1px #eef0f3`、radius 20px、阴影 `0 4px 20px rgba(0,0,0,.05)`、内边距 28px
- 标题 20px 字重 700；副标题 13px `#6b7280`
- 输入框细边框圆角 10px；主按钮纯黑
- 登录/注册双 tab 保留

### Sidebar / UserMenu
- 会话列表项：圆角 10px，hover 浅灰 `#f3f4f6`，当前项 `#f3f4f6` 高亮（非 antd 蓝）
- 字号 14px；删除操作保留
- UserMenu：简洁头像 + 用户名，登出下拉保留

### MarkdownRenderer
- 行高 1.75、字号 15px（跟随容器）
- 代码块浅底 `#f6f8fa` 圆角 8px
- 引用块、列表留白改善
- `skipHtml` 安全策略**不变**

### ErrorBoundary / 其他
- 保持 antd Result，随 token 自动适配，不做单独改动

## 5. 明确不动的内容

- 全部功能逻辑：路由、store、hooks、API 客户端、mock、SSE 消费
- 全部 API 契约与类型
- i18n key（除视觉相关文案外不增删）
- 现有测试行为：38 个测试应继续通过；仅视觉类断言（如颜色/类名）若受影响则按新 markup 微调，功能断言不变
- `body[data-theme=dark]` 钩子保留（不实现深色）

## 6. 验证方式

1. `cd frontend && bun run lint && bunx tsc --noEmit && bun run test` 全绿
2. Playwright 无头 E2E 复跑：登录 → 新建会话 → 发送提问 → 流式回答 → 引用来源，确认 UI 功能完好
3. 浏览器人工确认：视觉符合 DeepSeek 风格
4. 文档同步：若 AGENT.md/ARCHITECTURE.md 的前端样式约定有变更，同步更新

## 7. 不在本轮范围

- 深色模式实现（保留钩子）
- 响应式 / 移动端适配
- 新的功能组件（如 Markdown 编辑器、文件上传等）
- 动画/动效系统

## 8. 实施备注（2026-08-03，已完成）

- **body token 解析问题**：antd cssVar 变量（`--ant-*`）只挂在 ConfigProvider 渲染的容器上，不在 `body`/`html` 上。`main.css` 的 `@theme` token 原直接引用 `var(--ant-*)`，导致 `body` 背景/文字色解析失败（回退默认黑/透明）。修复：`@theme` 中每个 token 带显式 fallback（`var(--ant-color-text, #1a1a1a)`），body 用 fallback 值，容器内组件仍解析 antd 主题化值。已验证 `body` 计算样式为 `#f7f8fa` / `#1a1a1a` / 15px。
- **测试断言微调**：`MessageList.test.tsx`、`ConversationPage.test.tsx` 中来源标题断言由精确匹配改为正则子串（`getByText(/来源/)`），因 SourcesCard label 变为组合文本 `📎 引用来源 · 标题`。功能断言未变。
- **验证**：`bun run lint && bunx tsc --noEmit && bun run test` 全绿（38 测试）；Playwright 无头 E2E 对真实后端全流程通过（登录→注册→新建会话→流式回答→引用来源），视觉 token 正确。

## 9. 比例精修（2026-08-03）

基于 Playwright 提取的实际计算样式数据，修正以下比例不协调：

- **字体太小**：全局 `fontSize` 15 → 16（DeepSeek 同级），body 与 `.md-body` 同步，用户气泡/首页副标题改用 `text-base`。
- **品牌标题高度**：`h-14` 被 antd `controlHeight` 覆盖为 40px，改 `!h-12`（48px）与内容 header 对齐；标题字号 `text-base`→`text-lg`。
- **会话标题层级弱**：header 加 `font-semibold`（实测 fw 400→600），左侧留白 `px-4`→`px-6`。
- **首页 CTA 重量不足**：加 `CompassOutlined` 图标，标题 `whitespace-nowrap` 防折行。
- **登录副标题比例陡**：13px → `text-sm`（14px）。
- **Composer 输入框**：textarea 残留圆角 10px，改 `!rounded-none !border-none !shadow-none` 与卡片融为一体；发送/停止按钮改 `!rounded-full` 胶囊形，现代聊天输入形态。
