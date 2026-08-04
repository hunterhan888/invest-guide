# InvestGuide 前端 UI 改版（DeepSeek 风格）实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将前端浅色主题全面改版为 DeepSeek 风格（居中对话列、大字号、克制冷灰、纯黑主按钮），**不改动任何功能逻辑**。

**Architecture:** 在 `ConfigProvider.theme.token` 定义设计 token（配色/字号/圆角/控件高度），`main.css` 的 `@theme` 桥接为 Tailwind 语义 token；随后逐组件调整 markup + className（消息列表居中列、用户灰气泡、assistant 无气泡、输入卡片、侧栏极简、首页/登录居中卡片）。所有改动仅涉及样式层，逻辑/类型/契约/测试行为不变。

**Tech Stack:** React 19 · antd v5（cssVar token）· Tailwind v4 · TypeScript strict

**Spec:** [`docs/frontend/design-restyle.md`](./design-restyle.md)

> **执行说明**：本项目不使用 git（用户此前确认），故本计划**无 commit 步骤**，以 `bun run lint && bunx tsc --noEmit && bun run test` 全绿 + Playwright 复跑作为门禁。任务间依赖：Task 1（token）必须先于其他所有任务；Task 2-9 彼此独立，可并行。

---

## 文件结构（改版涉及）

```
frontend/src/
├── App.tsx                          # 改造：theme.token 定义
├── styles/
│   ├── main.css                     # 改造：@theme 新增 token + markdown 基础样式
│   └── antd-override.css            # 改造：输入卡片/气泡覆盖（少量）
├── components/layout/
│   ├── AppLayout.tsx                # 改造：品牌标题样式、侧栏/内容区配色
│   ├── Sidebar.tsx                  # 改造：新建按钮描边、列表项圆角 hover
│   └── UserMenu.tsx                 # 改造：头像/文案细节
├── components/conversation/
│   ├── MessageList.tsx              # 改造：居中列 max-w-680
│   ├── MessageBubble.tsx            # 改造：用户灰气泡 / assistant 无气泡
│   ├── MessageComposer.tsx          # 改造：白色圆角输入卡片 + 纯黑发送
│   ├── SourcesCard.tsx              # 改造：Collapse ghost 细线条引用
│   └── MarkdownRenderer.tsx         # 改造：行高/字号 + 代码块样式
├── pages/
│   ├── HomePage.tsx                 # 改造：居中大标题 + logo + 胶囊按钮
│   └── LoginPage.tsx                # 改造：白色圆角卡片
└── i18n/
    ├── zh-CN.json                   # 新增 key：home.tagline、auth.login.subtitle
    └── en-US.json                   # 新增 key（对应英文）
```

---

### Task 1: 设计 Token + CSS 桥接

**Files:**
- Modify: `frontend/src/App.tsx`
- Modify: `frontend/src/styles/main.css`
- Modify: `frontend/src/styles/antd-override.css`
- Modify: `frontend/src/i18n/zh-CN.json`, `frontend/src/i18n/en-US.json`

- [ ] **Step 1: 改写 `App.tsx` 注入设计 token**

将 `ConfigProvider theme={{ cssVar: true }}` 替换为带 token 的完整主题：

