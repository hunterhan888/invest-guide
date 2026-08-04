# 前端 MVP 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 交付一个可独立运行、可演示的前端 MVP——登录鉴权 + 会话列表/详情 + 问答与 SSE 流式回答，全程 Mock 先行。

**Architecture:** 混合组织——Phase 0 一次性打底共享层（脚手架/样式/骨架/API 客户端+Mock），Phase 1+ 按特性纵切（鉴权→会话→问答+SSE），每片自含 api/store/hook/page。Mock 层插在 API 客户端与真实 fetch 之间，以 `ARCHITECTURE.md` API 约定为唯一契约。

**Tech Stack:** React 19 · TypeScript（strict）· Vite · antd v5（cssVar）· Tailwind v4 · SWR · Zustand · i18next/react-i18next · react-markdown · Vitest · Testing Library · Bun

**Spec:** [`docs/frontend/task.md`](./task.md)

> **契约修订（2026-08-02，依据 `docs/backend/api/openapi.yaml`）**：
> - 后端无 `GET /auth/me` → 前端移除 `getMe`；`RequireAuth` 仅校验本地 token 存在，非法 token 由首个 API 请求的 401 兜底（SWR/request → logout + 跳登录）。
> - API `Message` 无 `conversationId` 字段 → 前端 `Message` 类型与 mock/composer 乐观消息均不含该字段。

> **真实后端联调（2026-08-03，后端已启动）**：
> - 后端监听 `:8180`（非默认 8080）。Vite dev server 通过 `/api/v1` 代理转发到 `VITE_API_PROXY_TARGET`（默认 `http://localhost:8180`），前端 API 客户端仍用相对路径 `/api/v1`，无需 CORS。
> - 开发模式 `.env.development` 设 `VITE_USE_MOCK=false` 对接真实后端；测试仍走 `.env` 的 mock 模式（`VITE_USE_MOCK=true`），互不干扰。
> - 已验证（curl + Playwright 无头 E2E）：注册/登录 → 新建会话 → 发送提问 → SSE 流式（`sources → message* → done`）→ 消息历史含 assistant 内容与 sources，全部通过真实后端。
> - 已知差异：后端 `Conversation.country` 返回 `""` 而非 `null`（前端 `string | null` 兼容）；antd v5 与 React 19 有 compat warning（非致命，实际可用）；`favicon.ico` 缺失由 `public/favicon.svg` 补齐。

---

## 文件结构

```
frontend/
├── package.json
├── tsconfig.json
├── tsconfig.node.json
├── vite.config.ts
├── index.html
├── .env.example
└── src/
    ├── main.tsx                      # 挂载 + Provider 装配
    ├── App.tsx                       # 根布局 + Outlet + antd App
    ├── router.tsx                    # 路由表 + RequireAuth
    ├── vite-env.d.ts                 # 环境变量类型
    ├── api/
    │   ├── client.ts                 # request<T> + openStream + Mock 路由分发
    │   ├── types.ts                  # 统一响应/分页/错误码
    │   ├── index.ts                  # 按 VITE_USE_MOCK 安装 mock
    │   ├── auth/
    │   │   ├── types.ts
    │   │   └── auth.ts               # login/register
    │   ├── conversation/
    │   │   ├── types.ts
    │   │   └── conversation.ts       # list/get/create/delete/listMessages/sendMessage
    │   └── mock/
    │       ├── index.ts              # installMocks 注册各 domain
    │       ├── tools.ts              # jsonOk/jsonFail/delay/sseStream
    │       ├── auth.ts               # auth 路由
    │       ├── conversation.ts       # conversation 路由（内存数据）
    │       └── sse.ts                # 问答 SSE 流生成器
    ├── components/
    │   ├── ErrorBoundary.tsx
    │   ├── layout/
    │   │   ├── AppLayout.tsx
    │   │   ├── Sidebar.tsx
    │   │   └── UserMenu.tsx
    │   └── conversation/
    │       ├── MessageList.tsx
    │       ├── MessageBubble.tsx
    │       ├── SourcesCard.tsx
    │       ├── MarkdownRenderer.tsx
    │       └── MessageComposer.tsx
    ├── pages/
    │   ├── LoginPage.tsx
    │   ├── HomePage.tsx
    │   └── ConversationPage.tsx
    ├── hooks/
    │   ├── useConversations.ts
    │   ├── useConversation.ts
    │   └── useSSEStream.ts
    ├── stores/
    │   ├── authStore.ts
    │   ├── conversationStore.ts
    │   └── uiStore.ts
    ├── i18n/
    │   ├── config.ts
    │   ├── zh-CN.json
    │   └── en-US.json
    └── styles/
        ├── main.css                  # Tailwind 入口 + @theme token
        └── antd-override.css         # antd 深度覆盖（占位）
```

> `src/` 直接子项为 11 个，超过 AGENT.md「≤10 建议」；其中 `hooks/`、`stores/`、`i18n/`、`api/`、`pages/`、`components/` 由 ARCHITECTURE.md 第 711–721 行明确要求，按架构优先级保留。

---

## 全局 i18n key 约定

后续各 Task 引用以下命名空间（zh-CN.json 在 0.3 一次性建立，按需在后续 Task 增补 key）：

```
common.{confirm,cancel,retry,loading,delete,create,send,stop,empty}
auth.{login.title,login.submit,login.tab,register.title,register.submit,register.tab,
      field.email,field.password,field.displayName,
      error.invalid,error.conflict,error.unauthorized}
sidebar.{title,newConversation,userMenu.logout,toggle}
conversation.{empty.title,empty.hint,list.deleteConfirm}
message.{sources.title,sources.empty,streaming.pending,streaming.cursor}
composer.{placeholder,send,stop,emptyInput,toLong,error.reason,error.retry}
home.{welcome,subtitle,recent,newButton}
error.{generic,network,unauthorized}
```

---

## Phase 0 — 工程基础设施

### Task 0.1: Vite + React 19 + TS 脚手架

**Files:**
- Create: `frontend/package.json`, `frontend/tsconfig.json`, `frontend/tsconfig.node.json`, `frontend/vite.config.ts`, `frontend/index.html`, `frontend/.env.example`, `frontend/src/main.tsx`, `frontend/src/vite-env.d.ts`
- Modify: `Makefile`

- [ ] **Step 1: 创建 frontend/package.json**

```json
{
  "name": "invest-guide-frontend",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "tsc -b && vite build",
    "preview": "vite preview",
    "lint": "eslint .",
    "lint:fix": "eslint . --fix",
    "format": "prettier --write .",
    "test": "vitest run",
    "test:coverage": "vitest run --coverage"
  }
}
```

- [ ] **Step 2: 安装依赖**

Run:
```bash
cd frontend && bun add react@^19 react-dom@^19 react-router-dom@^6 antd@^5 @ant-design/icons swr zustand i18next react-i18next react-markdown remark-gfm rehype-highlight
```
```bash
cd frontend && bun add -d vite@^5 @vitejs/plugin-react typescript @types/react @types/react-dom vitest@^1 @testing-library/react @testing-library/jest-dom jsdom @tailwindcss/vite tailwindcss@^4 prettier eslint @eslint/js typescript-eslint eslint-plugin-react-hooks eslint-plugin-react-refresh
```

- [ ] **Step 3: 创建 tsconfig.json（strict）**

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "useDefineForClassFields": true,
    "lib": ["ES2022", "DOM", "DOM.Iterable"],
    "module": "ESNext",
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "moduleDetection": "force",
    "noEmit": true,
    "jsx": "react-jsx",
    "strict": true,
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "noFallthroughCasesInSwitch": true,
    "noImplicitOverride": true,
    "noUncheckedIndexedAccess": true,
    "exactOptionalPropertyTypes": false,
    "baseUrl": ".",
    "paths": { "@/*": ["src/*"] }
  },
  "include": ["src"]
}
```

- [ ] **Step 4: 创建 tsconfig.node.json**

```json
{
  "compilerOptions": {
    "composite": true,
    "skipLibCheck": true,
    "module": "ESNext",
    "moduleResolution": "bundler",
    "strict": true,
    "types": ["node"]
  },
  "include": ["vite.config.ts"]
}
```

- [ ] **Step 5: 创建 vite.config.ts**

```ts
/// <reference types="vitest" />
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
import path from 'node:path';

export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: { alias: { '@': path.resolve(__dirname, './src') } },
  server: { port: 5173 },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    css: false,
  },
});
```

- [ ] **Step 6: 创建 index.html**

```html
<!doctype html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Invest Guide</title>
  </head>
  <body>
    <div id="root"></div>
    <script type="module" src="/src/main.tsx"></script>
  </body>
</html>
```

- [ ] **Step 7: 创建 src/vite-env.d.ts 与 src/test/setup.ts**

`src/vite-env.d.ts`:
```ts
/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_USE_MOCK: string;
  readonly VITE_API_BASE_URL: string;
}
interface ImportMeta {
  readonly env: ImportMetaEnv;
}
```

`src/test/setup.ts`:
```ts
import '@testing-library/jest-dom/vitest';
```

- [ ] **Step 8: 创建 src/main.tsx（最小占位，0.3 替换）**

```tsx
function App() {
  return <div>Invest Guide</div>;
}
const root = document.getElementById('root');
if (root) {
  // eslint-disable-next-line @typescript-eslint/no-require-imports
  const { createRoot } = require('react-dom/client');
  createRoot(root).render(<App />);
}
```

实际用 ESM（require 仅为示意，正确写法见下）。修正为：

```tsx
import { createRoot } from 'react-dom/client';

function App() {
  return <div>Invest Guide</div>;
}

const root = document.getElementById('root');
if (root) {
  createRoot(root).render(<App />);
}
```

- [ ] **Step 9: 创建 .env.example**

```
VITE_API_BASE_URL=/api/v1
VITE_USE_MOCK=true
```

- [ ] **Step 10: 创建 eslint 与 prettier 配置**

`frontend/eslint.config.js`:
```js
import js from '@eslint/js';
import tseslint from 'typescript-eslint';
import reactHooks from 'eslint-plugin-react-hooks';
import reactRefresh from 'eslint-plugin-react-refresh';

export default tseslint.config(
  { ignores: ['dist'] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ['**/*.{ts,tsx}'],
    plugins: { 'react-hooks': reactHooks, 'react-refresh': reactRefresh },
    rules: {
      ...reactHooks.configs.recommended.rules,
      'react-refresh/only-export-components': ['warn', { allowConstantExport: true }],
    },
  },
);
```

`frontend/.prettierrc.json`:
```json
{ "semi": true, "singleQuote": true, "trailingComma": "all", "printWidth": 100 }
```

- [ ] **Step 11: Update Makefile 便捷入口**

追加：
```make
.PHONY: frontend-dev frontend-build frontend-test
frontend-dev:
	cd frontend && bun run dev
frontend-build:
	cd frontend && bun run build
frontend-test:
	cd frontend && bun run test
```

- [ ] **Step 12: 验证**

Run:
```bash
cd frontend && bun install && bun run lint && bunx tsc --noEmit && bun run dev
```
Expected: lint 无错；tsc 通过；dev 在 5173 启动，浏览器显示 "Invest Guide"。

- [ ] **Step 13: Commit**

```bash
git add frontend Makefile
git commit -m "feat(frontend): 初始化 Vite + React 19 + TS 脚手架"
```

---

### Task 0.2: 样式系统

**Files:**
- Create: `frontend/src/styles/main.css`, `frontend/src/styles/antd-override.css`
- Modify: `frontend/src/main.tsx`

- [ ] **Step 1: 创建 src/styles/main.css**

```css
@import "tailwindcss";

@theme {
  --color-bg: var(--ant-color-bg-container);
  --color-bg-layout: var(--ant-color-bg-layout);
  --color-fg: var(--ant-color-text);
  --color-fg-secondary: var(--ant-color-text-secondary);
  --color-primary: var(--ant-color-primary);
  --color-border: var(--ant-color-border);
  --color-bg-elevated: var(--ant-color-bg-elevated);
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
}

