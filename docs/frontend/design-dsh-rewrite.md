# InvestGuide 前端重写设计（全面照搬 DeepSeek-Harness 设计体系）

> 落点：`docs/frontend/design-dsh-rewrite.md`
> 日期：2026-08-19
> 状态：已批准（交互式头脑风暴逐项确认）
> 范围：**推倒现有 antd + Tailwind 架构，完整照搬 DeepSeek-Harness 前端设计体系重写 UI 层；全部功能逻辑/数据流/API 契约/测试行为保持不变**

## 1. 背景与目标

invest-guide 前端使用 antd v5 默认主题 + Tailwind 原子类，用户认为"不好看、不专业"。参考项目 `deepseek-harness`（`/home/hunter/code/deepseek-harness`）拥有一套完整、专业的设计体系：`--dsw-*` 静态色板 + 语义化别名 token、CSS Modules、自研基础组件、三栏可拖拽布局、双主题（浅/暗）、克制的中性色 + DeepSeek 品牌蓝。

目标：**完全照搬 DeepSeek-Harness 的设计语言与技术栈**，重写 invest-guide 前端，使其观感专业、统一、可维护。

**已确认决策**：
1. 技术栈完全照搬：CSS Modules + 自研基础组件 + `--dsw-*` token 体系，**彻底移除 antd、tailwindcss、@ant-design/icons**
2. 保留现有全部功能：登录/注册、会话列表侧栏（今天/昨天/更早分组）、首页欢迎态、会话聊天页（用户气泡/助手 Markdown/来源引用/流式 SSE/停止/重试）
3. 来源引用保留「来源卡 + 内联 [N] 芯片」交互，用 DeepSeek 设计语言重绘
4. 双主题（浅色/暗色）+ 用户切换，`body[data-ds-dark-theme]` 为唯一 CSS 切换点，zustand 持久化
5. 品牌：主色 `deepseek-500 #4176E6`（品牌蓝）；自研简洁指南针 logo（SVG 渐变），不用 🤖
6. Markdown 渲染：保留 react-markdown + remark-gfm + rehype-highlight，仅重绘样式
7. 基础设施保留：zustand、react-router-dom、swr、i18next（不照搬 DeepSeek-Harness 的运行时/状态方案）
8. 布局：照搬 AppFrame 三栏（可折叠侧栏 + 对话主区 + 可拖拽/可收起右侧详情栏）
9. 右侧详情栏：显示当前选中 assistant 消息的来源列表 + 元信息（tokens/时间/引用数）

## 2. 技术栈与目录结构

| 层 | 方案 |
|---|---|
| 样式 | CSS Modules（`ComponentName.module.css`）+ 设计 token CSS 变量，无 Tailwind |
| UI 组件 | 自研基础组件：`Button` / `Input` / `Modal` / `Menu` / `Tooltip` / `Toast` / `Pill` / `DisclosureRow` |
| 图标 | 自研轻量 SVG 图标集（替换 `@ant-design/icons`） |
| 主题 | 双主题 token（浅/暗），`body[data-ds-dark-theme]` 切换，`themeStore`(zustand) 持久化 |
| 状态/路由/数据 | 保留 zustand + react-router-dom + swr + i18next |
| Markdown | 保留 react-markdown + remark-gfm + rehype-highlight |

**依赖变更**：
- 移除：`antd`、`@ant-design/icons`、`tailwindcss`、`@tailwindcss/vite`
- 保留：react、react-dom、react-router-dom、react-i18next、i18next、react-markdown、remark-gfm、rehype-highlight、swr、zustand

**目录结构（`frontend/src/`，满足 AGENT.md 目录 ≤10 子项规则）**：