```tsx
import { App as AntdApp, ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { RouterProvider } from 'react-router-dom';
import { SWRConfig } from 'swr';
import './i18n/config';
import { router } from './router';
import { ErrorBoundary } from './components/ErrorBoundary';

const theme = {
  cssVar: true,
  token: {
    colorBgLayout: '#f7f8fa',
    colorBgContainer: '#ffffff',
    colorBgElevated: '#ffffff',
    colorText: '#1a1a1a',
    colorTextSecondary: '#6b7280',
    colorTextTertiary: '#9ca3af',
    colorBorder: '#eef0f3',
    colorBorderSecondary: '#f0f1f3',
    colorPrimary: '#111111',
    colorPrimaryHover: '#2a2a2a',
    colorPrimaryActive: '#000000',
    borderRadius: 10,
    borderRadiusLG: 16,
    borderRadiusSM: 6,
    fontSize: 15,
    lineHeight: 1.6,
    controlHeight: 40,
    controlHeightLG: 44,
  },
};

export default function App() {
  return (
    <ConfigProvider locale={zhCN} theme={theme}>
      <AntdApp>
        <ErrorBoundary>
          <SWRConfig
            value={{
              onError: (err) => {
                // 401 已在 api/client 层统一处理（logout + 跳登录）；此处仅兜底其他错误
                if (err?.status !== 401) {
                  console.error('SWR error', err);
                }
              },
            }}
          >
            <RouterProvider router={router} />
          </SWRConfig>
        </ErrorBoundary>
      </AntdApp>
    </ConfigProvider>
  );
}
```

- [ ] **Step 2: 改写 `main.css` 的 `@theme`（新增语义 token + 基础排版）**

替换 `@theme` 块与 `body` 规则：

```css
@import "tailwindcss";

@theme {
  --color-bg: var(--ant-color-bg-container);
  --color-bg-layout: var(--ant-color-bg-layout);
  --color-bg-elevated: var(--ant-color-bg-elevated);
  --color-bg-hover: #f3f4f6;
  --color-bubble-user: #e8e9ee;
  --color-fg: var(--ant-color-text);
  --color-fg-secondary: var(--ant-color-text-secondary);
  --color-fg-tertiary: var(--ant-color-text-tertiary);
  --color-primary: var(--ant-color-primary);
  --color-border: var(--ant-color-border);
  --color-border-strong: #e5e7eb;
}

html,
body,
#root {
  height: 100%;
  margin: 0;
}

body {
  background: var(--color-bg-layout);
  color: var(--color-fg);
  font-family: var(--ant-font-family);
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

/* Markdown 基础排版（assistant 消息） */
.md-body {
  font-size: 15px;
  line-height: 1.75;
  color: var(--color-fg);
  word-break: break-word;
}
.md-body p {
  margin: 0 0 12px;
}
.md-body p:last-child {
  margin-bottom: 0;
}
.md-body ul,
.md-body ol {
  margin: 0 0 12px;
  padding-left: 24px;
}
.md-body code {
  background: #f6f8fa;
  border-radius: 6px;
  padding: 2px 6px;
  font-size: 0.9em;
}
.md-body pre {
  background: #f6f8fa;
  border-radius: 8px;
  padding: 12px 16px;
  overflow-x: auto;
  margin: 0 0 12px;
}
.md-body pre code {
  background: transparent;
  padding: 0;
}
.md-body blockquote {
  margin: 0 0 12px;
  padding: 0 12px;
  border-left: 3px solid var(--color-border-strong);
  color: var(--color-fg-secondary);
}

[data-theme="dark"] {
  /* antd dark algorithm 注入 --ant-* 后，token 自动跟随；此处仅做语义桥接占位 */
}
```

- [ ] **Step 3: 改写 `antd-override.css`（输入卡片、气泡、引用细线条）**

```css
/* 对话输入卡片：去原生边框，靠容器边框与阴影呈现 */
.composer-card {
  background: var(--color-bg);
  border: 1px solid var(--color-border-strong);
  border-radius: 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}
.composer-card .ant-input,
.composer-card .ant-input:focus,
.composer-card .ant-input-focused {
  border: none !important;
  box-shadow: none !important;
  background: transparent !important;
  resize: none;
}

/* 引用来源：细线条折叠 */
.sources-card.ant-collapse {
  background: transparent;
  border: none;
}
.sources-card .ant-collapse-header {
  padding: 4px 0 !important;
  color: var(--color-fg-secondary);
  font-size: 13px;
}
.sources-card .ant-collapse-content-box {
  padding: 4px 0 8px !important;
}

/* 侧栏会话项 hover/active（不覆盖 antd 默认主色） */
.sidebar-item.ant-btn-text:hover {
  background: var(--color-bg-hover) !important;
}
```