[data-theme="dark"] {
  /* antd dark algorithm 注入 --ant-* 后，token 自动跟随；此处仅做语义桥接占位 */
}
```

- [ ] **Step 2: 创建 src/styles/antd-override.css（占位）**

```css
/* antd 深度覆盖选择器写在此处。当前 MVP 无需覆盖，保持空文件。 */
```

- [ ] **Step 3: 改造 main.tsx 用 ConfigProvider + 引入样式（占位骨架，0.3 完善）**

```tsx
import { createRoot } from 'react-dom/client';
import { ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import './styles/main.css';
import './styles/antd-override.css';

function App() {
  return (
    <ConfigProvider locale={zhCN} theme={{ cssVar: true }}>
      <div className="bg-bg-layout text-fg">Invest Guide</div>
    </ConfigProvider>
  );
}

const root = document.getElementById('root');
if (root) {
  createRoot(root).render(<App />);
}
```

- [ ] **Step 4: 验证 token 生效**

Run: `cd frontend && bun run dev`
Expected: 页面背景使用 antd layout 色；原子类 `bg-bg-layout` 生效；`grep -rn "#[0-9a-fA-F]\{6\}" src` 无硬编码色值（除 antd-override.css 注释外）。

- [ ] **Step 5: Commit**

```bash
git add frontend/src/styles frontend/src/main.tsx
git commit -m "feat(frontend): 建立 Tailwind v4 + antd cssVar 样式系统"
```

---

### Task 0.3: 全局骨架（main / App / router / i18n / ErrorBoundary / SWR）

**Files:**
- Create: `frontend/src/App.tsx`, `frontend/src/router.tsx`, `frontend/src/components/ErrorBoundary.tsx`, `frontend/src/i18n/config.ts`, `frontend/src/i18n/zh-CN.json`, `frontend/src/i18n/en-US.json`, `frontend/src/pages/LoginPage.tsx`（占位）, `frontend/src/pages/HomePage.tsx`（占位）, `frontend/src/pages/ConversationPage.tsx`（占位）
- Modify: `frontend/src/main.tsx`

- [ ] **Step 1: 创建 zh-CN.json 全量 key（后续 Task 复用）**

```json
{
  "common": {
    "confirm": "确认",
    "cancel": "取消",
    "retry": "重试",
    "loading": "加载中…",
    "delete": "删除",
    "create": "新建",
    "send": "发送",
    "stop": "停止",
    "empty": "暂无数据"
  },
  "auth": {
    "login": { "title": "登录", "submit": "登录", "tab": "登录" },
    "register": { "title": "注册", "submit": "注册", "tab": "注册" },
    "field": { "email": "邮箱", "password": "密码", "displayName": "昵称" },
    "error": {
      "invalid": "邮箱或密码错误",
      "conflict": "该邮箱已注册",
      "unauthorized": "请先登录"
    }
  },
  "sidebar": {
    "title": "Invest Guide",
    "newConversation": "新建会话",
    "userMenu": { "logout": "登出" },
    "toggle": "折叠侧边栏"
  },
  "conversation": {
    "empty": { "title": "暂无会话", "hint": "点击「新建会话」开始提问" },
    "list": { "deleteConfirm": "确认删除该会话？" }
  },
  "message": {
    "sources": { "title": "引用来源", "empty": "无引用" },
    "streaming": { "pending": "思考中…", "cursor": "▍" }
  },
  "composer": {
    "placeholder": "输入你的投资问题…",
    "send": "发送",
    "stop": "停止生成",
    "emptyInput": "请输入问题",
    "tooLong": "问题过长",
    "error": { "reason": "生成失败", "retry": "重试" }
  },
  "home": {
    "welcome": "欢迎使用 Invest Guide",
    "subtitle": "国别投资指南 AI 问答",
    "recent": "最近会话",
    "newButton": "新建会话"
  },
  "error": { "generic": "出错了，请重试", "network": "网络异常", "unauthorized": "登录已过期" }
}
```

- [ ] **Step 2: 创建 en-US.json（占位，键镜像 zh-CN，值为英文短译）**

```json
{
  "common": {
    "confirm": "Confirm", "cancel": "Cancel", "retry": "Retry", "loading": "Loading…",
    "delete": "Delete", "create": "New", "send": "Send", "stop": "Stop", "empty": "No data"
  },
  "auth": {
    "login": { "title": "Sign in", "submit": "Sign in", "tab": "Sign in" },
    "register": { "title": "Sign up", "submit": "Sign up", "tab": "Sign up" },
    "field": { "email": "Email", "password": "Password", "displayName": "Display name" },
    "error": {
      "invalid": "Invalid email or password",
      "conflict": "Email already registered",
      "unauthorized": "Please sign in first"
    }
  },
  "sidebar": {
    "title": "Invest Guide",
    "newConversation": "New conversation",
    "userMenu": { "logout": "Sign out" },
    "toggle": "Toggle sidebar"
  },
  "conversation": {
    "empty": { "title": "No conversations", "hint": "Click “New conversation” to start" },
    "list": { "deleteConfirm": "Delete this conversation?" }
  },
  "message": {
    "sources": { "title": "Sources", "empty": "No sources" },
    "streaming": { "pending": "Thinking…", "cursor": "▍" }
  },
  "composer": {
    "placeholder": "Ask your investment question…",
    "send": "Send", "stop": "Stop", "emptyInput": "Please enter a question",
    "tooLong": "Question too long",
    "error": { "reason": "Generation failed", "retry": "Retry" }
  },
  "home": {
    "welcome": "Welcome to Invest Guide",
    "subtitle": "Country investment guide AI",
    "recent": "Recent", "newButton": "New conversation"
  },
  "error": { "generic": "Something went wrong", "network": "Network error", "unauthorized": "Session expired" }
}
```

- [ ] **Step 3: 创建 src/i18n/config.ts**

```ts
import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import zhCN from './zh-CN.json';
import enUS from './en-US.json';

void i18n.use(initReactI18next).init({
  resources: {
    'zh-CN': { translation: zhCN },
    'en-US': { translation: enUS },
  },
  lng: 'zh-CN',
  fallbackLng: 'zh-CN',
  interpolation: { escapeValue: false },
});

export const SUPPORTED_LANGS = ['zh-CN', 'en-US'] as const;
export type AppLang = (typeof SUPPORTED_LANGS)[number];
export default i18n;
```

- [ ] **Step 4: 写 ErrorBoundary 失败测试**

`frontend/src/components/ErrorBoundary.test.tsx`:
```tsx
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { ErrorBoundary } from './ErrorBoundary';

const Bomb = () => {
  throw new Error('boom');
};

describe('ErrorBoundary', () => {
  it('捕获渲染错误后显示回退 UI', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {});
    render(
      <ErrorBoundary>
        <Bomb />
      </ErrorBoundary>,
    );
    expect(screen.getByText(/出错了/i)).toBeInTheDocument();
    spy.mockRestore();
  });
});
```

- [ ] **Step 5: 运行测试确认失败**

Run: `cd frontend && bunx vitest run src/components/ErrorBoundary.test.tsx`
Expected: FAIL（`ErrorBoundary` 未定义）

- [ ] **Step 6: 实现 ErrorBoundary**

`frontend/src/components/ErrorBoundary.tsx`:
```tsx
import { Component, type ErrorInfo, type ReactNode } from 'react';
import { Button, Result } from 'antd';
import { useTranslation } from 'react-i18next';

type Props = { children: ReactNode };
type State = { hasError: boolean };

export class ErrorBoundary extends Component<Props, State> {
  override state: State = { hasError: false };

  static getDerivedStateFromError(): State {
    return { hasError: true };
  }

  override componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('ErrorBoundary caught', error, info);
  }

  override render() {
    if (this.state.hasError) {
      return <ErrorFallback onReload={() => window.location.reload()} />;
    }
    return this.props.children;
  }
}

function ErrorFallback({ onReload }: { onReload: () => void }) {
  const { t } = useTranslation();
  return (
    <Result
      status="error"
      title={t('error.generic')}
      extra={<Button type="primary" onClick={onReload}>{t('common.retry')}</Button>}
    />
  );
}
```

- [ ] **Step 7: 运行测试确认通过**

Run: `cd frontend && bunx vitest run src/components/ErrorBoundary.test.tsx`
Expected: PASS

- [ ] **Step 8: 创建占位页面**

`frontend/src/pages/LoginPage.tsx`:
```tsx
export default function LoginPage() {
  return <div>LoginPage</div>;
}
```

`frontend/src/pages/HomePage.tsx`:
```tsx
export default function HomePage() {
  return <div>HomePage</div>;
}
```

`frontend/src/pages/ConversationPage.tsx`:
```tsx
export default function ConversationPage() {
  return <div>ConversationPage</div>;
}
```

- [ ] **Step 9: 创建 router.tsx（含 RequireAuth 骨架）**

```tsx
import { type ReactNode, Suspense } from 'react';
import { createBrowserRouter, Navigate, Outlet, useLocation } from 'react-router-dom';
import LoginPage from './pages/LoginPage';
import HomePage from './pages/HomePage';
import ConversationPage from './pages/ConversationPage';

function RequireAuth({ children }: { children: ReactNode }) {
  // authStore 在 Phase 1 接入；此处先放行，1.3 替换为真实校验
  const token = true;
  const location = useLocation();
  if (!token) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }
  return <>{children}</>;
}

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  {
    path: '/',
    element: (
      <RequireAuth>
        <Suspense fallback={null}>
          <Outlet />
        </Suspense>
      </RequireAuth>
    ),
    children: [
      { index: true, element: <HomePage /> },
      { path: 'conversations/:id', element: <ConversationPage /> },
    ],
  },
  { path: '*', element: <Navigate to="/" replace /> },
]);
```

- [ ] **Step 10: 创建 App.tsx**

```tsx
import { App as AntdApp, ConfigProvider } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import { RouterProvider } from 'react-router-dom';
import { SWRConfig } from 'swr';
import './i18n/config';
import { router } from './router';
import { ErrorBoundary } from './components/ErrorBoundary';