```
frontend/src/
├── styles/          # tokens.css（静态色板/语义别名/scrollbar/shiki 等）+ main.css 入口
├── theme/           # ThemeProvider + themeStore + 品牌 logo
├── primitives/      # 基础组件 Button/Input/Modal/Menu/Tooltip/Toast/Icon/DisclosureRow/Pill
├── layout/          # AppFrame/Sidebar/DetailsPanel/UserMenu
├── pages/           # LoginPage/RegisterPage/HomePage/ConversationPage
├── features/        # conversation/(chat+sources) + home/(composer) + auth/(表单)
├── api/             # 保持不变
├── stores/          # 保持不变 + themeStore 新增
├── hooks/           # 保持不变 + 新增（useResize/useTheme 等）
├── i18n/            # 保持不变
└── main.tsx         # 入口
```

## 3. 设计 token 与主题体系

### 3.1 静态色板（`styles/tokens.css`）

从 `deepseek-harness/packages/client/ui-theme/src/styles/design-platform.css` 搬入全套 `--dsw-static-*`：

- bluish 中性色阶：`--dsw-static-neutral-bluish-{00,50,60,75,100,150,200,300,400,500,600,700,750,800,850,875,900,950,1000}`
- 纯中性色阶：`--dsw-static-neutral-{00,50,100,150,200,250,300,400,500,550,600,700,800,850,900,1000}`
- DeepSeek 品牌蓝：`--dsw-static-deepseek-{50,100,200,300,400,450,500,600,700-delete,800,900}`
- 状态色：`--dsw-static-red-{50,100,400,500,600,900}`、`--dsw-static-green-{100,400,500,900}`、`--dsw-static-amber-{100,400,500,600,900}`
- 通用蓝：`--dsw-static-blue-{50,50p,75,100,300,400,450,500,600,800,900,950}`
- 明暗两套静态定义（`body` 与 `body[data-ds-dark-theme]`），与参考实现保持一致

### 3.2 语义别名（明暗两套）

照搬核心别名体系：

- 背景层：`--dsw-alias-bg-base` / `layer-1` / `layer-2` / `layer-3` / `overlay` / `mask-*`
- 文字：`--dsw-alias-label-primary` / `secondary` / `tertiary` / `caption` / `dimmed`
- 边框：`--dsw-alias-border-l1` / `l2` / `l3` / `l4`
- 交互态：`--dsw-alias-interactive-bg-hover` / `hover-accent` / `hover-danger` / `hover-solid` / `active`
- 按钮：`--dsw-alias-button-primary-fill` / `primary-hover` / `ghost-active-*` / `floating-*` / `elevated-fill` / `contrast-fill` / `info-fill` / `info-hover`
- 特定区域：`--dsw-specific-bubble` / `bubble-highlight` / `sidebar-fill` / `sidebar-nav-item-*` / `input-major` / `menu` / `selector` / `tip`
- Markdown：`--dsw-alias-markdown-code-block` / `code-block-banner` / `inline-code` / `citation` / `tag` / `placeholder`
- 状态：`--dsw-alias-state-error-primary/secondary`、`state-success-*`、`state-warn-*`、`state-business-*`
- 滚动条：`--dsw-alias-scrollbar-bg-l1/l2`、`scrollbar-hover-l1/l2`
- Toast/Tooltip：`--dsw-alias-toast-bg` / `tooltip-bg`

### 3.3 字体与动效（`styles/base.css`）

```css
:root {
  --dsw-font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC',
    'Hiragino Sans GB', 'Microsoft YaHei', 'Helvetica Neue', Helvetica, Arial, sans-serif;
  --ds-font-family-code: 'SF Mono', 'JetBrains Mono', 'Fira Code', Consolas,
    'Liberation Mono', Menlo, Courier, 'PingFang SC', 'Microsoft YaHei';
  --ds-ease-in-out: cubic-bezier(0.4, 0, 0.2, 1);
  --ds-transition-duration: 0.2s;
  --ds-transition-duration-fast: 0.1s;
  --ds-transition-duration-slow: 0.3s;
}
```

### 3.4 明暗切换机制

- `body[data-ds-dark-theme]` 是 CSS 侧唯一切换点（弃用现有 `data-theme=dark`）
- `themeStore`（zustand + localStorage 持久化，key 沿用 `investguide.` 前缀）：`'light' | 'dark'`
- `<ThemeProvider>` 在根组件挂载时写入 `<body data-ds-dark-theme>`；`prefers-reduced-motion` 时禁用动效
- 移除 `App.tsx` 的 antd `ConfigProvider` 与 `theme` 配置