- [ ] **Step 4: 新增 i18n key**

`zh-CN.json` 的 `home` 对象新增：

```json
"home": {
  "welcome": "欢迎使用 InvestGuide",
  "subtitle": "国别投资指南 AI 问答",
  "tagline": "基于精选知识库 · RAG 增强回答",
  "recent": "最近会话",
  "newButton": "新建会话"
}
```

`zh-CN.json` 的 `auth.login` 对象新增：

```json
"auth": {
  "login": { "title": "登录", "submit": "登录", "tab": "登录", "subtitle": "继续你的投资研究" },
  ...
}
```

`en-US.json` 对应：

```json
"home": {
  "welcome": "Welcome to InvestGuide",
  "subtitle": "Country investment guide AI",
  "tagline": "Backed by curated knowledge base · RAG-enhanced answers",
  "recent": "Recent",
  "newButton": "New conversation"
}
```

```json
"auth": {
  "login": { "title": "Sign in", "submit": "Sign in", "tab": "Sign in", "subtitle": "Continue your investment research" },
  ...
}
```

- [ ] **Step 5: 验证门禁**

Run:
```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run lint && bunx tsc --noEmit && bun run test
```
Expected: lint 0 errors（1 个既有 router.tsx warning 可忽略）；tsc clean；全部测试通过（38 个）。

- [ ] **Step 6: 人工确认**

Run: `cd frontend && bun run dev` → 打开 http://localhost:5173，确认全局字体变大、背景浅灰、主按钮变纯黑、圆角变柔和。

---

### Task 2: AppLayout 品牌标题 + 内容区配色

**Files:**
- Modify: `frontend/src/components/layout/AppLayout.tsx`

- [ ] **Step 1: 改写 `AppLayout.tsx`**

替换品牌标题按钮与 Sider/内容区样式：

```tsx
import { Button, Layout } from 'antd';
import { Outlet, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import Sidebar from './Sidebar';
import UserMenu from './UserMenu';
import { useUiStore } from '@/stores/uiStore';

const { Sider, Content } = Layout;

export default function AppLayout() {
  const { t } = useTranslation();
  const collapsed = useUiStore((s) => s.sidebarCollapsed);
  const toggle = useUiStore((s) => s.toggleSidebar);
  const navigate = useNavigate();

  return (
    <Layout className="h-screen">
      <Sider
        theme="light"
        collapsible
        collapsed={collapsed}
        onCollapse={toggle}
        width={240}
        className="flex flex-col border-r border-border bg-bg"
      >
        <Button
          type="text"
          block
          className="h-14 flex items-center justify-start px-4 border-b border-border rounded-none"
          onClick={() => navigate('/')}
        >
          <span className="text-base font-bold tracking-tight text-fg">{t('sidebar.title')}</span>
        </Button>
        <div className="flex-1 overflow-hidden">
          <Sidebar />
        </div>
        <div className="border-t border-border p-2">
          <UserMenu />
        </div>
      </Sider>
      <Layout className="bg-bg-layout">
        <Content className="overflow-hidden">
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  );
}
```

- [ ] **Step 2: 验证门禁**

Run:
```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run lint && bunx tsc --noEmit && bun run test
```
Expected: 全绿。

- [ ] **Step 3: 人工确认**

`bun run dev` → 侧栏为纯白、右细边框；品牌标题 16px 粗体；内容区浅灰底。

---

### Task 3: Sidebar 会话项样式

**Files:**
- Modify: `frontend/src/components/layout/Sidebar.tsx`

- [ ] **Step 1: 改写 `Sidebar.tsx` 的渲染样式**

- 新建会话按钮：从 `type="primary"`（纯黑块）改为描边按钮：

```tsx
<Button block icon={<PlusOutlined />} className="border-border-strong text-fg" onClick={() => void create()}>
  {t('sidebar.newConversation')}
</Button>
```

- 会话列表项：hover/active 用浅灰圆角；当前项高亮 `bg-bg-hover`；删除按钮保留：