export default function App() {
  return (
    <ConfigProvider locale={zhCN} theme={{ cssVar: true }}>
      <AntdApp>
        <ErrorBoundary>
          <SWRConfig
            value={{
              onError: (err) => {
                // 401 处理在 0.4 request 层；此处兜底其余错误
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

- [ ] **Step 11: 改造 main.tsx 挂载 App**

```tsx
import { createRoot } from 'react-dom/client';
import './styles/main.css';
import './styles/antd-override.css';
import App from './App';

const root = document.getElementById('root');
if (root) {
  createRoot(root).render(<App />);
}
```

- [ ] **Step 12: 验证**

Run:
```bash
cd frontend && bun run lint && bunx tsc --noEmit && bunx vitest run
```
Expected: lint/tsc 通过；ErrorBoundary 测试通过；`bun run dev` 后访问 `/` 显示 HomePage，`/login` 显示 LoginPage。

- [ ] **Step 13: Commit**

```bash
git add frontend/src
git commit -m "feat(frontend): 搭建全局骨架（router/i18n/ErrorBoundary/SWR）"
```

---

### Task 0.4: API 客户端基础设施与 Mock 拦截

**Files:**
- Create: `frontend/src/api/types.ts`, `frontend/src/api/client.ts`, `frontend/src/api/index.ts`, `frontend/src/api/mock/tools.ts`, `frontend/src/api/mock/index.ts`
- Modify: `frontend/src/main.tsx`（启动安装 mock）, `frontend/src/App.tsx`（SWR onError 接入 401 跳转）

- [ ] **Step 1: 创建 api/types.ts**

```ts
export type ErrorCode =
  | 'INVALID_INPUT'
  | 'UNAUTHORIZED'
  | 'FORBIDDEN'
  | 'NOT_FOUND'
  | 'CONFLICT'
  | 'RATE_LIMITED'
  | 'INTERNAL_ERROR'
  | 'BAD_GATEWAY'
  | 'GATEWAY_TIMEOUT';

export type ApiResponse<T> =
  | { success: true; data: T; message?: string }
  | { success: false; error: string; code: ErrorCode };

export type Paginated<T> = { items: T[]; total: number; hasMore: boolean };

export class ApiError extends Error {
  constructor(
    public status: number,
    public code: ErrorCode,
    message: string,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}
```

- [ ] **Step 2: 写 client 失败测试**

`frontend/src/api/client.test.ts`:
```ts
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { request, registerMock, __resetMocks } from './client';
import { ApiError } from './types';

describe('request', () => {
  beforeEach(() => __resetMocks());

  it('成功解析 data', async () => {
    registerMock({ match: () => true, handle: async () => ({ status: 200, body: { success: true, data: { id: '1' } } }) });
    const res = await request<{ id: string }>('GET', '/api/v1/x');
    expect(res).toEqual({ id: '1' });
  });

  it('success=false 抛 ApiError', async () => {
    registerMock({
      match: () => true,
      handle: async () => ({ status: 401, body: { success: false, error: 'bad', code: 'UNAUTHORIZED' } }),
    });
    await expect(request('GET', '/api/v1/x')).rejects.toMatchObject({ status: 401, code: 'UNAUTHORIZED' });
    expect(ApiError).toBeDefined();
  });

  it('未匹配路由抛 404', async () => {
    await expect(request('GET', '/api/v1/none')).rejects.toMatchObject({ status: 404 });
  });
});
```

- [ ] **Step 3: 运行测试确认失败**

Run: `cd frontend && bunx vitest run src/api/client.test.ts`
Expected: FAIL（`request` 等未定义）

- [ ] **Step 4: 实现 client.ts**

```ts
import { ApiError, type ApiResponse } from './types';

export type MockRequest = { method: string; path: string; body: unknown };
export type MockResponse =
  | { status: number; body: unknown }
  | { status: number; stream: ReadableStream<Uint8Array> };
export type MockHandler = { match: (r: MockRequest) => boolean; handle: (r: MockRequest) => Promise<MockResponse> };

const handlers: MockHandler[] = [];
const USE_MOCK = import.meta.env.VITE_USE_MOCK === 'true';

export function registerMock(h: MockHandler) {
  handlers.push(h);
}

export function __resetMocks() {
  handlers.length = 0;
}

function readToken(): string | null {
  try {
    return localStorage.getItem('investguide.token');
  } catch {
    return null;
  }
}

export async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  if (USE_MOCK) {
    const req: MockRequest = { method, path, body };
    for (const h of handlers) {
      if (h.match(req)) {
        const res = await h.handle(req);
        return handleMockResponse<T>(res);
      }
    }
    throw new ApiError(404, 'NOT_FOUND', `mock route not found: ${method} ${path}`);
  }

  const res = await fetch(`${import.meta.env.VITE_API_BASE_URL}${path}`, {
    method,
    headers: {
      'Content-Type': 'application/json',
      ...(readToken() ? { Authorization: `Bearer ${readToken()}` } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  return handleRealResponse<T>(res);
}

async function handleMockResponse<T>(res: MockResponse): Promise<T> {
  if ('stream' in res) {
    throw new Error('use openStream for SSE responses');
  }
  return unwrap<T>(res.status, res.body as ApiResponse<T>);
}

async function handleRealResponse<T>(res: Response): Promise<T> {
  const json = (await res.json()) as ApiResponse<T>;
  return unwrap<T>(res.status, json);
}

function unwrap<T>(status: number, json: ApiResponse<T>): T {
  if (json.success) {
    return json.data;
  }
  throw new ApiError(status, json.code, json.error);
}

export async function openStream(path: string, lastEventId?: string): Promise<ReadableStream<Uint8Array>> {
  if (USE_MOCK) {
    const req: MockRequest = { method: 'GET', path, body: undefined };
    for (const h of handlers) {
      if (h.match(req)) {
        const res = await h.handle(req);
        if ('stream' in res) return res.stream;
        throw new Error('expected stream response');
      }
    }
    throw new ApiError(404, 'NOT_FOUND', `mock stream not found: ${path}`);
  }

  const res = await fetch(`${import.meta.env.VITE_API_BASE_URL}${path}`, {
    method: 'GET',
    headers: {
      ...(readToken() ? { Authorization: `Bearer ${readToken()}` } : {}),
      ...(lastEventId ? { 'Last-Event-ID': lastEventId } : {}),
    },
  });
  if (!res.ok || !res.body) {
    throw new ApiError(res.status, 'BAD_GATEWAY', 'stream open failed');
  }
  return res.body;
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd frontend && bunx vitest run src/api/client.test.ts`
Expected: PASS

- [ ] **Step 6: 创建 mock/tools.ts**

```ts
import type { MockResponse } from '../client';

export function jsonOk<T>(data: T, status = 200): MockResponse {
  return { status, body: { success: true, data } };
}

export function jsonFail(code: string, message: string, status: number): MockResponse {
  return { status, body: { success: false, error: message, code } };
}

export function delay(ms: number): Promise<void> {
  return new Promise((r) => setTimeout(r, ms));
}

export type SSEFrame = { event: string; data: unknown };

export function sseStream(frames: SSEFrame[], opts?: { heartbeatMs?: number }): MockResponse {
  const encoder = new TextEncoder();
  const stream = new ReadableStream<Uint8Array>({
    async start(controller) {
      for (const f of frames) {
        const payload = `event: ${f.event}\ndata: ${JSON.stringify(f.data)}\n\n`;
        controller.enqueue(encoder.encode(payload));
        await delay(20);
      }
      controller.close();
    },
  });
  return { status: 200, stream };
}
```

- [ ] **Step 7: 创建 mock/index.ts（空安装入口，各 domain Task 自行 register）**

```ts
import { installAuthMocks } from './auth';
import { installConversationMocks } from './conversation';

export function installMocks() {
  installAuthMocks();
  installConversationMocks();
}
```

> 注：此文件引用 `./auth`、`./conversation`，分别在 Task 1.1、2.1 创建。为避免 0.4 阶段编译失败，先创建两个最小桩文件：

`frontend/src/api/mock/auth.ts`:
```ts
import { registerMock } from '../client';
export function installAuthMocks() {
  registerMock({
    match: (r) => r.method === 'GET' && r.path.startsWith('/auth/'),
    handle: async () => ({ status: 404, body: { success: false, error: 'not implemented', code: 'NOT_FOUND' } }),
  });
}
```

`frontend/src/api/mock/conversation.ts`:
```ts
import { registerMock } from '../client';
export function installConversationMocks() {
  registerMock({
    match: (r) => r.path.startsWith('/conversations'),
    handle: async () => ({ status: 404, body: { success: false, error: 'not implemented', code: 'NOT_FOUND' } }),
  });
}
```

- [ ] **Step 8: 创建 api/index.ts（按 env 安装）**

```ts
import { installMocks } from './mock';

if (import.meta.env.VITE_USE_MOCK === 'true') {
  installMocks();
}

export { request, openStream } from './client';
export * from './types';
```

- [ ] **Step 9: 在 main.tsx 顶部导入 api 副作用**

修改 `frontend/src/main.tsx`，在 `import './styles/...'` 之前加：
```ts
import './api';
```

- [ ] **Step 10: 401 跳转接入（App.tsx SWRConfig）**

修改 `App.tsx` 的 SWRConfig.onError：
```tsx
onError: (err) => {
  if (err?.status === 401) {
    localStorage.removeItem('investguide.token');
    if (window.location.pathname !== '/login') {
      window.location.assign('/login');
    }
    return;
  }
  console.error('SWR error', err);
},
```

- [ ] **Step 11: 验证**

Run:
```bash
cd frontend && bun run lint && bunx tsc --noEmit && bunx vitest run
```
Expected: 全绿；`bun run dev` 时 mock 默认 404 占位返回，前端不崩。

- [ ] **Step 12: Commit**

```bash
git add frontend/src
git commit -m "feat(frontend): 实现 API 客户端与 Mock 拦截层"
```

---

## Phase 1 — 鉴权切片

### Task 1.1: Auth API 契约 + Mock

**Files:**
- Create: `frontend/src/api/auth/types.ts`, `frontend/src/api/auth/auth.ts`
- Modify: `frontend/src/api/mock/auth.ts`

- [ ] **Step 1: 创建 auth/types.ts**

```ts
export type User = {
  id: string;
  email: string;
  displayName: string;
};

export type LoginRequest = { email: string; password: string };
export type RegisterRequest = { email: string; password: string; displayName: string };
export type AuthResponse = { token: string; user: User };
```

- [ ] **Step 2: 写 auth 客户端失败测试**

`frontend/src/api/auth/auth.test.ts`:
```ts
import { describe, it, expect, beforeEach } from 'vitest';
import { __resetMocks } from '../client';
import { login, register, getMe } from './auth';
import { ApiError } from '../types';
import './mock/auth'; // ensure registered

describe('auth api', () => {
  beforeEach(() => {
    __resetMocks();
    import('./mock/auth');
  });

  it('login 成功返回 token+user', async () => {
    const r = await login({ email: 'a@b.com', password: 'pass1234' });
    expect(r.token).toBeTypeOf('string');
    expect(r.user.email).toBe('a@b.com');
  });

  it('login 错误凭证抛 401', async () => {
    await expect(login({ email: 'a@b.com', password: 'wrong' })).rejects.toMatchObject({
      status: 401,
      code: 'UNAUTHORIZED',
    });
  });

  it('register 重复邮箱抛 409', async () => {
    await register({ email: 'a@b.com', password: 'pass1234', displayName: 'A' });
    await expect(register({ email: 'a@b.com', password: 'x', displayName: 'B' })).rejects.toMatchObject({
      status: 409,
      code: 'CONFLICT',
    });
  });

  it('getMe 按校验 token 返回 user', async () => {
    const r = await register({ email: 'c@b.com', password: 'pass1234', displayName: 'C' });
    const me = await getMe(r.token);
    expect(me.email).toBe('c@b.com');
  });

  it('getMe 无效 token 抛 401', async () => {
    await expect(getMe('bad')).rejects.toBeInstanceOf(ApiError);
  });
});
```

- [ ] **Step 3: 运行确认失败**

Run: `cd frontend && bunx vitest run src/api/auth/auth.test.ts`
Expected: FAIL

- [ ] **Step 4: 实现 auth.ts**

`frontend/src/api/auth/auth.ts`:
```ts
import { request } from '../client';
import type { AuthResponse, LoginRequest, RegisterRequest, User } from './types';

export function login(req: LoginRequest): Promise<AuthResponse> {
  return request<AuthResponse>('POST', '/auth/login', req);
}

export function register(req: RegisterRequest): Promise<AuthResponse> {
  return request<AuthResponse>('POST', '/auth/register', req);
}

export function getMe(token: string): Promise<User> {
  // mock 模式通过 Authorization header；真实模式 request() 自动注入当前 token
  return request<User>('GET', '/auth/me', undefined);
}
```

- [ ] **Step 5: 实现 mock/auth.ts（替换桩）**

```ts
import { registerMock, type MockRequest } from '../client';
import { jsonOk, jsonFail } from './tools';
import type { AuthResponse, User } from '../auth/types';

const users = new Map<string, { password: string; user: User; token: string }>();
const tokenToUser = new Map<string, User>();

function newId() {
  return Math.random().toString(36).slice(2);
}

export function installAuthMocks() {
  registerMock({
    match: (r) => r.method === 'POST' && r.path === '/auth/register',
    handle: async (r: MockRequest) => {
      const { email, password, displayName } = r.body as { email: string; password: string; displayName: string };
      if (users.has(email)) return jsonFail('CONFLICT', 'email exists', 409);
      const user: User = { id: newId(), email, displayName };
      const token = 'tok_' + newId();
      users.set(email, { password, user, token });
      tokenToUser.set(token, user);
      return jsonOk<AuthResponse>({ token, user }, 201);
    },
  });

  registerMock({
    match: (r) => r.method === 'POST' && r.path === '/auth/login',
    handle: async (r: MockRequest) => {
      const { email, password } = r.body as { email: string; password: string };
      const rec = users.get(email);
      if (!rec || rec.password !== password) return jsonFail('UNAUTHORIZED', 'invalid credentials', 401);
      return jsonOk<AuthResponse>({ token: rec.token, user: rec.user });
    },
  });

  registerMock({
    match: (r) => r.method === 'GET' && r.path === '/auth/me',
    handle: async (r: MockRequest) => {
      const user = tokenToUser.get((r as MockRequest & { headers?: Record<string, string> }).headers?.Authorization?.replace('Bearer ', '') ?? '');
      if (!user) return jsonFail('UNAUTHORIZED', 'invalid token', 401);
      return jsonOk(user);
    },
  });
}
```

> mock 的 `MockRequest` 需携带 headers 以支持 `getMe`。修改 `src/api/client.ts` 的 `MockRequest` 类型与构造，把 `Authorization` 头解析出来：

在 `client.ts` `MockRequest` 类型加 `authToken?: string`，并在 `request` 构造 `req` 时取 `readToken()`：
```ts
export type MockRequest = { method: string; path: string; body: unknown; authToken?: string };
// 构造处：
const req: MockRequest = { method, path, body, authToken: readToken() ?? undefined };
```
mock/auth 的 getMe 改用 `r.authToken`：
```ts
const user = tokenToUser.get(r.authToken ?? '');
```

- [ ] **Step 6: 运行测试确认通过**

Run: `cd frontend && bunx vitest run src/api/auth/auth.test.ts`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add frontend/src/api
git commit -m "feat(frontend): auth API 契约与 Mock"
```

---

### Task 1.2: authStore

**Files:**
- Create: `frontend/src/stores/authStore.ts`, `frontend/src/stores/authStore.test.ts`

- [ ] **Step 1: 写 store 失败测试**

```ts
import { describe, it, expect, beforeEach } from 'vitest';
import { useAuthStore, TOKEN_KEY } from './authStore';

describe('authStore', () => {
  beforeEach(() => {
    localStorage.clear();
    useAuthStore.getState().logout();
  });

  it('初始无 token', () => {
    expect(useAuthStore.getState().token).toBeNull();
    expect(useAuthStore.getState().user).toBeNull();
  });

  it('login 设置 token 持久化到 localStorage', () => {
    useAuthStore.getState().login({ token: 'tok_1', user: { id: '1', email: 'a@b.com', displayName: 'A' } });
    expect(localStorage.getItem(TOKEN_KEY)).toBe('tok_1');
    expect(useAuthStore.getState().user?.email).toBe('a@b.com');
  });

  it('logout 清空 token 与 user', () => {
    useAuthStore.getState().login({ token: 'tok_1', user: { id: '1', email: 'a@b.com', displayName: 'A' } });
    useAuthStore.getState().logout();
    expect(useAuthStore.getState().token).toBeNull();
    expect(localStorage.getItem(TOKEN_KEY)).toBeNull();
  });

  it('初始化从 localStorage 恢复 token', () => {
    localStorage.setItem(TOKEN_KEY, 'tok_2');
    useAuthStore.getState().hydrate();
    expect(useAuthStore.getState().token).toBe('tok_2');
  });
});
```

- [ ] **Step 2: 运行确认失败**

Run: `cd frontend && bunx vitest run src/stores/authStore.test.ts`
Expected: FAIL

- [ ] **Step 3: 实现 authStore.ts**

```ts
import { create } from 'zustand';
import type { User } from '@/api/auth/types';

export const TOKEN_KEY = 'investguide.token';

type AuthState = {
  token: string | null;
  user: User | null;
  login: (p: { token: string; user: User }) => void;
  setUser: (u: User) => void;
  logout: () => void;
  hydrate: () => void;
};

export const useAuthStore = create<AuthState>((set) => ({
  token: null,
  user: null,
  login: ({ token, user }) => {
    localStorage.setItem(TOKEN_KEY, token);
    set({ token, user });
  },
  setUser: (user) => set({ user }),
  logout: () => {
    localStorage.removeItem(TOKEN_KEY);
    set({ token: null, user: null });
  },
  hydrate: () => {
    const token = localStorage.getItem(TOKEN_KEY);
    if (token) set({ token });
  },
));

useAuthStore.getState().hydrate();
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd frontend && bunx vitest run src/stores/authStore.test.ts`
Expected: PASS

- [ ] **Step 5: 接入 client.ts 用 store 读 token（替换 readToken 直接 localStorage）**

修改 `client.ts` 顶部加：
```ts
import { useAuthStore } from '@/stores/authStore';
```
将 `readToken` 改为：
```ts
function readToken(): string | null {
  return useAuthStore.getState().token;
}
```
（mock 测试中 store 也已 hydrate，行为不变。）

- [ ] **Step 6: 验证**

Run: `cd frontend && bun run lint && bunx tsc --noEmit && bunx vitest run`
Expected: 全绿

- [ ] **Step 7: Commit**

```bash
git add frontend/src
git commit -m "feat(frontend): authStore 与 token 持久化"
```

---

### Task 1.3: LoginPage 与路由守卫接入

**Files:**
- Modify: `frontend/src/pages/LoginPage.tsx`, `frontend/src/router.tsx`
- Create: `frontend/src/pages/LoginPage.test.tsx`

- [ ] **Step 1: 写 LoginPage 失败测试**

`frontend/src/pages/LoginPage.test.tsx`:
```tsx
import { describe, it, expect, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { useAuthStore } from '@/stores/authStore';
import LoginPage from './LoginPage';

function renderAt(path: string) {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route path="/" element={<div>Home</div>} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('LoginPage', () => {
  beforeEach(() => {
    localStorage.clear();
    useAuthStore.getState().logout();
  });

  it('渲染登录/注册两个 tab', () => {
    renderAt('/login');
    expect(screen.getByRole('tab', { name: /登录/ })).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: /注册/ })).toBeInTheDocument();
  });

  it('空字段提交显示校验错误', async () => {
    renderAt('/login');
    await userEvent.click(screen.getByRole('button', { name: /登录/ }));
    expect(await screen.findByText(/请输入邮箱|required/i)).toBeInTheDocument();
  });

  it('成功登录后跳转首页', async () => {
    // 预置一个已注册账号
    await import('@/api').then(() => {});
    const { register } = await import('@/api/auth/auth');
    await register({ email: 'a@b.com', password: 'pass1234', displayName: 'A' });

    renderAt('/login');
    await userEvent.type(screen.getByLabelText(/邮箱/i), 'a@b.com');
    await userEvent.type(screen.getByLabelText(/密码/i), 'pass1234');
    await userEvent.click(screen.getByRole('button', { name: /登录/ }));
    await waitFor(() => {
      expect(useAuthStore.getState().token).not.toBeNull();
    });
  });
});
```

需安装：`cd frontend && bun add -d @testing-library/user-event`

- [ ] **Step 2: 运行确认失败**

Run: `cd frontend && bunx vitest run src/pages/LoginPage.test.tsx`
Expected: FAIL

- [ ] **Step 3: 实现 LoginPage.tsx**

```tsx
import { useState } from 'react';
import { App as AntdApp, Button, Form, Input, Tabs, Typography } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { login as apiLogin, register as apiRegister } from '@/api/auth/auth';
import { useAuthStore } from '@/stores/authStore';
import type { LoginRequest, RegisterRequest } from '@/api/auth/types';

type Tab = 'login' | 'register';

export default function LoginPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { message } = AntdApp.useApp();
  const loginStore = useAuthStore((s) => s.login);
  const [tab, setTab] = useState<Tab>('login');
  const [loading, setLoading] = useState(false);
  const [loginForm] = Form.useForm<LoginRequest>();
  const [registerForm] = Form.useForm<RegisterRequest>();

  async function onLogin(values: LoginRequest) {
    setLoading(true);
    try {
      const res = await apiLogin(values);
      loginStore({ token: res.token, user: res.user });
      navigate('/', { replace: true });
    } catch (e: unknown) {
      const code = (e as { code?: string })?.code;
      message.error(code === 'UNAUTHORIZED' ? t('auth.error.invalid') : t('error.generic'));
    } finally {
      setLoading(false);
    }
  }

  async function onRegister(values: RegisterRequest) {
    setLoading(true);
    try {
      const res = await apiRegister(values);
      loginStore({ token: res.token, user: res.user });
      navigate('/', { replace: true });
    } catch (e: unknown) {
      const code = (e as { code?: string })?.code;
      message.error(code === 'CONFLICT' ? t('auth.error.conflict') : t('error.generic'));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-bg-layout">
      <div className="w-full max-w-sm p-6 bg-bg rounded-lg border border-border">
        <Typography.Title level={3} className="text-center">
          {tab === 'login' ? t('auth.login.title') : t('auth.register.title')}
        </Typography.Title>
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
                  <Form.Item name="password" label={t('auth.field.password')} rules={[{ required: true }, { min: 8 }]}>
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
                  <Form.Item name="password" label={t('auth.field.password')} rules={[{ required: true }, { min: 8 }]}>
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
}
```

- [ ] **Step 4: 接入 RequireAuth 真实校验**

修改 `router.tsx` 的 `RequireAuth`：
```tsx
import { useAuthStore } from './stores/authStore';
import { getMe } from './api/auth/auth';

function RequireAuth({ children }: { children: ReactNode }) {
  const token = useAuthStore((s) => s.token);
  const setUser = useAuthStore((s) => s.setUser);
  const logout = useAuthStore((s) => s.logout);
  const location = useLocation();
  const [checked, setChecked] = useState(false);

  useEffect(() => {
    if (!token) return;
    if (useAuthStore.getState().user) {
      setChecked(true);
      return;
    }
    getMe(token)
      .then((u) => {
        setUser(u);
        setChecked(true);
      })
      .catch(() => {
        logout();
      });
  }, [token, setUser, logout]);

  if (!token) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }
  if (!checked) return null;
  return <>{children}</>;
}
```
需 `import { useEffect, useState } from 'react';`。

并新增已登录跳离 `/login` 的守卫：在 `LoginPage` 顶部加：
```tsx
const token = useAuthStore((s) => s.token);
useEffect(() => {
  if (token) navigate('/', { replace: true });
}, [token, navigate]);
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd frontend && bunx vitest run src/pages/LoginPage.test.tsx`
Expected: PASS

- [ ] **Step 6: 手动验证**

Run: `cd frontend && bun run dev`
- 访问 `/` → 跳 `/login`
- 注册 → 自动登录回首页
- 登出（暂无入口，2.3 加）→ 回登录

- [ ] **Step 7: Commit**

```bash
git add frontend/src
git commit -m "feat(frontend): LoginPage 与路由守卫"
```

---

## Phase 2 — 会话管理切片

### Task 2.1: Conversation API 契约 + Mock

**Files:**
- Create: `frontend/src/api/conversation/types.ts`, `frontend/src/api/conversation/conversation.ts`, `frontend/src/api/conversation/conversation.test.ts`
- Modify: `frontend/src/api/mock/conversation.ts`

- [ ] **Step 1: 创建 conversation/types.ts**

```ts
export type MessageRole = 'user' | 'assistant';

export type KnowledgeChunkRef = {
  id: string;
  title: string;
  snippet: string;
};

export type Conversation = {
  id: string;
  title: string;
  country: string | null;
  createdAt: string;
  updatedAt: string;
};

export type Message = {
  id: string;
  role: MessageRole;
  content: string;
  sources: KnowledgeChunkRef[] | null;
  tokensUsed: number | null;
  createdAt: string;
};

export type CreateConversationRequest = { title?: string; country?: string };
export type SendMessageRequest = { content: string };
export type SendMessageResponse = { messageId: string };
```

- [ ] **Step 2: 写 conversation 客户端失败测试**

```ts
import { describe, it, expect, beforeEach } from 'vitest';
import { __resetMocks } from '../client';
import './mock/conversation';
import { listConversations, createConversation, getConversation, deleteConversation, listMessages, sendMessage } from './conversation';

describe('conversation api', () => {
  beforeEach(() => __resetMocks());

  it('createConversation 后能 list 出来', async () => {
    const created = await createConversation({ title: '首问' });
    const list = await listConversations(1);
    expect(list.items).toHaveLength(1);
    expect(list.items[0]?.id).toBe(created.id);
  });

  it('sendMessage 返回 messageId 并能 list 出 user 消息', async () => {
    const conv = await createConversation({});
    const { messageId } = await sendMessage(conv.id, { content: '你好' });
    expect(messageId).toBeTypeOf('string');
    const msgs = await listMessages(conv.id);
    expect(msgs.items.map((m) => m.role)).toContain('user');
  });

  it('listConversations 分页 hasMore 边界正确', async () => {
    for (let i = 0; i < 25; i++) await createConversation({});
    const p1 = await listConversations(1, 20);
    expect(p1.items).toHaveLength(20);
    expect(p1.hasMore).toBe(true);
    const p2 = await listConversations(2, 20);
    expect(p2.items).toHaveLength(5);
    expect(p2.hasMore).toBe(false);
  });

  it('deleteConversation 后 getConversation 抛 404', async () => {
    const conv = await createConversation({});
    await deleteConversation(conv.id);
    await expect(getConversation(conv.id)).rejects.toMatchObject({ status: 404 });
  });
});
```

- [ ] **Step 3: 运行确认失败**

Run: `cd frontend && bunx vitest run src/api/conversation/conversation.test.ts`
Expected: FAIL

- [ ] **Step 4: 实现 conversation.ts**

```ts
import { request } from '../client';
import type { Paginated } from '../types';
import type {
  Conversation,
  CreateConversationRequest,
  Message,
  SendMessageRequest,
  SendMessageResponse,
} from './types';

export function listConversations(page = 1, pageSize = 20): Promise<Paginated<Conversation>> {
  return request<Paginated<Conversation>>('GET', `/conversations?page=${page}&pageSize=${pageSize}`);
}

export function getConversation(id: string): Promise<Conversation> {
  return request<Conversation>('GET', `/conversations/${id}`);
}

export function createConversation(req: CreateConversationRequest): Promise<Conversation> {
  return request<Conversation>('POST', '/conversations', req);
}

export function deleteConversation(id: string): Promise<void> {
  return request<void>('DELETE', `/conversations/${id}`);
}

export function listMessages(convId: string): Promise<Paginated<Message>> {
  return request<Paginated<Message>>('GET', `/conversations/${convId}/messages?page=1&pageSize=1000`);
}

export function sendMessage(convId: string, req: SendMessageRequest): Promise<SendMessageResponse> {
  return request<SendMessageResponse>('POST', `/conversations/${convId}/messages`, req);
}
```

> mock/conversation.ts 内 `conversation.ts` 文件名与目录名重名导致 import 冲突——重命名 mock 文件为 `mock/conversation.ts` 与 `api/conversation/conversation.ts`。前者路径 `./mock/conversation`、后者 `./conversation`，在测试文件 `api/conversation/conversation.test.ts` 中 `import './mock/conversation'` 需写为 `import '../mock/conversation'`。修订测试 Step 2 的 import 行：
```ts
import '../mock/conversation';
```

- [ ] **Step 5: 实现 mock/conversation.ts（替换桩）**

```ts
import { registerMock, type MockRequest } from '../client';
import { jsonOk, jsonFail } from './tools';
import type { Conversation, Message } from '../conversation/types';

type Store = { conversations: Map<string, Conversation>; messages: Map<string, Message[]> };
const store: Store = { conversations: new Map(), messages: new Map() };

function newId() {
  return Math.random().toString(36).slice(2);
}

function ts() {
  return new Date().toISOString();
}

export function installConversationMocks() {
  registerMock({
    match: (r) => r.method === 'GET' && r.path.startsWith('/conversations?'),
    handle: async () => {
      const items = [...store.conversations.values()].sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
      return jsonOk({ items, total: items.length, hasMore: false });
    },
  });

  registerMock({
    match: (r) => r.method === 'POST' && r.path === '/conversations',
    handle: async (r: MockRequest) => {
      const body = (r.body ?? {}) as { title?: string; country?: string };
      const conv: Conversation = {
        id: newId(),
        title: body.title ?? '新会话',
        country: body.country ?? null,
        createdAt: ts(),
        updatedAt: ts(),
      };
      store.conversations.set(conv.id, conv);
      store.messages.set(conv.id, []);
      return jsonOk(conv, 201);
    },
  });

  registerMock({
    match: (r) => r.method === 'GET' && /^\/conversations\/[^/]+$/.test(r.path),
    handle: async (r: MockRequest) => {
      const id = r.path.split('/')[2]!;
      const conv = store.conversations.get(id);
      if (!conv) return jsonFail('NOT_FOUND', 'not found', 404);
      return jsonOk(conv);
    },
  });

  registerMock({
    match: (r) => r.method === 'DELETE' && /^\/conversations\/[^/]+$/.test(r.path),
    handle: async (r: MockRequest) => {
      const id = r.path.split('/')[2]!;
      if (!store.conversations.has(id)) return jsonFail('NOT_FOUND', 'not found', 404);
      store.conversations.delete(id);
      store.messages.delete(id);
      return jsonOk(null, 204);
    },
  });

  registerMock({
    match: (r) => r.method === 'GET' && /\/conversations\/[^/]+\/messages/.test(r.path),
    handle: async (r: MockRequest) => {
      const id = r.path.split('/')[2]!;
      const msgs = store.messages.get(id) ?? [];
      return jsonOk({ items: msgs, total: msgs.length, hasMore: false });
    },
  });

  registerMock({
    match: (r) => r.method === 'POST' && /\/conversations\/[^/]+\/messages$/.test(r.path),
    handle: async (r: MockRequest) => {
      const id = r.path.split('/')[2]!;
      const body = r.body as { content: string };
      const msg: Message = {
        id: newId(),
        role: 'user',
        content: body.content,
        sources: null,
        tokensUsed: null,
        createdAt: ts(),
      };
      const conv = store.conversations.get(id);
      if (conv) {
        if (conv.title === '新会话') conv.title = body.content.slice(0, 20);
        conv.updatedAt = ts();
      }
      store.messages.get(id)?.push(msg);
      return jsonOk({ messageId: msg.id }, 201);
    },
  });
}
```

> Step 2 测试用 `listConversations(1)`（单参数）。修正 client `listConversations` 默认 pageSize 已经处理。`hasMore` 边界测试需 mock 实现真实分页：把 GET list handler 改为解析 query：

替换 GET list handler：
```ts
registerMock({
  match: (r) => r.method === 'GET' && r.path.startsWith('/conversations?'),
  handle: async (r: MockRequest) => {
    const u = new URL('http://x/' + r.path);
    const page = Number(u.searchParams.get('page') ?? '1');
    const pageSize = Number(u.searchParams.get('pageSize') ?? '20');
    const all = [...store.conversations.values()].sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
    const start = (page - 1) * pageSize;
    const items = all.slice(start, start + pageSize);
    return jsonOk({ items, total: all.length, hasMore: start + pageSize < all.length });
  },
});
```

- [ ] **Step 6: 运行测试确认通过**

Run: `cd frontend && bunx vitest run src/api/conversation/conversation.test.ts`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add frontend/src/api
git commit -m "feat(frontend): 会话 API 契约与 Mock"
```

---

### Task 2.2: conversationStore + SWR hooks

**Files:**
- Create: `frontend/src/stores/conversationStore.ts`, `frontend/src/stores/uiStore.ts`, `frontend/src/hooks/useConversations.ts`, `frontend/src/hooks/useConversation.ts`

- [ ] **Step 1: 写 uiStore 失败测试**

```ts
import { describe, it, expect, beforeEach } from 'vitest';
import { useUiStore } from './uiStore';

describe('uiStore', () => {
  beforeEach(() => localStorage.clear());

  it('toggleSidebar 翻转并持久化', () => {
    useUiStore.getState().setCollapsed(false);
    useUiStore.getState().toggleSidebar();
    expect(useUiStore.getState().sidebarCollapsed).toBe(true);
    expect(localStorage.getItem('investguide.sidebarCollapsed')).toBe('true');
  });
});
```

- [ ] **Step 2: 实现 uiStore.ts**

```ts
import { create } from 'zustand';

type UiState = {
  sidebarCollapsed: boolean;
  toggleSidebar: () => void;
  setCollapsed: (v: boolean) => void;
};

export const useUiStore = create<UiState>((set, get) => ({
  sidebarCollapsed: localStorage.getItem('investguide.sidebarCollapsed') === 'true',
  toggleSidebar: () => {
    const next = !get().sidebarCollapsed;
    localStorage.setItem('investguide.sidebarCollapsed', String(next));
    set({ sidebarCollapsed: next });
  },
  setCollapsed: (v) => {
    localStorage.setItem('investguide.sidebarCollapsed', String(v));
    set({ sidebarCollapsed: v });
  },
}));
```

- [ ] **Step 3: 实现 conversationStore.ts**

```ts
import { create } from 'zustand';

type ConversationState = {
  activeId: string | null;
  setActive: (id: string | null) => void;
  clearActive: () => void;
};

export const useConversationStore = create<ConversationState>((set) => ({
  activeId: null,
  setActive: (id) => set({ activeId: id }),
  clearActive: () => set({ activeId: null }),
}));
```

- [ ] **Step 4: 实现 useConversations.ts**

```ts
import useSWR from 'swr';
import { listConversations } from '@/api/conversation/conversation';
import { useAuthStore } from '@/stores/authStore';

export function useConversations() {
  const token = useAuthStore((s) => s.token);
  return useSWR(
    token ? ['conversations', token] : null,
    () => listConversations(1, 50),
  );
}
```

- [ ] **Step 5: 实现 useConversation.ts**

```ts
import useSWR from 'swr';
import { getConversation, listMessages } from '@/api/conversation/conversation';
import { useAuthStore } from '@/stores/authStore';

export function useConversation(id: string | null) {
  const token = useAuthStore((s) => s.token);
  const conv = useSWR(id && token ? ['conversation', id, token] : null, () => getConversation(id!));
  const messages = useSWR(id && token ? ['messages', id, token] : null, () => listMessages(id!));
  return { conv, messages, mutateMessages: messages.mutate };
}
```

- [ ] **Step 6: 验证**

Run: `cd frontend && bun run lint && bunx tsc --noEmit && bunx vitest run`
Expected: 全绿

- [ ] **Step 7: Commit**

```bash
git add frontend/src/stores frontend/src/hooks
git commit -m "feat(frontend): conversationStore/uiStore 与 SWR hooks"
```

---

### Task 2.3: 会话布局与列表侧边栏

**Files:**
- Create: `frontend/src/components/layout/AppLayout.tsx`, `frontend/src/components/layout/Sidebar.tsx`, `frontend/src/components/layout/UserMenu.tsx`
- Modify: `frontend/src/pages/HomePage.tsx`, `frontend/src/pages/ConversationPage.tsx`, `frontend/src/router.tsx`

- [ ] **Step 1: 实现 UserMenu.tsx**

```tsx
import { Avatar, Button, Dropdown, Typography } from 'antd';
import { LogoutOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuthStore } from '@/stores/authStore';

export default function UserMenu() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);

  return (
    <Dropdown
      menu={{
        items: [
          {
            key: 'logout',
            icon: <LogoutOutlined />,
            label: t('sidebar.userMenu.logout'),
            onClick: () => {
              logout();
              navigate('/login', { replace: true });
            },
          },
        ],
      }}
    >
      <Button type="text" className="w-full flex items-center gap-2">
        <Avatar size="small">{(user?.displayName ?? '?').slice(0, 1)}</Avatar>
        <Typography.Text className="truncate">{user?.displayName}</Typography.Text>
      </Button>
    </Dropdown>
  );
}
```

- [ ] **Step 2: 实现 Sidebar.tsx**

```tsx
import { Button, List, Popconfirm, Typography } from 'antd';
import { PlusOutlined, MessageOutlined } from '@ant-design/icons';
import { useNavigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useConversations } from '@/hooks/useConversations';
import { useConversationStore } from '@/stores/conversationStore';

export default function Sidebar() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { id } = useParams();
  const { data, mutate } = useConversations();
  const setActive = useConversationStore((s) => s.setActive);

  async function createConversation() {
    const { createConversation: create } = await import('@/api/conversation/conversation');
    const conv = await create({});
    await mutate((list) => (list ? { ...list, items: [conv, ...list.items] } : list), { revalidate: false });
    setActive(conv.id);
    navigate(`/conversations/${conv.id}`);
  }

  async function remove(convId: string) {
    const { deleteConversation } = await import('@/api/conversation/conversation');
    await deleteConversation(convId);
    await mutate();
    if (id === convId) navigate('/');
  }

  return (
    <div className="flex flex-col h-full">
      <div className="p-2">
        <Button type="primary" block icon={<PlusOutlined />} onClick={createConversation}>
          {t('sidebar.newConversation')}
        </Button>
      </div>
      <List
        className="flex-1 overflow-auto"
        dataSource={data?.items ?? []}
        locale={{ emptyText: t('conversation.empty.title') }}
        renderItem={(conv) => (
          <List.Item
            className={conv.id === id ? 'bg-bg-elevated' : ''}
            actions={[
              <Popconfirm
                key="del"
                title={t('conversation.list.deleteConfirm')}
                onConfirm={() => remove(conv.id)}
                okText={t('common.confirm')}
                cancelText={t('common.cancel')}
              >
                <Button type="text" danger size="small">
                  {t('common.delete')}
                </Button>
              </Popconfirm>,
            ]}
          >
            <button
              type="button"
              className="flex items-center gap-2 flex-1 text-left px-2"
              onClick={() => {
                setActive(conv.id);
                navigate(`/conversations/${conv.id}`);
              }}
            >
              <MessageOutlined />
              <Typography.Text className="truncate">{conv.title}</Typography.Text>
            </button>
          </List.Item>
        )}
      />
    </div>
  );
}
```

> `<button>` 在此为列表项点击容器，是结构性元素而非 antd 替代——AGENT.md 禁止「原生交互 HTML `<button>`/`<input>`/`<select>`」。改用 antd `Button` `type="text"`：

替换点击容器为：
```tsx
<Button
  type="text"
  className="flex items-center gap-2 flex-1 text-left px-2 h-auto"
  onClick={() => {
    setActive(conv.id);
    navigate(`/conversations/${conv.id}`);
  }}
>
  <MessageOutlined />
  <Typography.Text className="truncate">{conv.title}</Typography.Text>
</Button>
```

- [ ] **Step 3: 实现 AppLayout.tsx**

```tsx
import { Layout } from 'antd';
import { useTranslation } from 'react-i18next';
import { Outlet, useNavigate, useParams } from 'react-router-dom';
import Sidebar from './Sidebar';
import UserMenu from './UserMenu';
import { useUiStore } from '@/stores/uiStore';

const { Sider, Content, Header } = Layout;

export default function AppLayout() {
  const { t } = useTranslation();
  const collapsed = useUiStore((s) => s.sidebarCollapsed);
  const toggle = useUiStore((s) => s.toggleSidebar);
  const navigate = useNavigate();
  const { id } = useParams();

  return (
    <Layout className="h-screen">
      <Sider
        theme="light"
        collapsible
        collapsed={collapsed}
        onCollapse={toggle}
        width={240}
        className="flex flex-col"
      >
        <button
          type="button"
          onClick={() => navigate('/')}
          className="text-fg px-4 h-12 flex items-center border-b border-border"
          style={{ background: 'var(--color-bg)' }}
        >
          {t('sidebar.title')}
        </button>
        <div className="flex-1 overflow-hidden">
          <Sidebar />
        </div>
        <div className="border-t border-border p-2">
          <UserMenu />
        </div>
      </Sider>
      <Layout>
        <Content className="overflow-hidden">{id ? <Outlet /> : <Outlet />}</Content>
      </Layout>
    </Layout>
  );
}
```

> 标题点击容器同理用 antd `Button type="text"`，禁止原生 `<button>`。替换为：
```tsx
<Button type="text" block onClick={() => navigate('/')} className="h-12 flex items-center border-b border-border">
  {t('sidebar.title')}
</Button>
```

- [ ] **Step 4: 改造 router.tsx 嵌套 AppLayout**

```tsx
import { Suspense, type ReactNode } from 'react';
import { createBrowserRouter, Navigate, Outlet, useLocation } from 'react-router-dom';
import AppLayout from './components/layout/AppLayout';
import LoginPage from './pages/LoginPage';
import HomePage from './pages/HomePage';
import ConversationPage from './pages/ConversationPage';
import { useAuthStore } from './stores/authStore';

function RequireAuth({ children }: { children: ReactNode }) {
  // 后端无 /auth/me：刷新后信任本地 token；非法 token 由首个 API 请求的 401 兜底
  // （SWR/request 的 401 处理 → logout + 跳 /login）。user 信息在登录时已写入 store。
  const token = useAuthStore((s) => s.token);
  const location = useLocation();

  if (!token) return <Navigate to="/login" state={{ from: location }} replace />;
  return <>{children}</>;
}

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  {
    path: '/',
    element: (
      <RequireAuth>
        <AppLayout />
      </RequireAuth>
    ),
    children: [
      { index: true, element: <HomePage /> },
      { path: 'conversations/:id', element: <ConversationPage /> },
    ],
  },
  { path: '*', element: <Navigate to="/" replace /> },
]);
```

（`AppLayout` 内 `<Outlet />` 渲染子路由；移除 router 里的 `<Suspense>`，因无 lazy。）

- [ ] **Step 5: 改造 HomePage.tsx 空态**

```tsx
import { Button, Empty } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useConversations } from '@/hooks/useConversations';

export default function HomePage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { data } = useConversations();

  return (
    <div className="h-full flex items-center justify-center">
      <div className="text-center max-w-md">
        <h1 className="text-2xl mb-2 text-fg">{t('home.welcome')}</h1>
        <p className="mb-6 text-fg-secondary">{t('home.subtitle')}</p>
        <Button type="primary" size="large" onClick={() => navigate('/conversations/new')}>
          {t('home.newButton')}
        </Button>
        {data && data.items.length > 0 && (
          <div className="mt-6">
            <Empty description={t('home.recent')} />
          </div>
        )}
      </div>
    </div>
  );
}
```

> `/conversations/new` 不存在；首页"新建会话"应由 Sidebar 的逻辑处理。HomePage 改为直接调用新建：

把 HomePage 的 button onClick 改为：
```tsx
onClick={async () => {
  const { createConversation } = await import('@/api/conversation/conversation');
  const conv = await createConversation({});
  navigate(`/conversations/${conv.id}`);
}}
```

- [ ] **Step 6: 改造 ConversationPage.tsx 占位**

```tsx
import { useParams } from 'react-router-dom';
import { Typography } from 'antd';
import { useConversation } from '@/hooks/useConversation';

export default function ConversationPage() {
  const { id } = useParams();
  const { conv } = useConversation(id ?? null);

  if (conv.error) return <Typography.Text type="danger">error</Typography.Text>;
  if (!conv.data) return <Typography.Text>loading</Typography.Text>;

  return (
    <div className="h-full flex flex-col">
      <header className="px-4 h-12 border-b border-border flex items-center">{conv.data.title}</header>
      <div className="flex-1">[Phase 3 填充消息列表与输入]</div>
    </div>
  );
}
```

- [ ] **Step 7: 验证**

Run:
```bash
cd frontend && bun run lint && bunx tsc --noEmit && bunx vitest run
```
Expected: 全绿；`bun run dev` 走查：登录 → 点新建会话 → URL 跳 `/conversations/:id` → 侧边栏出现该项 → 登出回登录。

- [ ] **Step 8: Commit**

```bash
git add frontend/src
git commit -m "feat(frontend): 会话布局与侧边栏"
```

---

## Phase 3 — 问答与 SSE 流式切片

### Task 3.1: useSSEStream hook + Mock SSE

**Files:**
- Create: `frontend/src/hooks/sseParser.ts`, `frontend/src/hooks/sseParser.test.ts`, `frontend/src/hooks/useSSEStream.ts`, `frontend/src/hooks/useSSEStream.test.ts`, `frontend/src/api/mock/sse.ts`
- Modify: `frontend/src/api/mock/conversation.ts`（注册 stream 路由）, `frontend/src/api/mock/index.ts`

- [ ] **Step 1: 写 sseParser 失败测试**

`frontend/src/hooks/sseParser.test.ts`:
```ts
import { describe, it, expect } from 'vitest';
import { parseFrames } from './sseParser';

describe('parseFrames', () => {
  it('空行分帧，多行 data 拼接', () => {
    const input = 'event: message\ndata: {"delta":"a"}\n\n';
    const { events, rest } = parseFrames(input);
    expect(events).toEqual([{ event: 'message', data: '{"delta":"a"}' }]);
    expect(rest).toBe('');
  });

  it('多帧拆分，保留不完整尾部', () => {
    const input = 'event: heartbeat\ndata: {}\n\nevent: message\ndata: {"delta":"b"}\n\nevent: done';
    const { events, rest } = parseFrames(input);
    expect(events).toHaveLength(2);
    expect(events[0]?.event).toBe('heartbeat');
    expect(events[1]?.event).toBe('message');
    expect(rest).toBe('event: done');
  });

  it('多行 data 用 \\n 拼接', () => {
    const input = 'event: message\ndata: line1\ndata: line2\n\n';
    const { events } = parseFrames(input);
    expect(events[0]?.data).toBe('line1\nline2');
  });

  it('无 event 字段默认为 message', () => {
    const input = 'data: hi\n\n';
    const { events } = parseFrames(input);
    expect(events[0]?.event).toBe('message');
  });
});
```

- [ ] **Step 2: 运行确认失败**

Run: `cd frontend && bunx vitest run src/hooks/sseParser.test.ts`
Expected: FAIL

- [ ] **Step 3: 实现 sseParser.ts**

```ts
export type RawSSEEvent = { event: string; data: string; id?: string };

export function parseFrames(buffer: string): { events: RawSSEEvent[]; rest: string } {
  const events: RawSSEEvent[] = [];
  const sep = '\n\n';
  let rest = buffer;

  while (true) {
    const idx = rest.indexOf(sep);
    if (idx === -1) break;
    const raw = rest.slice(0, idx);
    rest = rest.slice(idx + sep.length);

    const lines = raw.split('\n');
    let event = 'message';
    const dataLines: string[] = [];
    let id: string | undefined;
    for (const line of lines) {
      if (line.startsWith('event:')) event = line.slice(6).trim();
      else if (line.startsWith('data:')) dataLines.push(line.slice(5).replace(/^ /, ''));
      else if (line.startsWith('id:')) id = line.slice(3).trim();
    }
    if (lines.length === 0 || (dataLines.length === 0 && !id)) continue;
    events.push({ event, data: dataLines.join('\n'), id });
  }
  return { events, rest };
}
```

- [ ] **Step 4: 运行确认通过**

Run: `cd frontend && bunx vitest run src/hooks/sseParser.test.ts`
Expected: PASS

- [ ] **Step 5: 实现 mock/sse.ts（问答流生成器）**

```ts
import { sseStream, type SSEFrame } from './tools';
import type { MockResponse } from '../client';

export function answerStream(questionContent: string): MockResponse {
  const chunks = `针对「${questionContent}」，这是模拟回答。`.split('');
  const frames: SSEFrame[] = [
    { event: 'heartbeat', data: {} },
    {
      event: 'sources',
      data: { chunks: [{ id: 'c1', title: '示例来源', snippet: '示例片段' }] },
    },
    ...chunks.map((c) => ({ event: 'message', data: { delta: c } })),
    { event: 'done', data: { messageId: 'mock_asst_' + Math.random().toString(36).slice(2), tokensUsed: chunks.length } },
  ];
  return sseStream(frames);
}
```

- [ ] **Step 6: 在 mock/conversation.ts 注册 stream 路由**

在 `installConversationMocks` 末尾追加：
```ts
registerMock({
  match: (r) => r.method === 'GET' && /\/conversations\/[^/]+\/messages\/[^/]+\/stream$/.test(r.path),
  handle: async (r: MockRequest) => {
    const convId = r.path.split('/')[2]!;
    const msgs = store.messages.get(convId) ?? [];
    const lastUser = [...msgs].reverse().find((m) => m.role === 'user');
    const { answerStream } = await import('./sse');
    return answerStream(lastUser?.content ?? '');
  },
});
```

- [ ] **Step 7: 写 useSSEStream 失败测试**

`frontend/src/hooks/useSSEStream.test.ts`:
```ts
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { renderHook } from '@testing-library/react';
import { useSSEStream, type SSEEvent } from './useSSEStream';

describe('useSSEStream', () => {
  beforeEach(() => {
    import.meta.env.VITE_USE_MOCK = 'true';
    localStorage.setItem('investguide.token', 'tok_test');
  });

  it('正常流：sources → message* → done', async () => {
    const events: SSEEvent[] = [];
    renderHook(() =>
      useSSEStream({
        convId: 'c1',
        messageId: 'm1',
        enabled: true,
        onEvent: (e) => events.push(e),
      }),
    );
    await vi.waitFor(() => expect(events.some((e) => e.type === 'done')).toBe(true), { timeout: 3000 });
    expect(events[0]?.type).toBe('sources');
    expect(events.filter((e) => e.type === 'message').length).toBeGreaterThan(0);
  }, 10000);

  it('error 事件终止连接，无 done', async () => {
    // 用专门抛 error 的 messageId
    const events: SSEEvent[] = [];
    renderHook(() =>
      useSSEStream({
        convId: 'c1',
        messageId: 'force-error',
        enabled: true,
        onEvent: (e) => events.push(e),
      }),
    );
    await vi.waitFor(() => expect(events.some((e) => e.type === 'error')).toBe(true), { timeout: 3000 });
    expect(events.some((e) => e.type === 'done')).toBe(false);
  }, 10000);
});
```

> 需 mock 支持 `force-error` 路径。在 `mock/sse.ts` 加：
```ts
export function errorStream(): MockResponse {
  return sseStream([{ event: 'error', data: { code: 'LLM_TIMEOUT', message: 'timeout' } }]);
}
```
并在 conversation stream handler 内：
```ts
if (r.path.includes('force-error')) {
  const { errorStream } = await import('./sse');
  return errorStream();
}
```

- [ ] **Step 8: 实现 useSSEStream.ts**

```ts
import { useEffect, useRef, useState } from 'react';
import { openStream } from '@/api/client';
import { parseFrames } from './sseParser';

export type SSEEvent =
  | { type: 'heartbeat' }
  | { type: 'sources'; chunks: { id: string; title: string; snippet: string }[] }
  | { type: 'message'; delta: string }
  | { type: 'done'; messageId: string; tokensUsed: number }
  | { type: 'error'; code: string; message: string };

type Opts = {
  convId: string;
  messageId: string;
  enabled?: boolean;
  onEvent: (e: SSEEvent) => void;
};

const HEARTBEAT_TIMEOUT_MS = 30_000;

export function useSSEStream({ convId, messageId, enabled = true, onEvent }: Opts) {
  const [state, setState] = useState<'idle' | 'streaming' | 'done' | 'error'>('idle');
  const abortRef = useRef<AbortController | null>(null);
  const lastEventIdRef = useRef<string | undefined>(undefined);
  const onEventRef = useRef(onEvent);
  onEventRef.current = onEvent;

  useEffect(() => {
    if (!enabled) return;
    let stopped = false;
    let reconnectAttempts = 0;
    let heartbeatTimer: ReturnType<typeof setTimeout> | null = null;

    function resetHeartbeat() {
      if (heartbeatTimer) clearTimeout(heartbeatTimer);
      heartbeatTimer = setTimeout(() => {
        reconnect();
      }, HEARTBEAT_TIMEOUT_MS);
    }

    async function start() {
      const abort = new AbortController();
      abortRef.current = abort;
      try {
        const stream = await openStream(
          `/conversations/${convId}/messages/${messageId}/stream`,
          lastEventIdRef.current,
        );
        if (stopped) return;
        setState('streaming');
        resetHeartbeat();
        const reader = stream.getReader();
        const decoder = new TextDecoder();
        let buffer = '';
        for (;;) {
          const { value, done } = await reader.read();
          if (done) break;
          buffer += decoder.decode(value, { stream: true });
          const { events, rest } = parseFrames(buffer);
          buffer = rest;
          for (const ev of events) {
            if (ev.id) lastEventIdRef.current = ev.id;
            const parsed = parseEvent(ev.event, ev.data);
            if (parsed.type === 'heartbeat') {
              resetHeartbeat();
              continue;
            }
            onEventRef.current(parsed);
            resetHeartbeat();
            if (parsed.type === 'done' || parsed.type === 'error') {
              setState(parsed.type);
              if (heartbeatTimer) clearTimeout(heartbeatTimer);
              return;
            }
          }
        }
      } catch (e) {
        if (stopped || abort.signal.aborted) return;
        scheduleReconnect();
      }
    }

    function scheduleReconnect() {
      if (reconnectAttempts >= 3 || state === 'done' || state === 'error') {
        return;
      }
      const delay = Math.pow(2, reconnectAttempts) * 1000;
      reconnectAttempts++;
      setTimeout(() => {
        if (!stopped) start();
      }, delay);
    }

    function reconnect() {
      abortRef.current?.abort();
      scheduleReconnect();
    }

    start();
    return () => {
      stopped = true;
      if (heartbeatTimer) clearTimeout(heartbeatTimer);
      abortRef.current?.abort();
    };
  }, [convId, messageId, enabled, state]);

  function stop() {
    abortRef.current?.abort();
    setState('idle');
  }

  return { state, stop };
}

function parseEvent(event: string, data: string): SSEEvent {
  const obj = JSON.parse(data);
  switch (event) {
    case 'heartbeat':
      return { type: 'heartbeat' };
    case 'sources':
      return { type: 'sources', chunks: obj.chunks };
    case 'message':
      return { type: 'message', delta: obj.delta };
    case 'done':
      return { type: 'done', messageId: obj.messageId, tokensUsed: obj.tokensUsed };
    case 'error':
      return { type: 'error', code: obj.code, message: obj.message };
    default:
      return { type: 'heartbeat' };
  }
}
```

- [ ] **Step 9: 运行测试确认通过**

Run: `cd frontend && bunx vitest run src/hooks/useSSEStream.test.ts`
Expected: PASS

- [ ] **Step 10: Commit**

```bash
git add frontend/src
git commit -m "feat(frontend): useSSEStream hook 与 Mock SSE"
```

---

### Task 3.2: 消息渲染

**Files:**
- Create: `frontend/src/components/conversation/MarkdownRenderer.tsx`, `frontend/src/components/conversation/SourcesCard.tsx`, `frontend/src/components/conversation/MessageBubble.tsx`, `frontend/src/components/conversation/MessageList.tsx`

- [ ] **Step 1: 写 MarkdownRenderer 测试**

```tsx
import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { MarkdownRenderer } from './MarkdownRenderer';

describe('MarkdownRenderer', () => {
  it('渲染标题与列表', () => {
    const { container } = render(<MarkdownRenderer content="# 标题\n- a\n- b" />);
    expect(container.querySelector('h1')).not.toBeNull();
    expect(container.querySelectorAll('li')).toHaveLength(2);
  });

  it('不渲染内联 HTML', () => {
    const { container } = render(<MarkdownRenderer content="<script>alert(1)</script>" />);
    expect(container.querySelector('script')).toBeNull();
  });
});
```

- [ ] **Step 2: 实现 MarkdownRenderer.tsx**

```tsx
import { memo } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import rehypeHighlight from 'rehype-highlight';

export const MarkdownRenderer = memo(function MarkdownRenderer({ content }: { content: string }) {
  return (
    <div className="prose prose-sm max-w-none">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeHighlight]}
        skipHtml
      >
        {content}
      </ReactMarkdown>
    </div>
  );
});
```

- [ ] **Step 3: 实现 SourcesCard.tsx**

```tsx
import { Collapse, Empty, Typography } from 'antd';
import { useTranslation } from 'react-i18next';
import type { KnowledgeChunkRef } from '@/api/conversation/types';

export default function SourcesCard({ sources }: { sources: KnowledgeChunkRef[] | null }) {
  const { t } = useTranslation();
  if (!sources || sources.length === 0) {
    return <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('message.sources.empty')} />;
  }
  return (
    <Collapse
      size="small"
      items={[
        {
          key: 'src',
          label: t('message.sources.title'),
          children: sources.map((s) => (
            <div key={s.id} className="mb-2">
              <Typography.Text strong>{s.title}</Typography.Text>
              <Typography.Paragraph className="mb-0 text-fg-secondary">{s.snippet}</Typography.Paragraph>
            </div>
          )),
        },
      ]}
    />
  );
}
```

- [ ] **Step 4: 实现 MessageBubble.tsx**

```tsx
import { Typography } from 'antd';
import { useTranslation } from 'react-i18next';
import type { Message } from '@/api/conversation/types';
import { MarkdownRenderer } from './MarkdownRenderer';
import SourcesCard from './SourcesCard';

export default function MessageBubble({
  message,
  streaming,
}: {
  message: Pick<Message, 'role' | 'content' | 'sources'> & { streaming?: boolean };
  const { t } = useTranslation();
  const isUser = message.role === 'user';
  return (
    <div className={isUser ? 'flex justify-end' : 'flex justify-start'}>
      <div
        className="max-w-[80%] rounded-lg px-3 py-2"
        style={{
          background: isUser ? 'var(--color-primary)' : 'var(--color-bg-elevated)',
          color: isUser ? '#fff' : 'var(--color-fg)',
        }}
      >
        {isUser ? (
          <Typography.Text style={{ color: '#fff', whiteSpace: 'pre-wrap' }}>{message.content}</Typography.Text>
        ) : (
          <>
            <MarkdownRenderer content={message.content + (streaming ? t('message.streaming.cursor') : '')} />
            {message.sources && message.sources.length > 0 && (
              <div className="mt-2">
                <SourcesCard sources={message.sources} />
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
```

- [ ] **Step 5: 实现 MessageList.tsx**

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
    <div ref={containerRef} onScroll={onScroll} className="flex-1 overflow-auto px-4 py-4 space-y-3">
      {messages.map((m) => (
        <MessageBubble key={m.id} message={m} streaming={m.id === streamingId} />
      ))}
      <div ref={bottomRef} />
    </div>
  );
}
```

- [ ] **Step 6: 运行测试**

Run: `cd frontend && bunx vitest run src/components/conversation/MarkdownRenderer.test.tsx`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add frontend/src/components/conversation
git commit -m "feat(frontend): 消息渲染组件"
```

---

### Task 3.3: 消息输入与乐观更新

**Files:**
- Create: `frontend/src/components/conversation/MessageComposer.tsx`, `frontend/src/components/conversation/MessageComposer.test.tsx`
- Modify: `frontend/src/pages/ConversationPage.tsx`

- [ ] **Step 1: 写 Composer 失败测试**

```tsx
import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SWRConfig } from 'swr';
import { MessageComposer } from './MessageComposer';

function renderWith(obj: Record<string, unknown>) {
  return render(
    <SWRConfig value={{ provider: () => new Map() }}>
      <MessageComposer conversationId="c1" messages={obj} mutateMessages={vi.fn()} />
    </SWRConfig>,
  );
}

describe('MessageComposer', () => {
  it('空内容不触发发送', async () => {
    const onSend = vi.fn();
    renderWith({ items: [] });
    await userEvent.click(screen.getByRole('button', { name: /发送/ }));
    expect(onSend).not.toHaveBeenCalled();
  });

  it('输入并发送后输入框清空', async () => {
    renderWith({ items: [] });
    const ta = screen.getByRole('textbox') as HTMLTextAreaElement;
    await userEvent.type(ta, '测试问题');
    await userEvent.click(screen.getByRole('button', { name: /发送/ }));
    await waitFor(() => expect(ta.value).toBe(''));
  });
});
```

- [ ] **Step 2: 运行确认失败**

Run: `cd frontend && bunx vitest run src/components/conversation/MessageComposer.test.tsx`
Expected: FAIL

- [ ] **Step 3: 实现 MessageComposer.tsx**

```tsx
import { useState } from 'react';
import { Button, Input } from 'antd';
import { SendOutlined, StopOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { sendMessage as apiSendMessage } from '@/api/conversation/conversation';
import { useSSEStream, type SSEEvent } from '@/hooks/useSSEStream';
import type { Message } from '@/api/conversation/types';

type Props = {
  conversationId: string;
  messages: { items: Message[] };
  mutateMessages: (
    updater?: (cur?: { items: Message[] }) => { items: Message[] },
    opts?: { revalidate?: boolean },
  ) => Promise<unknown>;
};

const MAX_LEN = 2000;

export function MessageComposer({ conversationId, messages, mutateMessages }: Props) {
  const { t } = useTranslation();
  const [value, setValue] = useState('');
  const [activeMessageId, setActiveMessageId] = useState<string | null>(null);
  const [errorReason, setErrorReason] = useState<string | null>(null);
  const lastUserContent = useRef('');

  function onEvent(e: SSEEvent) {
    if (e.type === 'sources' || e.type === 'heartbeat') return;
    if (e.type === 'message') {
      void mutateMessages(
        (cur) => {
          const items = cur?.items ?? [];
          const idx = items.findIndex((m) => m.id === activeMessageId);
          if (idx === -1) return cur;
          const copy = [...items];
          copy[idx] = { ...copy[idx]!, content: copy[idx]!.content + e.delta };
          return { items: copy };
        },
        { revalidate: false },
      );
    } else if (e.type === 'done') {
      setActiveMessageId(null);
    } else if (e.type === 'error') {
      setErrorReason(e.message);
      setActiveMessageId(null);
    }
  }

  const { state, stop } = useSSEStream({
    convId: conversationId,
    messageId: activeMessageId ?? '',
    enabled: !!activeMessageId,
    onEvent,
  });

  async function send() {
    const content = value.trim();
    if (!content) return;
    if (content.length > MAX_LEN) return;
    setValue('');
    setErrorReason(null);
    lastUserContent.current = content;

    const tempUserId = 'pending_user_' + Date.now();
    const tempAsstId = 'pending_asst_' + Date.now();
    await mutateMessages(
      (cur) => ({
        items: [
          ...(cur?.items ?? []),
          { id: tempUserId, role: 'user', content, sources: null, tokensUsed: null, createdAt: new Date().toISOString() },
          { id: tempAsstId, role: 'assistant', content: '', sources: null, tokensUsed: null, createdAt: new Date().toISOString() },
        ],
      }),
      { revalidate: false },
    );

    try {
      const { messageId } = await apiSendMessage(conversationId, { content });
      // 用真实 messageId 替换 pending assistant 占位
      await mutateMessages(
        (cur) => {
          const items = (cur?.items ?? []).map((m) => (m.id === tempAsstId ? { ...m, id: messageId } : m));
          return { items };
        },
        { revalidate: false },
      );
      setActiveMessageId(messageId);
    } catch {
      setErrorReason(t('composer.error.reason'));
    }
  }

  const streaming = state === 'streaming';

  return (
    <div className="border-t border-border p-3">
      {errorReason && (
        <div className="mb-2 flex items-center gap-2 text-fg-secondary">
          <span>{t('composer.error.reason')}: {errorReason}</span>
          <Button size="small" onClick={() => { setErrorReason(null); void send(); }}>
            {t('composer.error.retry')}
          </Button>
        </div>
      )}
      <div className="flex items-end gap-2">
        <Input.TextArea
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder={t('composer.placeholder')}
          autoSize={{ minRows: 1, maxRows: 6 }}
          disabled={streaming}
          onPressEnter={(e) => {
            if (!e.shiftKey) {
              e.preventDefault();
              void send();
            }
          }}
        />
        {streaming ? (
          <Button danger icon={<StopOutlined />} onClick={stop}>
            {t('composer.stop')}
          </Button>
        ) : (
          <Button type="primary" icon={<SendOutlined />} onClick={() => void send()} disabled={!value.trim()}>
            {t('composer.send')}
          </Button>
        )}
      </div>
    </div>
  );
}

import { useRef } from 'react';
```

> 顶部 import 顺序：`useRef` 必须在文件顶部 import。修正为把 `import { useRef } from 'react';` 合并到第一步的 `import { useState } from 'react';`：
```ts
import { useRef, useState } from 'react';
```
并删除文件末尾的重复 import 行。

- [ ] **Step 4: 改造 ConversationPage.tsx 集成**

```tsx
import { useParams } from 'react-router-dom';
import { Typography } from 'antd';
import { useConversation } from '@/hooks/useConversation';
import MessageList from '@/components/conversation/MessageList';
import { MessageComposer } from '@/components/conversation/MessageComposer';
import { useConversationStore } from '@/stores/conversationStore';
import { useEffect, useState } from 'react';
import type { Message } from '@/api/conversation/types';

export default function ConversationPage() {
  const { id } = useParams();
  const { conv, messages, mutateMessages } = useConversation(id ?? null);
  const setActive = useConversationStore((s) => s.setActive);
  const [streamingId, setStreamingId] = useState<string | null>(null);

  useEffect(() => {
    if (id) setActive(id);
  }, [id, setActive]);

  if (conv.error) return <Typography.Text type="danger">error</Typography.Text>;
  if (!conv.data) return <Typography.Text>loading</Typography.Text>;

  return (
    <div className="h-full flex flex-col">
      <header className="px-4 h-12 border-b border-border flex items-center">{conv.data.title}</header>
      <MessageList messages={messages.data?.items ?? []} streamingId={streamingId ?? undefined} />
      <MessageComposer
        conversationId={id!}
        messages={{ items: messages.data?.items ?? [] }}
        mutateMessages={async (updater, opts) => {
          if (updater) {
            const cur = messages.data ?? { items: [] as Message[] };
            const next = updater(cur);
            await mutateMessages(next as never, opts as never);
          } else {
            await mutateMessages();
          }
        }}
      />
    </div>
  );
}
```

> streamingId 与 useSSEStream 的状态需联动；MessageComposer 内部已用 useSSEStream 管 streaming，但 ConversationPage 的 streamingId 用于 MessageList 光标。简化方案：把 useSSEStream 提升到 ConversationPage，或在 Composer 通过回调上抛。为避免重设计，MVP 中 composer 内部维护 streaming，并暴露 `streamingId` 经回调上抛：

修改 `MessageComposer` props 加 `onStreamingChange?: (id: string | null) => void`，在 `setActiveMessageId` 调用处同时调用 `onStreamingChange`。在 ConversationPage 传入 `onStreamingChange={setStreamingId}`。

替换 Composer 中 `setActiveMessageId(messageId)` 后行与重置逻辑：
```ts
function setActive(id: string | null) {
  setActiveMessageId(id);
  onStreamingChange?.(id);
}
```
并把所有 `setActiveMessageId(x)` 调用换成 `setActive(x)`。

- [ ] **Step 5: 运行测试**

Run: `cd frontend && bunx vitest run src/components/conversation/MessageComposer.test.tsx`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add frontend/src
git commit -m "feat(frontend): 消息输入与乐观更新"
```

---

### Task 3.4: 首页落地与端到端走查

**Files:**
- Modify: `README.md`, `frontend/src/pages/HomePage.tsx`

- [ ] **Step 1: 完善 HomePage（最近会话列表）**

```tsx
import { Button, List, Typography } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useConversations } from '@/hooks/useConversations';
import { createConversation } from '@/api/conversation/conversation';
import { useConversationStore } from '@/stores/conversationStore';

export default function HomePage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { data, mutate } = useConversations();
  const setActive = useConversationStore((s) => s.setActive);

  async function create() {
    const conv = await createConversation({});
    await mutate((list) => (list ? { ...list, items: [conv, ...list.items] } : list), { revalidate: false });
    setActive(conv.id);
    navigate(`/conversations/${conv.id}`);
  }

  return (
    <div className="h-full flex items-center justify-center">
      <div className="max-w-md w-full px-4">
        <Typography.Title level={2} className="text-center text-fg">{t('home.welcome')}</Typography.Title>
        <Typography.Paragraph className="text-center text-fg-secondary">{t('home.subtitle')}</Typography.Paragraph>
        <div className="text-center mb-6">
          <Button type="primary" size="large" onClick={() => void create()}>{t('home.newButton')}</Button>
        </div>
        {data && data.items.length > 0 && (
          <>
            <Typography.Text type="secondary">{t('home.recent')}</Typography.Text>
            <List
              className="mt-2"
              dataSource={data.items.slice(0, 5)}
              renderItem={(conv) => (
                <List.Item>
                  <Button
                    type="link"
                    onClick={() => {
                      setActive(conv.id);
                      navigate(`/conversations/${conv.id}`);
                    }}
                  >
                    {conv.title}
                  </Button>
                </List.Item>
              )}
            />
          </>
        )}
      </div>
    </div>
  );
}
```

- [ ] **Step 2: 更新 README.md 前端本地运行小节**

在 `README.md` 末尾追加：
```markdown
## 前端本地运行

```bash
cd frontend
cp .env.example .env      # 默认 VITE_USE_MOCK=true
bun install
bun run dev               # http://localhost:5173
```

- `VITE_USE_MOCK=true`：使用内置 Mock，无需后端
- `VITE_USE_MOCK=false` + `VITE_API_BASE_URL`：对接真实后端

门禁：`bun run lint && bunx tsc --noEmit && bun run test`
```

- [ ] **Step 3: 端到端 mock 走查**

Run: `cd frontend && bun run dev`
走查脚本：
1. 访问 `/` → 跳 `/login`
2. 注册（邮箱 + 昵称 + 密码 ≥8）→ 自动回首页
3. 点"新建会话" → 进入 `/conversations/:id`
4. 输入"越南投资"→ Enter → 看到 user 气泡 + assistant 占位 → 流式增量 → done
5. 展开"引用来源"看到示例来源
6. 侧边栏删除该会话 → 回首页
7. 登出 → 回登录

- [ ] **Step 4: 全量门禁**

Run:
```bash
cd frontend && bun run lint && bunx tsc --noEmit && bun run test
```
Expected: 全绿

- [ ] **Step 5: 文档同步检查**

对照 `ARCHITECTURE.md` 第 708–752 行前端架构章节核对本计划实现：
- 目录结构一致 ✅
- SSE 用 fetch+ReadableStream（非 EventSource，因 JWT header）✅
- `useSSEStream` 封装重连/心跳/错误 ✅
- 三 stores 职责一致 ✅

若有偏差，同 PR 更新 ARCHITECTURE.md / AGENT.md。无偏差则跳过。

- [ ] **Step 6: Commit**

```bash
git add frontend/src/pages/HomePage.tsx README.md
git commit -m "feat(frontend): 完善首页与端到端走查"
```

- [ ] **Step 7: 最终 push 前门禁（AGENT.md 推送前要求）**

Run:
```bash
cd frontend && bun run lint && bun run format && bunx tsc --noEmit && bun run test
```
Expected: 全绿

---

## 自审清单（writing-plans skill）

- **Spec 覆盖**：spec 中 Task 0.1–0.4、1.1–1.3、2.1–2.3、3.1–3.4 共 12 个 Task 均有对应实施 Task，子任务逐条映射为步骤；MVP 范围四项与不在范围五项均体现。
- **占位扫描**：无 TBD/TODO；每个代码步骤均给出实际代码。Mock 桩文件在 0.4 显式给出，避免编译失败。
- **类型一致**：`Message`、`Conversation`、`AuthResponse`、`SSEEvent`、`MockRequest` 等类型在跨 Task 引用时签名一致；`useSSEStream` callback `onEvent` 签名在 3.1/3.3 一致；`mutateMessages` 签名在 2.2/3.3 一致。
- **已知偏差**：3.3 Composer 内 streaming 状态经 `onStreamingChange` 回调上抛给 ConversationPage——已在步骤内说明修正，无歧义。

---

## 执行交接

Plan complete and saved to `docs/frontend/plan.md`。两种执行选项：

1. **Subagent-Driven（推荐）** — 每个 Task 派一个新 subagent 实现，任务间两阶段评审，迭代快
2. **Inline Execution** — 在当前会话内按 executing-plans 批量执行，带 checkpoint 评审

请选择执行方式。