### 3.5 品牌

- 主色用 `deepseek-500 #4176E6`
- 自研指南针 logo（SVG + 渐变，`theme/logo.tsx`），替换 HomePage 的 🤖 与 LoginPage 的 CompassOutlined

## 4. 布局 —— AppFrame 三栏

### 4.1 `AppFrame`（`layout/AppFrame.tsx` + `.module.css`）

照搬 DeepSeek-Harness `ui-layout/AppFrame`：

- CSS Grid 三列（`grid-template-columns`），`grid-template-rows: 100%`，`height: 100%`，背景 `--dsw-alias-bg-base`
- 侧栏列：`--dsw-specific-sidebar-fill` + 右 1px `--dsw-alias-border-l1`，`overflow: hidden`
- 中栏：`display: flex; flex-direction: column; overflow: hidden`
- 详情列：左 1px `--dsw-alias-border-l2`，`overflow: hidden`
- 拖拽手柄：8px 命中条（`position: absolute; cursor: col-resize`），详情侧可见 12×32 竖排 pill（悬停/拖拽显现）
- 拖拽中 `data-dragging` 关闭过渡；折叠/展开走 `--ds-transition-duration-slow` 曲线
- `prefers-reduced-motion` 禁用过渡
- 列宽持久化到 localStorage

### 4.2 `Sidebar`（`layout/Sidebar.tsx` + `.module.css`）

照搬 `ui-sidebar/SidebarRoot` 机制：

- 顶部品牌行：logo + 折叠开关（28px 圆形 icon 按钮）
- 新建会话：38px 高、12px 圆角胶囊（`border-l2` 描边 + `button-elevated-fill`），悬停 `button-floating-hover`
- 中部会话列表：按今天/昨天/更早分组，圆角 10px 行，hover `interactive-bg-hover`，当前项 `sidebar-nav-item-active` + `sidebar-nav-item-active-accent`（左侧 accent 条）
- 行悬停显示「更多」按钮 → 自研确认弹窗删除
- 底部：用户菜单 + 折叠开关
- 折叠态：56px rail，fade/slide 动画（照搬 `.fading` / `.railIn` / `.collapsed` 机制）
- 静音滚动条：非 hover 时 thumb 透明（照搬 `quietBars`），`scrollbar-gutter: stable`

### 4.3 `DetailsPanel`（`layout/DetailsPanel.tsx` + `.module.css`）

- 默认收起（宽度 0，border 不绘制），悬停手柄或选中消息时展开
- 内容：当前选中 assistant 消息的来源列表 + 元信息（tokens、时间、引用数）
- 点击正文 `[N]` 芯片 → 展开右栏 + 滚动高亮对应来源（复用现有 `handleSourceRef` 逻辑）
- 空态（无选中/无来源）：弱化占位文案

**与内联来源卡的职责划分**（避免重复渲染）：
- 消息下方的内联「来源」折叠卡保留，用于**就地快速浏览**当前消息的完整来源
- 右侧详情栏聚焦**选中消息的引用上下文与元信息**；当右栏展开且选中该消息时，内联卡与右栏展示同一来源集合，但右栏同时给出 tokens/时间/引用数等元信息
- `[N]` 芯片点击优先联动右栏（若右栏收起则先展开）；右栏收起时回退为展开内联卡并滚动高亮

### 4.4 顶栏

- 会话页顶部轻量 header（标题 + 新建按钮），DeepSeek 设计语言重绘

## 5. 核心页面与组件

### 5.1 `HomePage`

- 居中欢迎态：品牌蓝渐变指南针 logo（替换 🤖）+ 标题 + 副标题
- 中央大输入卡：`--dsw-specific-input-major` 背景、16px 圆角、`border-l2` 描边、聚焦品牌蓝边
- 右下圆形发送按钮（胶囊 primary）
- 自研 `Composer` 组件（首页与会话页共用；首页额外支持回车跳转新会话）