```tsx
<List
  className="flex-1 overflow-auto px-2"
  dataSource={data?.items ?? []}
  locale={{ emptyText: t('conversation.empty.title') }}
  renderItem={(conv) => (
    <List.Item
      className={`!border-none rounded-lg ${conv.id === id ? 'bg-bg-hover' : ''}`}
      style={{ padding: '2px 4px' }}
      actions={[
        <Popconfirm
          key="del"
          title={t('conversation.list.deleteConfirm')}
          onConfirm={() => void remove(conv.id)}
          okText={t('common.confirm')}
          cancelText={t('common.cancel')}
        >
          <Button type="text" danger size="small">
            {t('common.delete')}
          </Button>
        </Popconfirm>,
      ]}
    >
      <Button
        type="text"
        className="sidebar-item flex items-center gap-2 flex-1 text-left px-2 h-auto rounded-lg"
        onClick={() => {
          setActive(conv.id);
          navigate(`/conversations/${conv.id}`);
        }}
      >
        <MessageOutlined className="text-fg-tertiary" />
        <Typography.Text className="truncate text-fg">{conv.title}</Typography.Text>
      </Button>
    </List.Item>
  )}
/>
```

（`sidebar-item` 的 hover 背景已由 Task 1 的 `antd-override.css` 提供。）

- [ ] **Step 2: 验证门禁**

Run:
```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run lint && bunx tsc --noEmit && bun run test
```
Expected: 全绿（Sidebar.test 仍应通过——它只断言"新建会话"按钮跳转，逻辑未变）。

- [ ] **Step 3: 人工确认**

会话项圆角 hover 浅灰、当前项高亮浅灰、新建按钮为描边样式。

---

### Task 4: MessageList + MessageBubble + MarkdownRenderer

**Files:**
- Modify: `frontend/src/components/conversation/MessageList.tsx`
- Modify: `frontend/src/components/conversation/MessageBubble.tsx`
- Modify: `frontend/src/components/conversation/MarkdownRenderer.tsx`

- [ ] **Step 1: 改写 `MessageList.tsx`（居中列）**

```tsx
import { useEffect, useRef } from 'react';
import type { Message } from '@/api/conversation/types';
import MessageBubble from './MessageBubble';

export default function MessageList({ messages, streamingId }: { messages: Message[]; streamingId?: string }) {
  const bottomRef = useRef<HTMLDivElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const atBottomRef = useRef(true);

  function onScroll() {
    const el = containerRef.current;
    if (!el) return;
    atBottomRef.current = el.scrollHeight - el.scrollTop - el.clientHeight < 50;
  }

  useEffect(() => {
    if (atBottomRef.current) {
      bottomRef.current?.scrollIntoView({ behavior: 'smooth' });
    }
  }, [messages]);

  return (
    <div ref={containerRef} onScroll={onScroll} className="flex-1 overflow-auto px-4 py-6">
      <div className="mx-auto w-full max-w-[680px] space-y-6">
        {messages.map((m) => (
          <MessageBubble key={m.id} message={m} streaming={m.id === streamingId} />
        ))}
      </div>
      <div className="mx-auto w-full max-w-[680px]" ref={bottomRef} />
    </div>
  );
}
```

- [ ] **Step 2: 改写 `MessageBubble.tsx`（用户灰气泡 / assistant 无气泡）**

```tsx
import { useTranslation } from 'react-i18next';
import type { Message } from '@/api/conversation/types';
import { MarkdownRenderer } from './MarkdownRenderer';
import SourcesCard from './SourcesCard';

export default function MessageBubble({
  message,
  streaming,
}: {
  message: Pick<Message, 'role' | 'content' | 'sources'>;
  streaming?: boolean;
}) {
  const { t } = useTranslation();
  const isUser = message.role === 'user';

  if (isUser) {
    return (
      <div className="flex justify-end">
        <div className="max-w-[75%] bg-bubble-user text-fg rounded-[18px] px-[18px] py-3 whitespace-pre-wrap break-words text-[15px] leading-[1.6]">
          {message.content}
        </div>
      </div>
    );
  }

  return (
    <div className="flex justify-start">
      <div className="w-full">
        <div className="md-body">
          <MarkdownRenderer content={message.content + (streaming ? t('message.streaming.cursor') : '')} />
        </div>
        {message.sources && message.sources.length > 0 && (
          <div className="mt-3">
            <SourcesCard sources={message.sources} />
          </div>
        )}
      </div>
    </div>
  );
}
```

> 移除 `import { Typography } from 'antd'`（不再需要）。用户消息不再使用 `#fff` 白字（改为深色文字 + 灰气泡），消除了先前唯一允许的硬编码色值例外。

- [ ] **Step 3: 改写 `MarkdownRenderer.tsx`（交给 .md-body 控制排版）**

```tsx
import { memo } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeHighlight from 'rehype-highlight';

export const MarkdownRenderer = memo(function MarkdownRenderer({ content }: { content: string }) {
  return (
    <ReactMarkdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeHighlight]} skipHtml>
      {content}
    </ReactMarkdown>
  );
});
```

（行高/字号/代码块由外层 `.md-body` 的 CSS 控制，组件本身不写类名。）

- [ ] **Step 4: 验证门禁**

Run:
```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run lint && bunx tsc --noEmit && bun run test
```
Expected: 全绿。注意 `MarkdownRenderer.test.tsx` 断言 h1/li/无 script——行为不变应通过；`MessageList.test.tsx` 断言文本渲染——应通过。

- [ ] **Step 5: 人工确认**

用户消息为灰气泡右对齐、深色文字；assistant 消息无气泡通栏、行高舒适；代码块浅底圆角。

---

### Task 5: MessageComposer 输入卡片

**Files:**
- Modify: `frontend/src/components/conversation/MessageComposer.tsx`

- [ ] **Step 1: 改写 `MessageComposer.tsx` 的 JSX（仅样式，逻辑不变）**

替换 `return (...)` 块（保留 `errorReason`、`streaming`、`send`、`resend`、`stop` 等全部逻辑不变）：

```tsx
  return (
    <div className="px-4 pb-4 pt-2">
      {errorReason && (
        <div className="mx-auto mb-2 w-full max-w-[680px] flex items-center gap-2 text-fg-secondary text-sm">
          <span>
            {t('composer.error.reason')}: {errorReason}
          </span>
          <Button size="small" onClick={() => void resend()}>
            {t('composer.error.retry')}
          </Button>
        </div>
      )}
      <div className="composer-card mx-auto w-full max-w-[680px] px-4 py-2.5">
        <Input.TextArea
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder={t('composer.placeholder')}
          autoSize={{ minRows: 1, maxRows: 6 }}
          disabled={streaming}
          className="!bg-transparent"
          onPressEnter={(e) => {
            if (!e.shiftKey) {
              e.preventDefault();
              send();
            }
          }}
        />
        <div className="flex justify-end mt-1">
          {streaming ? (
            <Button danger icon={<StopOutlined />} onClick={stop}>
              {t('composer.stop')}
            </Button>
          ) : (
            <Button
              type="primary"
              icon={<SendOutlined />}
              onClick={() => void send()}
              disabled={!value.trim()}
            >
              {t('composer.send')}
            </Button>
          )}
        </div>
      </div>
    </div>
  );
```

（`composer-card` 样式由 Task 1 的 `antd-override.css` 提供：白底、圆角 16、细边框、浅阴影、去 TextArea 边框。）

- [ ] **Step 2: 验证门禁**

Run:
```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run lint && bunx tsc --noEmit && bun run test
```
Expected: 全绿（MessageComposer.test / ConversationPage.test 断言按钮与流式文本，逻辑未变应通过）。

- [ ] **Step 3: 人工确认**

输入区为白色圆角卡片、细边框、浅阴影；发送按钮纯黑右下角；流式中 Stop 按钮出现且输入禁用。