### 5.2 `ConversationPage`

- 消息流：`max-width: var(--dsh-chat-content-width)`（≈736px）居中，16px 列间距
- 用户消息：右对齐气泡，`--dsw-specific-bubble`（浅色 `deepseek-50`，暗色 `neutral-850`）、22px 圆角、`max-width: min(525px, 82%)`、padding `10px 16px`、无头像
- 助手消息：左侧品牌蓝渐变圆形图标 + 全宽 Markdown 列（无气泡背景）
- 助手消息下方保留内联「来源」折叠卡（`DisclosureRow` 实现，替代 antd Collapse），展示当前消息来源，点击 `[N]` 联动右栏（见 4.3）
- 流式：增量渲染（沿用 SSE 逻辑）+ 品牌蓝 shimmer 状态行（照搬 `.turnStatus` 渐变扫光）+ 停止按钮
- 错误/重试：自研折叠样式（照搬 retry `details`/`summary` 视觉，用 div+chevron 实现）
- 底部浮动「回到最底」圆形按钮（照搬 `.toBottom`）
- 保留自动滚动跟随逻辑

### 5.3 Markdown 渲染 `MarkdownRenderer`

- 保留 react-markdown + remark-gfm + rehype-highlight
- 样式用 DeepSeek 设计重绘：代码块 banner（`markdown-code-block-banner`）、内联代码（`markdown-inline-code`）、引用、标题、表格
- `[N]` 来源芯片：DeepSeek 风格 citation chip（照搬 `refChip`：圆角 6px、`rgba(97,135,216,0.22)` 底），点击联动右栏（右栏收起则先展开；否则回退内联卡，见 4.3）
- `skipHtml` 安全策略不变

### 5.4 基础组件 `primitives/`

- `Button`：胶囊（18px 圆角）、primary/ghost/outline/toolbar 变体、md/sm/icon 尺寸
- `Input` / `Textarea`：DeepSeek 输入框样式（聚焦品牌蓝边）
- `Modal`：自研（遮罩 + 居中卡片 + 关闭按钮 + Esc/遮罩关闭），替代 antd Modal/Popconfirm
- `Menu`：自研轻量下拉菜单（替代 antd Dropdown）
- `Tooltip`：自研轻量 tooltip
- `Toast`：自研（替代 antd message），成功/错误/警告变体
- `DisclosureRow`：折叠行（替代 antd Collapse，用于来源卡/错误详情）
- `Pill`：标签胶囊
- `Icon`：内联 SVG 图标集（send/stop/plus/more/settings/collapse/delete/copy/chevron/refresh/arrow-up 等）

### 5.5 `LoginPage` / `RegisterPage`

- 居中卡片 + 品牌蓝 logo + DeepSeek 风格表单（自研 Input）
- 保留现有校验规则与 remember 逻辑

## 6. 功能保持不变的部分

- 全部功能逻辑：路由、store、hooks、API 客户端、mock、SSE 消费、乐观更新、重试、来源跳转
- 全部 API 契约与类型
- i18n key 结构（新增 key 仅用于 UI 细节文案，遵循 i18n 规范）
- 现有测试行为：功能断言不变；仅类名/结构相关断言按新 markup 微调

## 7. 验证方式

1. `cd frontend && bun run lint && bun run format && bunx tsc --noEmit && bun run test` 全绿
2. 浏览器人工确认：视觉符合 DeepSeek-Harness 风格，明暗主题切换正常
3. 功能回归：登录/注册 → 新建会话 → 发送提问 → 流式回答 → 来源引用 → 删除会话
4. 文档同步：AGENT.md / ARCHITECTURE.md 中前端样式约定（antd 依赖、Tailwind 规则）随本次变更同步更新

## 8. 不在本轮范围

- 响应式 / 移动端适配（沿用桌面优先）
- 与 deepseek-harness 运行时/插槽/会话架构的整合
- 新的功能组件（文件上传、Markdown 编辑器等）
- 多语言新增（沿用现有 zh-CN / en-US）