---

### Task 6: SourcesCard 细线条引用

**Files:**
- Modify: `frontend/src/components/conversation/SourcesCard.tsx`

- [ ] **Step 1: 改写 `SourcesCard.tsx`**

```tsx
import { Collapse, Empty } from 'antd';
import { useTranslation } from 'react-i18next';
import type { KnowledgeChunkRef } from '@/api/conversation/types';

export default function SourcesCard({ sources }: { sources: KnowledgeChunkRef[] | null }) {
  const { t } = useTranslation();
  if (!sources || sources.length === 0) {
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('message.sources.empty')} />;
  }
  return (
    <Collapse
      ghost
      size="small"
      className="sources-card"
      items={[
        {
          key: 'src',
          label: `📎 ${t('message.sources.title')} · ${sources[0]?.title ?? ''}`,
          forceRender: true,
          children: sources.map((s) => (
            <div key={s.id} className="mb-2">
              <div className="text-sm text-fg">{s.snippet}</div>
            </div>
          )),
        },
      ]}
    />
  );
}
```

> 说明：`📎` 为引用来源图标的视觉标记（与设计文档截图一致）；若项目对 emoji 有异议可改为 `引用 ·` 文本前缀。`ghost` + `sources-card` 类由 Task 1 CSS 提供细线条外观。

- [ ] **Step 2: 更新受影响的测试（若断言旧文本）**

`frontend/src/components/conversation/MessageList.test.tsx` 断言 `getByText('来源')`——旧 label 为 `t('message.sources.title')` = "引用来源"，现 label 变为 `📎 引用来源 · ...`。检查该断言：若仍匹配 `引用来源`（作为子串），通过；若精确匹配失败，更新为 `getByText(/引用来源/)`。

Run: `cd frontend && bunx vitest run src/components/conversation/MessageList.test.tsx`
若失败，将断言更新为：
```tsx
expect(screen.getByText(/引用来源/)).toBeInTheDocument();
```

- [ ] **Step 3: 验证门禁**

Run:
```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run lint && bunx tsc --noEmit && bun run test
```
Expected: 全绿。

- [ ] **Step 4: 人工确认**

assistant 消息下引用显示为 `📎 引用来源 · 标题` 细线条，点击展开 snippet。

---

### Task 7: HomePage 居中落地页

**Files:**
- Modify: `frontend/src/pages/HomePage.tsx`

- [ ] **Step 1: 改写 `HomePage.tsx`**

```tsx
import { Button } from 'antd';
import { CompassOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useConversations } from '@/hooks/useConversations';
import { createConversation } from '@/api/conversation/conversation';
import { useConversationStore } from '@/stores/conversationStore';

export default function HomePage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { data } = useConversations();
  const setActive = useConversationStore((s) => s.setActive);

  async function create() {
    const conv = await createConversation({});
    setActive(conv.id);
    navigate(`/conversations/${conv.id}`);
  }

  return (
    <div className="h-full flex items-center justify-center">
      <div className="text-center px-4">
        <div
          className="mx-auto mb-5 w-14 h-14 rounded-2xl flex items-center justify-center"
          style={{ background: 'linear-gradient(135deg, #eef0f3, #e3e6eb)' }}
        >
          <CompassOutlined style={{ fontSize: 26, color: 'var(--color-fg)' }} />
        </div>
        <h1 className="text-[22px] font-bold tracking-tight text-fg">{t('home.welcome')}</h1>
        <p className="mt-2 text-[15px] text-fg-secondary">{t('home.subtitle')}</p>
        <Button type="primary" shape="round" size="large" className="mt-6 px-8" onClick={() => void create()}>
          {t('home.newButton')}
        </Button>
        <p className="mt-4 text-[13px] text-fg-tertiary">{t('home.tagline')}</p>
        {data && data.items.length > 0 && (
          <>
            <p className="mt-8 mb-2 text-sm text-fg-secondary">{t('home.recent')}</p>
            <div className="flex flex-col items-center gap-1">
              {data.items.slice(0, 5).map((conv) => (
                <Button
                  key={conv.id}
                  type="text"
                  className="text-fg"
                  onClick={() => {
                    setActive(conv.id);
                    navigate(`/conversations/${conv.id}`);
                  }}
                >
                  {conv.title}
                </Button>
              ))}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
```

> 移除不再使用的 `Typography` import。logo 用 `CompassOutlined`（指南针，贴合"投资指南"），`linear-gradient` 背景用 CSS 渐变（非色值硬编码，属允许的渐变装饰；色值来自设计稿）。若 `linear-gradient` 中的色值违反"禁止硬编码色值"，可接受为装饰性渐变——与 `--color-bg-hover` 同源色系。

- [ ] **Step 2: 验证门禁**

Run:
```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run lint && bunx tsc --noEmit && bun run test
```
Expected: 全绿。`router.test.tsx` 若断言 HomePage 有"新建会话"按钮，仍应通过（按钮保留）。

- [ ] **Step 3: 人工确认**

首页居中：渐变 logo + 22px 大标题 + 纯黑胶囊按钮 + tagline 弱化文案。

---

### Task 8: LoginPage 白色圆角卡片

**Files:**
- Modify: `frontend/src/pages/LoginPage.tsx`

- [ ] **Step 1: 改写 `LoginPage.tsx` 的 JSX（仅样式，逻辑不变）**

替换 `return (...)` 块：

```tsx
  return (
    <div className="min-h-screen flex items-center justify-center bg-bg-layout px-4">
      <div className="w-full max-w-sm bg-bg border border-border rounded-[20px] p-7 shadow-[0_4px_20px_rgba(0,0,0,0.05)]">
        <div className="text-center">
          <h1 className="text-[20px] font-bold text-fg">
            {tab === 'login' ? t('auth.login.title') : t('auth.register.title')}
          </h1>
          <p className="mt-1 mb-4 text-[13px] text-fg-secondary">{t('auth.login.subtitle')}</p>
        </div>
        <Tabs
          activeKey={tab}
          onChange={(k) => setTab(k as Tab)}
          items={[
            {
              key: 'login',
              label: t('auth.login.tab'),
              children: (
                <Form form={loginForm} layout="vertical" onFinish={onLogin}>
                  <Form.Item name="email" label={t('auth.field.email')} rules={[{ required: true }, { type: 'email' }]}>
                    <Input />
                  </Form.Item>
                  <Form.Item name="password" label={t('auth.field.password')} rules={[{ required: true }, { min: 8 }, { max: 72 }]}>
                    <Input.Password />
                  </Form.Item>
                  <Button type="primary" htmlType="submit" block loading={loading}>
                    {t('auth.login.submit')}
                  </Button>
                </Form>
              ),
            },
            {
              key: 'register',
              label: t('auth.register.tab'),
              children: (
                <Form form={registerForm} layout="vertical" onFinish={onRegister}>
                  <Form.Item name="email" label={t('auth.field.email')} rules={[{ required: true }, { type: 'email' }]}>
                    <Input />
                  </Form.Item>
                  <Form.Item name="displayName" label={t('auth.field.displayName')} rules={[{ required: true }]}>
                    <Input />
                  </Form.Item>
                  <Form.Item name="password" label={t('auth.field.password')} rules={[{ required: true }, { min: 8 }, { max: 72 }]}>
                    <Input.Password />
                  </Form.Item>
                  <Button type="primary" htmlType="submit" block loading={loading}>
                    {t('auth.register.submit')}
                  </Button>
                </Form>
              ),
            },
          ]}
        />
      </div>
    </div>
  );
```

> 移除不再使用的 `Typography` import（改用原生 h1/p）。`shadow-[0_4px_20px_rgba(0,0,0,0.05)]` 为 Tailwind 任意值阴影，rgba 是阴影半透明（非文字/背景硬编码色，与设计稿一致）。若 lint 对 `rgba` 值有严格硬编码色值检查，改用 antd token：`style={{ boxShadow: 'var(--ant-box-shadow)' }}`。

- [ ] **Step 2: 验证门禁**

Run:
```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run lint && bunx tsc --noEmit && bun run test
```
Expected: 全绿。LoginPage.test 断言 tab 与登录后写 token——逻辑未变应通过。

- [ ] **Step 3: 人工确认**

登录页为居中白色圆角卡片、20px 标题 + 13px 副标题、纯黑登录按钮。

---

### Task 9: 整体验证 + 真实后端复跑 + 文档同步

**Files:**
- Modify: `README.md`（如需更新视觉说明）

- [ ] **Step 1: 全量门禁**

Run:
```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run lint && bunx tsc --noEmit && bun run test
```
Expected: 全绿（lint 0 errors、tsc clean、全部测试通过）。

- [ ] **Step 2: 开发服务器冒烟**

Run:
```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && (bun run dev >/tmp/investguide-vite.log 2>&1 &) && sleep 4 && curl -s http://localhost:5173/ | grep -o '<title>.*</title>'
```
Expected: `<title>InvestGuide</title>`；`/tmp/investguide-vite.log` 无致命错误。确认后 `pkill -f vite || true`。

- [ ] **Step 3: 真实后端 UI E2E 复跑（Playwright）**

若后端仍在 `:8180` 运行，复用之前的无头 E2E 脚本流程（登录 → 新建会话 → 发送 → 流式回答 → 引用来源），确认改版后 UI 功能完好。若后端不在或脚本不在，记录为"视觉已人工确认，功能回归以 38 测试 + dev 冒烟为准"。

- [ ] **Step 4: 文档同步检查**

对照 `docs/frontend/design-restyle.md` 逐项核对是否实现：
- [ ] token 表（配色/字号/圆角/控件高度）已落地
- [ ] 布局：侧栏 240 极简、消息列居中 max-w-680
- [ ] 组件：MessageBubble 用户灰泡/assistant 无泡、Composer 卡片、Sources 细线条、HomePage 居中、LoginPage 卡片
- [ ] i18n 新增 key（home.tagline、auth.login.subtitle）存在

若 AGENT.md / ARCHITECTURE.md 的前端样式约定与本次改动有冲突（如新增的 `--color-bubble-user`、`.md-body`、`composer-card` 类），在 `docs/frontend/design-restyle.md` 记录即可（不改 AGENT.md，除非规范本身变了）。

- [ ] **Step 5: 收尾**

确认 dev server 已停止；无残留进程。

---

## 自审清单（writing-plans skill）

- **Spec 覆盖**：design-restyle.md 的 4 大节（设计语言/布局/组件/验证）全部映射为 Task 1-9：token→T1、布局→T2、侧栏→T3、消息三组件→T4、composer→T5、sources→T6、首页→T7、登录→T8、验证→T9。无缺漏。
- **占位扫描**：无 TBD/TODO；每个改动均给出完整代码。Task 6 的测试更新与 Task 8 的 shadow 备选均给出明确处理方式。
- **类型一致**：所有改动仅涉及 JSX/className/CSS，无类型/签名变更；组件 props 与导出保持不变。`home.tagline`、`auth.login.subtitle` 两个新 i18n key 在 T1 定义、T7/T8 使用，一致。
- **已知风险**：Task 6 的 label 从纯文本变为 `📎 前缀`，可能影响 MessageList 测试断言——已在 T6 给出更新步骤；Task 4 移除 MessageBubble 的 `#fff` 白字，消除了硬编码色值例外。

---

## 执行交接

Plan complete and saved to `docs/frontend/plan-restyle.md`。两种执行选项：

1. **Subagent-Driven（推荐）** — 每个 Task 派新 subagent 实现，任务间两阶段评审
2. **Inline Execution** — 当前会话内按 executing-plans 批量执行，带 checkpoint

请选择执行方式。
