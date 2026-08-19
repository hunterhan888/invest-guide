# InvestGuide 前端全面重写（DeepSeek-Harness 设计体系）Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 推倒 invest-guide 现有 antd + Tailwind 前端，完整照搬 DeepSeek-Harness 设计体系（CSS Modules + 自研基础组件 + `--dsw-*` 设计 token + 双主题），重写 UI 层而保留全部功能逻辑。

**Architecture:** 三层结构——`styles/`（设计 token CSS）+ `primitives/`（自研基础组件）+ `layout/`（AppFrame 三栏布局）与 `pages/`、`features/`。所有视觉值来自 `--dsw-*` token（浅/暗双主题挂 `body[data-ds-dark-theme]`）。数据流（zustand/swr/react-router/i18next/react-markdown）与 API 契约完全不变。

**Tech Stack:** React 19 + TypeScript + Vite 5 + CSS Modules + zustand + react-router-dom 6 + swr + i18next + react-markdown。参考源码：`/home/hunter/code/deepseek-harness`（包 `packages/client/ui-*`、`packages/client/ui-theme/src/styles/*`）。

**验证命令（每个任务末跑）:** `cd frontend && bun run lint && bunx tsc --noEmit && bun run test`

---

## 任务前准备

在开始任何任务前，先确认参考文件位置与基线测试：

- [ ] **Step 1: 确认参考文件存在**

Run:
```bash
ls /home/hunter/code/deepseek-harness/packages/client/ui-theme/src/styles/design-platform.css
ls /home/hunter/code/deepseek-harness/packages/client/ui-theme/src/styles/base.css
ls /home/hunter/code/deepseek-harness/packages/client/ui-theme/src/styles/scrollbar.css
```
Expected: 三个文件都存在。

- [ ] **Step 2: 基线测试确认**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test`
Expected: 全部通过（当前 38 个测试，可能因新增用例略多）。

- [ ] **Step 3: 提交基线**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git status --short && git log --oneline -3
```
Expected: 工作区干净，最近提交为 `f97383d docs(frontend): 新增 DeepSeek-Harness 风格前端重写设计文档`。

---

### Task 1: 依赖清理与构建配置

**Files:**
- Modify: `frontend/package.json`
- Modify: `frontend/vite.config.ts`
- Modify: `frontend/tsconfig.json`

- [ ] **Step 1: 更新依赖**

编辑 `frontend/package.json`：
- 从 `dependencies` 移除：`antd`、`@ant-design/icons`
- 从 `devDependencies` 移除：`@tailwindcss/vite`、`tailwindcss`
- 保留其余全部（react、react-dom、react-router-dom、react-i18next、i18next、react-markdown、remark-gfm、rehype-highlight、swr、zustand）

修改后 `dependencies` 应为：
```json
"dependencies": {
  "i18next": "^26.3.6",
  "react": "^19",
  "react-dom": "^19",
  "react-i18next": "^17.0.11",
  "react-markdown": "^10.1.0",
  "react-router-dom": "^6",
  "rehype-highlight": "^7.0.2",
  "remark-gfm": "^4.0.1",
  "swr": "^2.4.2",
  "zustand": "^5.0.14"
}
```

- [ ] **Step 2: 安装依赖**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun install`
Expected: 安装成功，antd/tailwind 相关包被移除。

- [ ] **Step 3: 更新 vite.config.ts**

编辑 `frontend/vite.config.ts`，移除 `tailwindcss` 插件，保留 CSS Modules 支持（Vite 内置）：

```ts
/// <reference types="vitest" />
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'node:path';

export default defineConfig({
  plugins: [react()],
  resolve: { alias: { '@': path.resolve(__dirname, './src') } },
  server: {
    port: 5173,
    proxy: {
      '/api/v1': {
        target: process.env.VITE_API_PROXY_TARGET ?? 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  test: {
    environment: 'jsdom',
    globals: true,
    setupFiles: ['./src/test/setup.ts'],
    css: false,
  },
});
```

- [ ] **Step 4: 更新 tsconfig.json 允许 CSS 模块导入**

编辑 `frontend/tsconfig.json` 的 `compilerOptions`，添加 `"types": ["vite/client"]` 外，确保没有把 `*.module.css` 当作错误。实际通过 `src/vite-env.d.ts` 补充声明（见 Task 2），tsconfig 无需改动。仅确认 `include: ["src"]` 不变即可。

- [ ] **Step 5: 验证配置**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bunx tsc --noEmit`
Expected: 会报错（antd 组件仍被引用），确认报错只来自 antd 相关文件，说明依赖与配置已切换。

- [ ] **Step 6: 提交**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git add frontend/package.json frontend/bun.lock frontend/vite.config.ts && git commit -m "build(frontend): 移除 antd/tailwind，切换为 CSS Modules 构建"
```

---

### Task 2: 设计 token 样式体系

**Files:**
- Create: `frontend/src/styles/tokens.css`
- Create: `frontend/src/styles/base.css`
- Create: `frontend/src/styles/scrollbar.css`
- Create: `frontend/src/styles/main.css`（重写）
- Modify: `frontend/src/vite-env.d.ts`（补充 CSS Modules 声明）

- [ ] **Step 1: 拷贝 design-platform.css**

Run:
```bash
mkdir -p /home/hunter/code/invest-guide-workspace/invest-guide/frontend/src/styles
cp /home/hunter/code/deepseek-harness/packages/client/ui-theme/src/styles/design-platform.css \
   /home/hunter/code/invest-guide-workspace/invest-guide/frontend/src/styles/tokens.css
```
Expected: `frontend/src/styles/tokens.css` 包含 `--dsw-static-*`、`--dsw-alias-*`、`--dsw-specific-*` 明暗两套定义。

- [ ] **Step 2: 补充 tokens.css 缺失的自定义属性**

`tokens.css` 引用了 `--dsw-shadow-lv2/lv3`、`--dsw-mask-blur`、`--dsw-separator-primary`，这些在 DeepSeek 上游 global.css 定义。在 `tokens.css` 文件末尾追加（在 `body` 规则之后、文件顶部 `body` 块内追加亦可，但注意作用域——追加到 `:root`）：

在文件顶部 `body {` 块的最后一个 `}` 之前（即 `--dsw-specific-tip` 行之后）追加浅色值：
```css
  --dsw-shadow-lv1: 0 1px 2px rgba(0, 0, 0, 0.04);
  --dsw-shadow-lv2: 0 4px 16px rgba(0, 0, 0, 0.08);
  --dsw-shadow-lv3: 0 12px 32px rgba(0, 0, 0, 0.12);
  --dsw-mask-blur: blur(2px);
  --dsw-separator-primary: var(--dsw-static-neutral-bluish-300);
```
在暗色 `body[data-ds-dark-theme] {` 块的最后一个 `}` 之前追加：
```css
  --dsw-shadow-lv1: 0 1px 2px rgba(0, 0, 0, 0.3);
  --dsw-shadow-lv2: 0 4px 16px rgba(0, 0, 0, 0.5);
  --dsw-shadow-lv3: 0 12px 32px rgba(0, 0, 0, 0.6);
  --dsw-mask-blur: blur(2px);
  --dsw-separator-primary: var(--dsw-static-neutral-bluish-400);
```

- [ ] **Step 3: 创建 base.css**

创建 `frontend/src/styles/base.css`：
```css
/* Base variables referenced by the token sheets and component CSS. Code font
   stack deliberately omits a bare `monospace` tail (Windows CJK falls back to
   SimSun otherwise). */
:root {
  --dsw-font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC',
    'Hiragino Sans GB', 'Microsoft YaHei', 'Helvetica Neue', Helvetica, Arial, sans-serif;
  --ds-font-family-code: 'SF Mono', 'JetBrains Mono', 'Fira Code', Consolas,
    'Liberation Mono', Menlo, Courier, 'PingFang SC', 'Microsoft YaHei';
  --ds-ease-in-out: cubic-bezier(0.4, 0, 0.2, 1);
  --ds-transition-duration: 0.2s;
  --ds-transition-duration-fast: 0.1s;
  --ds-transition-duration-slow: 0.3s;

  /* Shared layout values (conversation column / composer clearance). */
  --dsh-chat-content-width: 736px;
  --dsh-composer-side-clearance: 16px;
}
```

- [ ] **Step 4: 创建 scrollbar.css**

Run:
```bash
cp /home/hunter/code/deepseek-harness/packages/client/ui-theme/src/styles/scrollbar.css \
   /home/hunter/code/invest-guide-workspace/invest-guide/frontend/src/styles/scrollbar.css
```
Expected: 复制成功（该文件是自包含的，仅依赖 `--dsw-alias-scrollbar-*` 与 `--dsh-scrollbar-thumb{,-hover}`，由 tokens.css / 各表面声明）。

- [ ] **Step 5: 重写 main.css**

用以下内容完全覆盖 `frontend/src/styles/main.css`（移除 Tailwind `@import` 与 `@theme`）：

```css
@import './tokens.css';
@import './base.css';
@import './scrollbar.css';

html,
body,
#root {
  height: 100%;
  margin: 0;
}

body {
  background: var(--dsw-alias-bg-base);
  color: var(--dsw-alias-label-primary);
  font-size: 16px;
  line-height: 24px;
  font-family: var(--dsw-font-family);
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}

/* Markdown 基础排版（assistant 消息） */
.md-body {
  font-size: 16px;
  line-height: 28px;
  color: var(--dsw-alias-label-primary);
  word-break: break-word;
}
.md-body > :first-child {
  margin-top: 0;
}
.md-body > :last-child {
  margin-bottom: 0;
}
.md-body p {
  margin: 0 0 16px;
}
.md-body h1,
.md-body h2,
.md-body h3,
.md-body h4,
.md-body h5,
.md-body h6 {
  margin: 24px 0 12px;
  font-weight: 600;
  line-height: 1.3;
  color: var(--dsw-alias-label-primary);
}
.md-body h1 {
  font-size: 24px;
}
.md-body h2 {
  font-size: 20px;
}
.md-body h3 {
  font-size: 17px;
}
.md-body ul,
.md-body ol {
  margin: 0 0 16px;
  padding-left: 24px;
}
.md-body li {
  margin: 4px 0;
}
.md-body code {
  background: var(--dsw-alias-markdown-inline-code);
  border-radius: 6px;
  padding: 2px 6px;
  font-family: var(--ds-font-family-code);
  font-size: 0.9em;
}
.md-body pre {
  background: var(--dsw-alias-markdown-code-block);
  border: 1px solid var(--dsw-alias-border-l1);
  border-radius: 12px;
  padding: 12px 16px;
  overflow-x: auto;
  margin: 0 0 16px;
  position: relative;
}
.md-body pre > code {
  background: transparent;
  padding: 0;
}
.md-body blockquote {
  margin: 0 0 16px;
  padding: 4px 16px;
  border-left: 3px solid var(--dsw-alias-markdown-tag);
  color: var(--dsw-alias-label-secondary);
}
.md-body table {
  width: 100%;
  border-collapse: collapse;
  margin: 0 0 16px;
  font-size: 14px;
}
.md-body th,
.md-body td {
  border: 1px solid var(--dsw-alias-border-l2);
  padding: 8px 12px;
  text-align: left;
}
.md-body th {
  background: var(--dsw-alias-markdown-tag);
  font-weight: 600;
}
.md-body hr {
  border: none;
  border-top: 1px solid var(--dsw-alias-border-l2);
  margin: 24px 0;
}
.md-body a {
  color: var(--dsw-alias-brand-primary);
  text-decoration: none;
}
.md-body a:hover {
  text-decoration: underline;
}

/* DeepSeek 风格内联引用芯片 */
.source-chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 20px;
  min-width: 20px;
  padding: 0 6px;
  margin: 0 2px;
  border-radius: 6px;
  font-size: 11px;
  font-weight: 600;
  line-height: 1;
  color: var(--dsw-alias-label-primary);
  background: rgba(97, 135, 216, 0.22);
  cursor: pointer;
  vertical-align: middle;
  user-select: none;
  transition:
    background var(--ds-transition-duration) var(--ds-ease-in-out),
    transform 0.15s;
}
.source-chip:hover {
  background: var(--dsw-static-deepseek-200);
  transform: scale(1.08);
}
.source-chip:active {
  transform: scale(0.95);
}

.source-item {
  transition: background-color var(--ds-transition-duration) var(--ds-ease-in-out);
}
.source-highlight {
  background: var(--dsw-alias-interactive-bg-hover-accent) !important;
}
```

- [ ] **Step 6: 补充 vite-env.d.ts**

在 `frontend/src/vite-env.d.ts` 末尾追加：
```ts
declare module '*.module.css' {
  const classes: Record<string, string>;
  export default classes;
}
declare module '*.css';
```

- [ ] **Step 7: 验证 token 加载**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bunx tsc --noEmit`
Expected: 与 Task 1 相同的 antd 相关报错（不影响本次验证），CSS 导入不再报 TS 错误。

- [ ] **Step 8: 提交**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git add frontend/src/styles frontend/src/vite-env.d.ts && git commit -m "style(frontend): 引入 DeepSeek 设计 token 体系（tokens/base/scrollbar）"
```

---

### Task 3: 主题 store 与 ThemeProvider

**Files:**
- Create: `frontend/src/theme/themeStore.ts`
- Create: `frontend/src/theme/ThemeProvider.tsx`
- Create: `frontend/src/theme/logo.tsx`
- Create: `frontend/src/theme/themeStore.test.ts`

- [ ] **Step 1: 写失败测试**

创建 `frontend/src/theme/themeStore.test.ts`：
```ts
import { describe, it, expect, beforeEach } from 'vitest';
import { useThemeStore } from './themeStore';

describe('themeStore', () => {
  beforeEach(() => localStorage.clear());

  it('toggle 在 light/dark 间切换并持久化', () => {
    useThemeStore.getState().setMode('light');
    useThemeStore.getState().toggle();
    expect(useThemeStore.getState().mode).toBe('dark');
    expect(localStorage.getItem('investguide.theme')).toBe('dark');
  });

  it('hydrate 读取已保存偏好', () => {
    localStorage.setItem('investguide.theme', 'dark');
    useThemeStore.getState().hydrate();
    expect(useThemeStore.getState().mode).toBe('dark');
  });
});
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/theme/themeStore.test.ts`
Expected: FAIL（找不到 `./themeStore`）。

- [ ] **Step 3: 实现 themeStore**

创建 `frontend/src/theme/themeStore.ts`：
```ts
import { create } from 'zustand';

export type ThemeMode = 'light' | 'dark';

const THEME_KEY = 'investguide.theme';

type ThemeState = {
  mode: ThemeMode;
  setMode: (m: ThemeMode) => void;
  toggle: () => void;
  hydrate: () => void;
};

function readSaved(): ThemeMode {
  const saved = localStorage.getItem(THEME_KEY);
  if (saved === 'light' || saved === 'dark') return saved;
  if (typeof window !== 'undefined' && window.matchMedia?.('(prefers-color-scheme: dark)').matches) {
    return 'dark';
  }
  return 'light';
}

export const useThemeStore = create<ThemeState>((set, get) => ({
  mode: 'light',
  setMode: (mode) => {
    localStorage.setItem(THEME_KEY, mode);
    set({ mode });
  },
  toggle: () => {
    const next = get().mode === 'dark' ? 'light' : 'dark';
    get().setMode(next);
  },
  hydrate: () => set({ mode: readSaved() }),
}));

useThemeStore.getState().hydrate();
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/theme/themeStore.test.ts`
Expected: PASS。

- [ ] **Step 5: 实现 ThemeProvider**

创建 `frontend/src/theme/ThemeProvider.tsx`：
```tsx
import { useEffect, type ReactNode } from 'react';
import { useThemeStore } from './themeStore';

export function ThemeProvider({ children }: { children: ReactNode }) {
  const mode = useThemeStore((s) => s.mode);

  useEffect(() => {
    document.body.toggleAttribute('data-ds-dark-theme', mode === 'dark');
  }, [mode]);

  return <>{children}</>;
}
```

- [ ] **Step 6: 实现品牌 logo**

创建 `frontend/src/theme/logo.tsx`：
```tsx
export function CompassLogo({ size = 28 }: { size?: number }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 32 32"
      fill="none"
      aria-hidden="true"
    >
      <circle cx="16" cy="16" r="15" fill="url(#ig-logo-grad)" />
      <circle cx="16" cy="16" r="15" stroke="rgba(255,255,255,0.35)" strokeWidth="1" />
      <path
        d="M21.5 10.5l-2.6 7.8-7.8 2.6 2.6-7.8z"
        fill="#fff"
      />
      <circle cx="16" cy="16" r="2.2" fill="#fff" />
      <defs>
        <linearGradient id="ig-logo-grad" x1="0" y1="0" x2="32" y2="32">
          <stop stopColor="#6792f5" />
          <stop offset="1" stopColor="#3f72d8" />
        </linearGradient>
      </defs>
    </svg>
  );
}
```

- [ ] **Step 7: 运行全部测试**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run lint && bunx tsc --noEmit && bun run test`
Expected: 除 antd 相关文件报错外，theme 测试通过。

- [ ] **Step 8: 提交**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git add frontend/src/theme && git commit -m "feat(frontend): 新增主题 store、ThemeProvider 与品牌 logo"
```

---

### Task 4: Toast 系统

**Files:**
- Create: `frontend/src/primitives/Toast.tsx`
- Create: `frontend/src/primitives/Toast.module.css`
- Create: `frontend/src/primitives/ToastProvider.tsx`
- Create: `frontend/src/primitives/Toast.test.tsx`

- [ ] **Step 1: 写失败测试**

创建 `frontend/src/primitives/Toast.test.tsx`：
```tsx
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { act } from 'react';
import { ToastProvider, useToast } from './ToastProvider';

function Harness() {
  const toast = useToast();
  return (
    <button onClick={() => toast.error('出错了')}>trigger</button>
  );
}

describe('ToastProvider', () => {
  it('调用 toast.error 后渲染提示并自动消失', async () => {
    const userEvent = (await import('@testing-library/user-event')).default;
    const user = userEvent.setup();
    render(
      <ToastProvider>
        <Harness />
      </ToastProvider>,
    );
    await user.click(screen.getByRole('button', { name: 'trigger' }));
    expect(screen.getByText('出错了')).toBeInTheDocument();
    await act(async () => {
      await new Promise((r) => setTimeout(r, 3200));
    });
    expect(screen.queryByText('出错了')).not.toBeInTheDocument();
  });
});
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/primitives/Toast.test.tsx`
Expected: FAIL（找不到 `./ToastProvider`）。

- [ ] **Step 3: 实现 ToastProvider**

创建 `frontend/src/primitives/ToastProvider.tsx`：
```tsx
import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react';
import { Toast, type ToastKind } from './Toast';

type ToastItem = { id: number; kind: ToastKind; text: string };
type ToastApi = { success: (text: string) => void; error: (text: string) => void; info: (text: string) => void };

const ToastContext = createContext<ToastApi | null>(null);

const HOLD_MS = 3000;

export function ToastProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<ToastItem[]>([]);
  const nextId = useRef(0);

  const show = useCallback((kind: ToastKind, text: string) => {
    const id = ++nextId.current;
    setItems((cur) => [...cur, { id, kind, text }]);
    setTimeout(() => {
      setItems((cur) => cur.filter((t) => t.id !== id));
    }, HOLD_MS + 1000);
  }, []);

  const api = useMemo<ToastApi>(
    () => ({
      success: (t) => show('success', t),
      error: (t) => show('error', t),
      info: (t) => show('info', t),
    }),
    [show],
  );

  return (
    <ToastContext.Provider value={api}>
      {children}
      {items.map((t) => (
        <Toast key={t.id} kind={t.kind} text={t.text} />
      ))}
    </ToastContext.Provider>
  );
}

export function useToast(): ToastApi {
  const ctx = useContext(ToastContext);
  if (!ctx) throw new Error('useToast must be used within ToastProvider');
  return ctx;
}
```

- [ ] **Step 4: 实现 Toast 组件**

创建 `frontend/src/primitives/Toast.tsx`：
```tsx
import styles from './Toast.module.css';

export type ToastKind = 'success' | 'error' | 'info';

const KIND_ICON: Record<ToastKind, string> = {
  success: '✓',
  error: '✕',
  info: 'i',
};

export function Toast({ kind, text }: { kind: ToastKind; text: string }) {
  return (
    <div className={styles.toast} role="status">
      <span className={`${styles.icon} ${styles[kind]}`} aria-hidden="true">
        {KIND_ICON[kind]}
      </span>
      <span className={styles.text}>{text}</span>
    </div>
  );
}
```

创建 `frontend/src/primitives/Toast.module.css`：
```css
.toast {
  position: fixed;
  top: 120px;
  left: 50%;
  z-index: 1100;
  pointer-events: none;
  display: flex;
  align-items: center;
  gap: 10px;
  max-width: min(560px, calc(100vw - 48px));
  padding: 12px 16px;
  border-radius: 14px;
  background: var(--dsw-alias-button-contrast-fill);
  color: var(--dsw-alias-label-primary-inverted);
  font-size: 14px;
  line-height: 22px;
  box-shadow: var(--dsw-shadow-lv3);
  transform: translateX(-50%);
  animation:
    dsh-toast-in 160ms ease-out,
    dsh-toast-fade 500ms ease 3000ms forwards;
}

.icon {
  display: grid;
  place-items: center;
  flex: none;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  font-size: 11px;
  font-weight: 700;
  line-height: 1;
}

.icon.success {
  color: var(--dsw-static-green-400);
}
.icon.error {
  color: var(--dsw-static-red-400);
}
.icon.info {
  color: var(--dsw-static-deepseek-300);
}

.text {
  min-width: 0;
}

@keyframes dsh-toast-in {
  from {
    opacity: 0;
    transform: translate(-50%, -6px);
  }
  to {
    opacity: 1;
    transform: translate(-50%, 0);
  }
}

@keyframes dsh-toast-fade {
  to {
    opacity: 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .toast {
    animation: dsh-toast-fade 500ms ease 3000ms forwards;
  }
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/primitives/Toast.test.tsx`
Expected: PASS（3 秒自动消失用例可能因计时器略慢，若 flaky 可将 HOLD_MS 逻辑保持在 3000，等待 `waitFor` 而非固定 sleep——如失败改用 `await waitFor(() => expect(...).not.toBeInTheDocument(), { timeout: 5000 })`）。

- [ ] **Step 6: 提交**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git add frontend/src/primitives/Toast.tsx frontend/src/primitives/Toast.module.css frontend/src/primitives/ToastProvider.tsx frontend/src/primitives/Toast.test.tsx && git commit -m "feat(frontend): 新增自研 Toast 系统"
```

---

### Task 5: 图标集

**Files:**
- Create: `frontend/src/primitives/icons.tsx`
- Create: `frontend/src/primitives/icons.test.tsx`

- [ ] **Step 1: 写失败测试**

创建 `frontend/src/primitives/icons.test.tsx`：
```tsx
import { describe, it, expect } from 'vitest';
import { render } from '@testing-library/react';
import { SendIcon } from './icons';

describe('icons', () => {
  it('渲染 SVG 并支持 size', () => {
    const { container } = render(<SendIcon size={20} />);
    const svg = container.querySelector('svg');
    expect(svg).not.toBeNull();
    expect(svg).toHaveAttribute('width', '20');
  });
});
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/primitives/icons.test.tsx`
Expected: FAIL（找不到 `./icons`）。

- [ ] **Step 3: 实现图标集**

创建 `frontend/src/primitives/icons.tsx`：
```tsx
type IconProps = { size?: number; className?: string };

function Base({ size = 16, className, children }: IconProps & { children: React.ReactNode }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden="true"
    >
      {children}
    </svg>
  );
}

export function SendIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M12 19V5" />
      <path d="M5 12l7-7 7 7" />
    </Base>
  );
}

export function StopIcon(p: IconProps) {
  return (
    <Base {...p}>
      <rect x="7" y="7" width="10" height="10" rx="2" />
    </Base>
  );
}

export function PlusIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M12 5v14M5 12h14" />
    </Base>
  );
}

export function MoreIcon(p: IconProps) {
  return (
    <Base {...p}>
      <circle cx="5" cy="12" r="1.6" fill="currentColor" stroke="none" />
      <circle cx="12" cy="12" r="1.6" fill="currentColor" stroke="none" />
      <circle cx="19" cy="12" r="1.6" fill="currentColor" stroke="none" />
    </Base>
  );
}

export function CloseIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M18 6L6 18M6 6l12 12" />
    </Base>
  );
}

export function LogoutIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
      <path d="M16 17l5-5-5-5" />
      <path d="M21 12H9" />
    </Base>
  );
}

export function SunIcon(p: IconProps) {
  return (
    <Base {...p}>
      <circle cx="12" cy="12" r="4" />
      <path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" />
    </Base>
  );
}

export function MoonIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z" />
    </Base>
  );
}

export function PanelLeftIcon(p: IconProps) {
  return (
    <Base {...p}>
      <rect x="3" y="3" width="18" height="18" rx="2" />
      <path d="M9 3v18" />
    </Base>
  );
}

export function PanelRightIcon(p: IconProps) {
  return (
    <Base {...p}>
      <rect x="3" y="3" width="18" height="18" rx="2" />
      <path d="M15 3v18" />
    </Base>
  );
}

export function ChevronDownIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M6 9l6 6 6-6" />
    </Base>
  );
}

export function ChevronRightIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M9 6l6 6-6 6" />
    </Base>
  );
}

export function ArrowDownIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M12 5v14M5 12l7 7 7-7" />
    </Base>
  );
}

export function DeleteIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M3 6h18M8 6V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6" />
      <path d="M10 11v6M14 11v6" />
    </Base>
  );
}

export function CopyIcon(p: IconProps) {
  return (
    <Base {...p}>
      <rect x="9" y="9" width="13" height="13" rx="2" />
      <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
    </Base>
  );
}

export function RefreshIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M21 12a9 9 0 1 1-2.6-6.4M21 3v6h-6" />
    </Base>
  );
}

export function DocumentIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" />
      <path d="M14 2v6h6" />
    </Base>
  );
}

export function ClockIcon(p: IconProps) {
  return (
    <Base {...p}>
      <circle cx="12" cy="12" r="9" />
      <path d="M12 7v5l3 3" />
    </Base>
  );
}

export function SparkleIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M12 3l1.8 5.2L19 10l-5.2 1.8L12 17l-1.8-5.2L5 10l5.2-1.8z" />
    </Base>
  );
}

export function ChevronsUpIcon(p: IconProps) {
  return (
    <Base {...p}>
      <path d="M17 11l-5-5-5 5M17 18l-5-5-5 5" />
    </Base>
  );
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/primitives/icons.test.tsx`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git add frontend/src/primitives/icons.tsx frontend/src/primitives/icons.test.tsx && git commit -m "feat(frontend): 新增自研 SVG 图标集"
```

---

### Task 6: Button 组件

**Files:**
- Create: `frontend/src/primitives/Button.tsx`
- Create: `frontend/src/primitives/Button.module.css`
- Create: `frontend/src/primitives/Button.test.tsx`

- [ ] **Step 1: 写失败测试**

创建 `frontend/src/primitives/Button.test.tsx`：
```tsx
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from './Button';

describe('Button', () => {
  it('渲染 primary 按钮并触发 onClick', async () => {
    const onClick = vi.fn();
    const user = userEvent.setup();
    render(
      <Button variant="primary" onClick={onClick}>
        发送
      </Button>,
    );
    await user.click(screen.getByRole('button', { name: '发送' }));
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it('loading 时禁用', () => {
    render(<Button loading>发送</Button>);
    expect(screen.getByRole('button')).toBeDisabled();
  });

  it('disabled 禁用', () => {
    render(<Button disabled>发送</Button>);
    expect(screen.getByRole('button')).toBeDisabled();
  });

  it('renderProp children 渲染字符串', () => {
    render(<Button>文字</Button>);
    expect(screen.getByRole('button')).toHaveTextContent('文字');
  });
});
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/primitives/Button.test.tsx`
Expected: FAIL（找不到 `./Button`）。

- [ ] **Step 3: 实现 Button**

创建 `frontend/src/primitives/Button.tsx`：
```tsx
import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from 'react';
import styles from './Button.module.css';

type Variant = 'primary' | 'ghost' | 'outline' | 'toolbar';
type Size = 'sm' | 'md';

export type ButtonProps = {
  variant?: Variant;
  size?: Size;
  block?: boolean;
  icon?: ReactNode;
  loading?: boolean;
  className?: string;
} & Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'className'>;

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  { variant = 'ghost', size = 'md', block, icon, loading, className, children, disabled, ...rest },
  ref,
) {
  const classes = [
    styles.button,
    styles[variant],
    styles[size],
    block ? styles.block : '',
    className ?? '',
  ]
    .filter(Boolean)
    .join(' ');
  return (
    <button
      ref={ref}
      className={classes}
      disabled={disabled || loading}
      {...rest}
    >
      {icon && <span className={styles.icon}>{icon}</span>}
      {children != null && <span>{children}</span>}
      {loading && <span className={styles.spinner} aria-hidden="true" />}
    </button>
  );
});
```

创建 `frontend/src/primitives/Button.module.css`：
```css
.button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  border: none;
  border-radius: 18px;
  cursor: pointer;
  font-size: 14px;
  line-height: 22px;
  font-family: var(--dsw-font-family);
  color: var(--dsw-alias-label-primary);
  background: transparent;
  padding: 0 14px;
  box-sizing: border-box;
  transition:
    background var(--ds-transition-duration) var(--ds-ease-in-out),
    border-color var(--ds-transition-duration) var(--ds-ease-in-out),
    opacity var(--ds-transition-duration) var(--ds-ease-in-out);
}

.button:disabled {
  cursor: not-allowed;
  opacity: 0.4;
}

.md {
  height: 36px;
}

.sm {
  height: 28px;
  font-size: 12px;
  line-height: 18px;
  padding: 0 10px;
  border-radius: 14px;
}

.primary {
  background: var(--dsw-alias-button-primary-fill);
  color: var(--dsw-alias-label-primary-foreground);
}
.primary:hover:not(:disabled) {
  background: var(--dsw-alias-button-primary-hover);
}

.ghost:hover:not(:disabled) {
  background: var(--dsw-alias-interactive-bg-hover);
}
.ghost:active:not(:disabled) {
  background: var(--dsw-alias-interactive-bg-active);
}

.outline {
  border: 1px solid var(--dsw-alias-border-l2);
  background: transparent;
}
.outline:hover:not(:disabled) {
  background: var(--dsw-alias-interactive-bg-hover);
}

.toolbar {
  background: var(--dsw-alias-button-tool-bar-fill);
  color: var(--dsw-alias-label-primary-inverted);
}
.toolbar:hover:not(:disabled) {
  background: var(--dsw-alias-button-tool-bar-hover);
}

.icon {
  display: inline-flex;
  width: 16px;
  height: 16px;
  align-items: center;
  justify-content: center;
}

.block {
  display: flex;
  width: 100%;
}

.spinner {
  width: 14px;
  height: 14px;
  border: 2px solid currentcolor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: dsh-spin 0.8s linear infinite;
}

@keyframes dsh-spin {
  to {
    transform: rotate(360deg);
  }
}

@media (prefers-reduced-motion: reduce) {
  .spinner {
    animation: none;
  }
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/primitives/Button.test.tsx`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git add frontend/src/primitives/Button.tsx frontend/src/primitives/Button.module.css frontend/src/primitives/Button.test.tsx && git commit -m "feat(frontend): 新增自研 Button 组件"
```

---

### Task 7: Input / Textarea 组件

**Files:**
- Create: `frontend/src/primitives/Input.tsx`
- Create: `frontend/src/primitives/Input.module.css`
- Create: `frontend/src/primitives/Textarea.tsx`
- Create: `frontend/src/primitives/Textarea.module.css`
- Create: `frontend/src/primitives/Input.test.tsx`

- [ ] **Step 1: 写失败测试**

创建 `frontend/src/primitives/Input.test.tsx`：
```tsx
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Input } from './Input';
import { Textarea } from './Textarea';

describe('Input', () => {
  it('渲染输入框并支持受控值', async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    render(<Input value="" onChange={onChange} placeholder="邮箱" />);
    const input = screen.getByPlaceholderText('邮箱');
    await user.type(input, 'a@b.com');
    expect(onChange).toHaveBeenCalled();
  });

  it('支持 type=password', () => {
    render(<Input type="password" placeholder="密码" />);
    expect(screen.getByPlaceholderText('密码')).toHaveAttribute('type', 'password');
  });
});

describe('Textarea', () => {
  it('渲染 textarea', () => {
    render(<Textarea placeholder="问题" />);
    expect(screen.getByPlaceholderText('问题').tagName).toBe('TEXTAREA');
  });
});
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/primitives/Input.test.tsx`
Expected: FAIL（找不到模块）。

- [ ] **Step 3: 实现 Input**

创建 `frontend/src/primitives/Input.tsx`：
```tsx
import { forwardRef, type InputHTMLAttributes, type ReactNode } from 'react';
import styles from './Input.module.css';

export type InputProps = {
  icon?: ReactNode;
  className?: string;
} & Omit<InputHTMLAttributes<HTMLInputElement>, 'className'>;

export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { icon, className, ...rest },
  ref,
) {
  return (
    <span className={styles.wrap}>
      {icon && <span className={styles.icon}>{icon}</span>}
      <input ref={ref} className={`${styles.input} ${className ?? ''}`.trim()} {...rest} />
    </span>
  );
});
```

创建 `frontend/src/primitives/Input.module.css`：
```css
.wrap {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  height: 40px;
  padding: 0 12px;
  border: 1px solid var(--dsw-alias-border-l2);
  border-radius: 10px;
  background: var(--dsw-alias-bg-layer-1);
  box-sizing: border-box;
  transition:
    border-color var(--ds-transition-duration) var(--ds-ease-in-out),
    box-shadow var(--ds-transition-duration) var(--ds-ease-in-out);
}

.wrap:focus-within {
  border-color: var(--dsw-alias-brand-primary-new-colorprimary-new-color);
  box-shadow: 0 0 0 3px var(--dsw-alias-interactive-bg-hover-accent);
}

.icon {
  display: inline-flex;
  width: 16px;
  height: 16px;
  align-items: center;
  justify-content: center;
  color: var(--dsw-alias-label-tertiary);
}

.input {
  flex: 1;
  min-width: 0;
  border: none;
  outline: none;
  background: transparent;
  font-family: var(--dsw-font-family);
  font-size: 15px;
  line-height: 22px;
  color: var(--dsw-alias-label-primary);
}

.input::placeholder {
  color: var(--dsw-alias-label-caption);
}
```

- [ ] **Step 4: 实现 Textarea**

创建 `frontend/src/primitives/Textarea.tsx`：
```tsx
import { forwardRef, useEffect, useImperativeHandle, useRef, type TextareaHTMLAttributes } from 'react';
import styles from './Textarea.module.css';

export type TextareaProps = {
  className?: string;
  autoSize?: { minRows?: number; maxRows?: number };
} & Omit<TextareaHTMLAttributes<HTMLTextAreaElement>, 'className'>;

export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(function Textarea(
  { className, autoSize, ...rest },
  ref,
) {
  const innerRef = useRef<HTMLTextAreaElement | null>(null);

  useImperativeHandle(ref, () => innerRef.current as HTMLTextAreaElement);

  function resize() {
    const el = innerRef.current;
    if (!el) return;
    el.style.height = 'auto';
    const max = autoSize?.maxRows ? autoSize.maxRows * 22 + 24 : undefined;
    const next = Math.min(el.scrollHeight, max ?? el.scrollHeight);
    el.style.height = `${next}px`;
  }

  useEffect(() => {
    resize();
  }, [rest.value]);

  return (
    <textarea
      ref={(el) => {
        innerRef.current = el;
        if (typeof ref === 'function') ref(el);
      }}
      className={`${styles.textarea} ${className ?? ''}`.trim()}
      onInput={resize}
      {...rest}
    />
  );
});
```

创建 `frontend/src/primitives/Textarea.module.css`：
```css
.textarea {
  width: 100%;
  border: none;
  outline: none;
  background: transparent;
  resize: none;
  font-family: var(--dsw-font-family);
  font-size: 16px;
  line-height: 22px;
  color: var(--dsw-alias-label-primary);
  box-sizing: border-box;
}

.textarea::placeholder {
  color: var(--dsw-alias-label-caption);
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/primitives/Input.test.tsx`
Expected: PASS。

- [ ] **Step 6: 提交**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git add frontend/src/primitives/Input.tsx frontend/src/primitives/Input.module.css frontend/src/primitives/Textarea.tsx frontend/src/primitives/Textarea.module.css frontend/src/primitives/Input.test.tsx && git commit -m "feat(frontend): 新增自研 Input/Textarea 组件"
```

---

### Task 8: Modal 组件

**Files:**
- Create: `frontend/src/primitives/Modal.tsx`
- Create: `frontend/src/primitives/Modal.module.css`
- Create: `frontend/src/primitives/Modal.test.tsx`

- [ ] **Step 1: 写失败测试**

创建 `frontend/src/primitives/Modal.test.tsx`：
```tsx
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Modal } from './Modal';

describe('Modal', () => {
  it('open 为 false 时不渲染', () => {
    render(<Modal open={false}>内容</Modal>);
    expect(screen.queryByText('内容')).not.toBeInTheDocument();
  });

  it('open 为 true 时渲染标题与内容', () => {
    render(
      <Modal open title="确认删除" onClose={vi.fn()}>
        内容
      </Modal>,
    );
    expect(screen.getByText('确认删除')).toBeInTheDocument();
    expect(screen.getByText('内容')).toBeInTheDocument();
  });

  it('点击遮罩触发 onClose', async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    render(<Modal open onClose={onClose}>内容</Modal>);
    await user.click(screen.getByRole('presentation', { hidden: true }).firstChild as Element);
    expect(onClose).toHaveBeenCalled();
  });
});
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/primitives/Modal.test.tsx`
Expected: FAIL（找不到 `./Modal`）。

- [ ] **Step 3: 实现 Modal**

创建 `frontend/src/primitives/Modal.tsx`：
```tsx
import { useEffect, type ReactNode } from 'react';
import { createPortal } from 'react-dom';
import styles from './Modal.module.css';
import { CloseIcon } from './icons';

export type ModalProps = {
  open: boolean;
  onClose?: () => void;
  title?: ReactNode;
  description?: ReactNode;
  children?: ReactNode;
  footer?: ReactNode;
};

export function Modal({ open, onClose, title, description, children, footer }: ModalProps) {
  useEffect(() => {
    if (!open) return;
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') onClose?.();
    }
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [open, onClose]);

  if (!open) return null;

  return createPortal(
    <div className={styles.root} role="presentation">
      <div className={styles.mask} onClick={onClose} role="presentation" />
      <div className={styles.dialog} role="dialog" aria-modal="true">
        <div className={styles.header}>
          {title && <h3 className={styles.title}>{title}</h3>}
          {onClose && (
            <button type="button" className={styles.close} onClick={onClose} aria-label="close">
              <CloseIcon />
            </button>
          )}
        </div>
        {description && <p className={styles.description}>{description}</p>}
        {children && <div className={styles.body}>{children}</div>}
        {footer && <div className={styles.footer}>{footer}</div>}
      </div>
    </div>,
    document.body,
  );
}
```

创建 `frontend/src/primitives/Modal.module.css`：
```css
.root {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.mask {
  position: absolute;
  inset: 0;
  background: var(--dsw-alias-bg-mask-1);
  backdrop-filter: var(--dsw-mask-blur);
}

.dialog {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  gap: 20px;
  width: min(380px, 100%);
  padding: 0 0 24px;
  overflow: hidden;
  border: 1px solid var(--dsw-alias-border-inverted);
  border-radius: 24px;
  background: var(--dsw-alias-bg-layer-2);
  box-shadow: var(--dsw-shadow-lv3);
}

.body {
  display: flex;
  flex-direction: column;
  min-width: 0;
  margin-top: 0;
  padding: 0 24px;
}

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 22px 14px 0 24px;
}

.title {
  margin: 0;
  font-size: 16px;
  line-height: 24px;
  font-weight: 600;
  color: var(--dsw-alias-label-primary);
}

.close {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 8px;
  background: transparent;
  cursor: pointer;
  color: var(--dsw-alias-label-secondary);
}
.close:hover {
  background: var(--dsw-alias-interactive-bg-hover);
}

.description {
  margin: 0;
  padding: 0 24px;
  font-size: 14px;
  line-height: 22px;
  color: var(--dsw-alias-label-primary);
}

.footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  padding: 0 24px;
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/primitives/Modal.test.tsx`
Expected: PASS。若「点击遮罩」用例因结构复杂失败，改为在遮罩元素上加 `data-testid="modal-mask"`，测试用 `screen.getByTestId('modal-mask')`。

- [ ] **Step 5: 提交**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git add frontend/src/primitives/Modal.tsx frontend/src/primitives/Modal.module.css frontend/src/primitives/Modal.test.tsx && git commit -m "feat(frontend): 新增自研 Modal 组件"
```

---

### Task 9: Dropdown 与 Tooltip 组件

**Files:**
- Create: `frontend/src/primitives/Dropdown.tsx`
- Create: `frontend/src/primitives/Dropdown.module.css`
- Create: `frontend/src/primitives/Tooltip.tsx`
- Create: `frontend/src/primitives/Tooltip.module.css`
- Create: `frontend/src/primitives/Interaction.test.tsx`

- [ ] **Step 1: 写失败测试**

创建 `frontend/src/primitives/Interaction.test.tsx`：
```tsx
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Dropdown } from './Dropdown';
import { Tooltip } from './Tooltip';

describe('Dropdown', () => {
  it('点击触发项展开菜单，选中项触发 onSelect', async () => {
    const onSelect = vi.fn();
    const user = userEvent.setup();
    render(
      <Dropdown
        trigger={<button>菜单</button>}
        items={[{ key: 'a', label: '选项A', onClick: () => onSelect('a') }]}
      />,
    );
    await user.click(screen.getByRole('button', { name: '菜单' }));
    await user.click(screen.getByText('选项A'));
    expect(onSelect).toHaveBeenCalledWith('a');
  });
});

describe('Tooltip', () => {
  it('渲染 tooltip 文本', () => {
    render(<Tooltip content="提示内容">hover</Tooltip>);
    expect(screen.getByText('提示内容')).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/primitives/Interaction.test.tsx`
Expected: FAIL（找不到模块）。

- [ ] **Step 3: 实现 Dropdown**

创建 `frontend/src/primitives/Dropdown.tsx`：
```tsx
import { useEffect, useRef, useState, type ReactNode } from 'react';
import styles from './Dropdown.module.css';

export type MenuItem = {
  key: string;
  label: ReactNode;
  icon?: ReactNode;
  danger?: boolean;
  disabled?: boolean;
  onClick?: () => void;
};

type DropdownProps = {
  trigger: ReactNode;
  items: MenuItem[];
  align?: 'start' | 'end';
  className?: string;
};

export function Dropdown({ trigger, items, align = 'start', className }: DropdownProps) {
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    function onPointer(e: PointerEvent) {
      if (rootRef.current && !rootRef.current.contains(e.target as Node)) {
        setOpen(false);
      }
    }
    function onKey(e: KeyboardEvent) {
      if (e.key === 'Escape') setOpen(false);
    }
    window.addEventListener('pointerdown', onPointer);
    window.addEventListener('keydown', onKey);
    return () => {
      window.removeEventListener('pointerdown', onPointer);
      window.removeEventListener('keydown', onKey);
    };
  }, [open]);

  return (
    <div className={`${styles.root} ${className ?? ''}`.trim()} ref={rootRef}>
      <div onClick={() => setOpen((v) => !v)}>{trigger}</div>
      {open && (
        <div className={`${styles.list} ${align === 'end' ? styles.alignEnd : ''}`.trim()} role="menu">
          {items.map((item) => (
            <button
              key={item.key}
              type="button"
              role="menuitem"
              className={`${styles.item} ${item.danger ? styles.danger : ''}`.trim()}
              disabled={item.disabled}
              onClick={() => {
                setOpen(false);
                item.onClick?.();
              }}
            >
              {item.icon && <span className={styles.itemIcon}>{item.icon}</span>}
              <span className={styles.itemLabel}>{item.label}</span>
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
```

创建 `frontend/src/primitives/Dropdown.module.css`：
```css
.root {
  position: relative;
  display: inline-flex;
}

.list {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  z-index: 100;
  box-sizing: border-box;
  padding: 4px;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--dsw-alias-border-inverted);
  border-radius: 12px;
  background: var(--dsw-specific-menu);
  box-shadow: var(--dsw-shadow-lv3);
  min-width: 180px;
  max-width: 320px;
}

.alignEnd {
  left: auto;
  right: 0;
}

.item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-height: 40px;
  padding: 8px 10px;
  border: none;
  border-radius: 10px;
  background: transparent;
  cursor: pointer;
  font-family: var(--dsw-font-family);
  font-size: 14px;
  line-height: 22px;
  color: var(--dsw-alias-label-primary);
  text-align: left;
}

.item:hover:not(:disabled) {
  background: var(--dsw-alias-interactive-bg-hover);
}

.item:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.itemIcon {
  display: inline-flex;
  flex: none;
  width: 16px;
  height: 16px;
  align-items: center;
  justify-content: center;
  color: var(--dsw-alias-label-tertiary);
}

.itemLabel {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.danger {
  color: var(--dsw-alias-state-error-primary);
}
.danger .itemIcon {
  color: var(--dsw-alias-state-error-primary);
}
.danger:hover:not(:disabled) {
  background: var(--dsw-alias-interactive-bg-hover-danger);
}
```

- [ ] **Step 4: 实现 Tooltip**

创建 `frontend/src/primitives/Tooltip.tsx`：
```tsx
import { type ReactNode } from 'react';
import styles from './Tooltip.module.css';

export function Tooltip({ content, children }: { content: ReactNode; children: ReactNode }) {
  return (
    <span className={styles.root} data-tooltip={typeof content === 'string' ? content : undefined}>
      {children}
      {typeof content !== 'string' && <span className={styles.bubble}>{content}</span>}
    </span>
  );
}
```

创建 `frontend/src/primitives/Tooltip.module.css`：
```css
.root {
  position: relative;
  display: inline-flex;
}

.root[data-tooltip]::after,
.root .bubble {
  content: attr(data-tooltip);
  position: absolute;
  bottom: calc(100% + 6px);
  left: 50%;
  transform: translateX(-50%) translateY(2px);
  z-index: 300;
  padding: 5px 10px;
  border-radius: 8px;
  background: var(--dsw-alias-tooltip-bg);
  color: var(--dsw-alias-label-primary-inverted);
  font-size: 12px;
  line-height: 18px;
  white-space: nowrap;
  pointer-events: none;
  opacity: 0;
  transition: opacity var(--ds-transition-duration) var(--ds-ease-in-out);
}

.root:hover[data-tooltip]::after,
.root:hover .bubble,
.root:focus-within[data-tooltip]::after,
.root:focus-within .bubble {
  opacity: 1;
  transform: translateX(-50%) translateY(0);
}
```

注意：`content: attr(data-tooltip)` 与 `.bubble` 并存时 `.root::after` 的 `content` 会始终存在（即使 data-tooltip 未定义），故非字符串内容走 `.bubble`；为安全起见在 CSS 中为无 data-tooltip 的 `.root::after` 设 `display: none`，追加：
```css
.root:not([data-tooltip])::after {
  display: none;
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/primitives/Interaction.test.tsx`
Expected: PASS。

- [ ] **Step 6: 提交**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git add frontend/src/primitives/Dropdown.tsx frontend/src/primitives/Dropdown.module.css frontend/src/primitives/Tooltip.tsx frontend/src/primitives/Tooltip.module.css frontend/src/primitives/Interaction.test.tsx && git commit -m "feat(frontend): 新增自研 Dropdown/Tooltip 组件"
```

---

### Task 10: DisclosureRow 与 Pill 组件

**Files:**
- Create: `frontend/src/primitives/DisclosureRow.tsx`
- Create: `frontend/src/primitives/DisclosureRow.module.css`
- Create: `frontend/src/primitives/Pill.tsx`
- Create: `frontend/src/primitives/Pill.module.css`
- Create: `frontend/src/primitives/Disclosure.test.tsx`

- [ ] **Step 1: 写失败测试**

创建 `frontend/src/primitives/Disclosure.test.tsx`：
```tsx
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { DisclosureRow } from './DisclosureRow';
import { Pill } from './Pill';

describe('DisclosureRow', () => {
  it('默认收起，点击展开内容', async () => {
    const user = userEvent.setup();
    render(
      <DisclosureRow title="引用来源">
        <div>来源内容</div>
      </DisclosureRow>,
    );
    expect(screen.queryByText('来源内容')).not.toBeInTheDocument();
    await user.click(screen.getByText('引用来源'));
    expect(screen.getByText('来源内容')).toBeInTheDocument();
  });

  it('controlled expanded 生效', () => {
    render(
      <DisclosureRow title="标题" expanded>
        <div>可见</div>
      </DisclosureRow>,
    );
    expect(screen.getByText('可见')).toBeInTheDocument();
  });
});

describe('Pill', () => {
  it('渲染文本', () => {
    render(<Pill>标签</Pill>);
    expect(screen.getByText('标签')).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/primitives/Disclosure.test.tsx`
Expected: FAIL（找不到模块）。

- [ ] **Step 3: 实现 DisclosureRow**

创建 `frontend/src/primitives/DisclosureRow.tsx`：
```tsx
import { useState, type ReactNode } from 'react';
import styles from './DisclosureRow.module.css';
import { ChevronDownIcon } from './icons';

type DisclosureRowProps = {
  title: ReactNode;
  children: ReactNode;
  defaultOpen?: boolean;
  expanded?: boolean;
  onToggle?: (open: boolean) => void;
};

export function DisclosureRow({ title, children, defaultOpen = false, expanded, onToggle }: DisclosureRowProps) {
  const [innerOpen, setInnerOpen] = useState(defaultOpen);
  const open = expanded ?? innerOpen;

  function toggle() {
    const next = !open;
    setInnerOpen(next);
    onToggle?.(next);
  }

  return (
    <div className={styles.root} data-open={open}>
      <button type="button" className={styles.summary} onClick={toggle} aria-expanded={open}>
        <span className={styles.title}>{title}</span>
        <span className={styles.chevron}>
          <ChevronDownIcon size={14} />
        </span>
      </button>
      {open && <div className={styles.body}>{children}</div>}
    </div>
  );
}
```

创建 `frontend/src/primitives/DisclosureRow.module.css`：
```css
.root {
  width: 100%;
}

.summary {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px 0;
  border: none;
  border-radius: 8px;
  background: transparent;
  cursor: pointer;
  color: var(--dsw-alias-label-secondary);
  font-family: var(--dsw-font-family);
  font-size: 13px;
  line-height: 20px;
  text-align: left;
}

.summary:hover {
  background: var(--dsw-alias-interactive-bg-hover);
}

.title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.chevron {
  flex: none;
  display: inline-flex;
  color: var(--dsw-alias-label-tertiary);
  transition: transform var(--ds-transition-duration) var(--ds-ease-in-out);
}

.root[data-open='true'] .chevron {
  transform: rotate(180deg);
}

.body {
  padding: 4px 0 8px;
}
```

- [ ] **Step 4: 实现 Pill**

创建 `frontend/src/primitives/Pill.tsx`：
```tsx
import { type ReactNode } from 'react';
import styles from './Pill.module.css';

export function Pill({ children, tone }: { children: ReactNode; tone?: 'default' | 'accent' }) {
  return <span className={`${styles.pill} ${tone === 'accent' ? styles.accent : ''}`.trim()}>{children}</span>;
}
```

创建 `frontend/src/primitives/Pill.module.css`：
```css
.pill {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  border-radius: 999px;
  background: var(--dsw-alias-markdown-tag);
  color: var(--dsw-alias-label-secondary);
  font-size: 12px;
  line-height: 18px;
}

.accent {
  background: var(--dsw-static-deepseek-100);
  color: var(--dsw-static-deepseek-600);
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/primitives/Disclosure.test.tsx`
Expected: PASS。

- [ ] **Step 6: 提交**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git add frontend/src/primitives/DisclosureRow.tsx frontend/src/primitives/DisclosureRow.module.css frontend/src/primitives/Pill.tsx frontend/src/primitives/Pill.module.css frontend/src/primitives/Disclosure.test.tsx && git commit -m "feat(frontend): 新增自研 DisclosureRow/Pill 组件"
```

---

### Task 11: AppFrame 三栏布局

**Files:**
- Create: `frontend/src/layout/AppFrame.tsx`
- Create: `frontend/src/layout/AppFrame.module.css`
- Create: `frontend/src/layout/AppFrame.test.tsx`
- Modify: `frontend/src/stores/uiStore.ts`

- [ ] **Step 1: 扩展 uiStore**

编辑 `frontend/src/stores/uiStore.ts`，加入详情栏状态与列宽持久化：

```ts
import { create } from 'zustand';

type UiState = {
  sidebarCollapsed: boolean;
  detailsOpen: boolean;
  sidebarWidth: number;
  detailsWidth: number;
  toggleSidebar: () => void;
  setCollapsed: (v: boolean) => void;
  setDetailsOpen: (v: boolean) => void;
  setSidebarWidth: (v: number) => void;
  setDetailsWidth: (v: number) => void;
};

const SIDEBAR_DEFAULT = 340;
const DETAILS_DEFAULT = 360;

function readNum(key: string, fallback: number): number {
  const raw = localStorage.getItem(key);
  const n = raw ? Number(raw) : NaN;
  return Number.isFinite(n) && n > 0 ? n : fallback;
}

export const useUiStore = create<UiState>((set, get) => ({
  sidebarCollapsed: localStorage.getItem('investguide.sidebarCollapsed') === 'true',
  detailsOpen: localStorage.getItem('investguide.detailsOpen') === 'true',
  sidebarWidth: readNum('investguide.sidebarWidth', SIDEBAR_DEFAULT),
  detailsWidth: readNum('investguide.detailsWidth', DETAILS_DEFAULT),
  toggleSidebar: () => {
    const next = !get().sidebarCollapsed;
    localStorage.setItem('investguide.sidebarCollapsed', String(next));
    set({ sidebarCollapsed: next });
  },
  setCollapsed: (v) => {
    localStorage.setItem('investguide.sidebarCollapsed', String(v));
    set({ sidebarCollapsed: v });
  },
  setDetailsOpen: (v) => {
    localStorage.setItem('investguide.detailsOpen', String(v));
    set({ detailsOpen: v });
  },
  setSidebarWidth: (v) => {
    localStorage.setItem('investguide.sidebarWidth', String(v));
    set({ sidebarWidth: v });
  },
  setDetailsWidth: (v) => {
    localStorage.setItem('investguide.detailsWidth', String(v));
    set({ detailsWidth: v });
  },
}));
```

保留 `uiStore.test.ts` 并新增断言（可选）：
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

  it('详情栏状态持久化', () => {
    useUiStore.getState().setDetailsOpen(true);
    expect(useUiStore.getState().detailsOpen).toBe(true);
    expect(localStorage.getItem('investguide.detailsOpen')).toBe('true');
  });
});
```

- [ ] **Step 2: 写 AppFrame 测试**

创建 `frontend/src/layout/AppFrame.test.tsx`：
```tsx
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { AppFrame } from './AppFrame';

describe('AppFrame', () => {
  it('渲染侧栏/主区/详情三栏内容', () => {
    render(
      <AppFrame sidebar={<div>Sidebar</div>} main={<div>Main</div>} details={<div>Details</div>} />,
    );
    expect(screen.getByText('Sidebar')).toBeInTheDocument();
    expect(screen.getByText('Main')).toBeInTheDocument();
    expect(screen.getByText('Details')).toBeInTheDocument();
  });
});
```

- [ ] **Step 3: 运行测试确认失败**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/layout/AppFrame.test.tsx`
Expected: FAIL（找不到 `./AppFrame`）。

- [ ] **Step 4: 实现 AppFrame**

创建 `frontend/src/layout/AppFrame.tsx`：
```tsx
import { useCallback, useRef, type ReactNode } from 'react';
import { useUiStore } from '@/stores/uiStore';
import styles from './AppFrame.module.css';

type AppFrameProps = {
  sidebar: ReactNode;
  main: ReactNode;
  details?: ReactNode;
};

export function AppFrame({ sidebar, main, details }: AppFrameProps) {
  const sidebarCollapsed = useUiStore((s) => s.sidebarCollapsed);
  const detailsOpen = useUiStore((s) => s.detailsOpen);
  const sidebarWidth = useUiStore((s) => s.sidebarWidth);
  const detailsWidth = useUiStore((s) => s.detailsWidth);
  const setSidebarWidth = useUiStore((s) => s.setSidebarWidth);
  const setDetailsWidth = useUiStore((s) => s.setDetailsWidth);
  const setDetailsOpen = useUiStore((s) => s.setDetailsOpen);

  const draggingRef = useRef<'sidebar' | 'details' | null>(null);

  const onPointerDown = useCallback(
    (side: 'sidebar' | 'details') => (e: React.PointerEvent) => {
      e.preventDefault();
      draggingRef.current = side;
      const startX = e.clientX;
      const startWidth = side === 'sidebar' ? sidebarWidth : detailsWidth;
      const setter = side === 'sidebar' ? setSidebarWidth : setDetailsWidth;

      function onMove(ev: PointerEvent) {
        const delta = ev.clientX - startX;
        const next = side === 'sidebar' ? startWidth + delta : startWidth - delta;
        setter(Math.min(Math.max(next, 200), 560));
      }
      function onUp() {
        draggingRef.current = null;
        window.removeEventListener('pointermove', onMove);
        window.removeEventListener('pointerup', onUp);
      }
      window.addEventListener('pointermove', onMove);
      window.addEventListener('pointerup', onUp);
    },
    [sidebarWidth, detailsWidth, setSidebarWidth, setDetailsWidth],
  );

  return (
    <div
      className={styles.frame}
      data-dragging={draggingRef.current != null}
      style={{
        gridTemplateColumns: `${sidebarCollapsed ? 56 : sidebarWidth}px minmax(0, 1fr) ${
          detailsOpen ? detailsWidth : 0
        }px`,
      }}
    >
      <aside className={styles.sidebarCol}>{sidebar}</aside>
      {!sidebarCollapsed && (
        <div
          className={styles.handle}
          data-side="sidebar"
          role="separator"
          aria-orientation="vertical"
          style={{ left: `${sidebarWidth}px` }}
          onPointerDown={onPointerDown('sidebar')}
        />
      )}
      <main className={styles.centerCol}>{main}</main>
      {detailsOpen && details && (
        <div
          className={styles.handle}
          data-side="details"
          role="separator"
          aria-orientation="vertical"
          style={{ left: `calc(100% - ${detailsWidth}px)` }}
          onPointerDown={onPointerDown('details')}
        />
      )}
      <section className={styles.detailsCol} data-collapsed={!detailsOpen || undefined}>
        {details}
      </section>
    </div>
  );
}
```

创建 `frontend/src/layout/AppFrame.module.css`：
```css
.frame {
  position: relative;
  display: grid;
  grid-template-rows: 100%;
  height: 100%;
  overflow: hidden;
  background: var(--dsw-alias-bg-base);
  transition: grid-template-columns var(--ds-transition-duration-slow) var(--ds-ease-in-out);
}

.frame[data-dragging='true'] {
  transition: none;
}

.sidebarCol {
  min-width: 0;
  overflow: hidden;
  background: var(--dsw-specific-sidebar-fill);
  border-right: 1px solid var(--dsw-alias-border-l1);
}

.centerCol {
  min-width: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.detailsCol {
  min-width: 0;
  overflow: hidden;
  border-left: 1px solid var(--dsw-alias-border-l2);
}

.detailsCol[data-collapsed] {
  border-left: none;
}

.handle {
  position: absolute;
  top: 0;
  bottom: 0;
  width: 8px;
  margin-left: -4px;
  cursor: col-resize;
  z-index: 2;
  touch-action: none;
  transition: left var(--ds-transition-duration-slow) var(--ds-ease-in-out);
}

.frame[data-dragging='true'] .handle {
  transition: none;
}

.handle[data-side='details']::after {
  content: '';
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  width: 12px;
  height: 32px;
  border-radius: 10px;
  box-sizing: border-box;
  background: var(--dsw-alias-button-floating-fill);
  border: 1px solid var(--dsw-alias-border-l2-darkmode-thin);
  opacity: 0;
  transition: opacity var(--ds-transition-duration) var(--ds-ease-in-out);
}

.detailsCol:hover ~ .handle[data-side='details']::after,
.handle[data-side='details']:hover::after {
  opacity: 1;
}

@media (prefers-reduced-motion: reduce) {
  .frame,
  .handle {
    transition: none;
  }
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/layout/AppFrame.test.tsx src/stores/uiStore.test.ts`
Expected: PASS。

- [ ] **Step 6: 提交**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git add frontend/src/layout frontend/src/stores/uiStore.ts frontend/src/stores/uiStore.test.ts && git commit -m "feat(frontend): 新增 AppFrame 三栏布局与拖拽/持久化"
```

---

### Task 12: Sidebar 组件

**Files:**
- Create: `frontend/src/layout/Sidebar.tsx`（替换旧组件）
- Create: `frontend/src/layout/Sidebar.module.css`
- Modify: `frontend/src/layout/Sidebar.test.tsx`

- [ ] **Step 1: 实现 Sidebar**

创建 `frontend/src/layout/Sidebar.tsx`（覆盖原 `components/layout/Sidebar.tsx`，新路径按设计文档放 `layout/`）：

```tsx
import { useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button } from '@/primitives/Button';
import { Modal } from '@/primitives/Modal';
import { Tooltip } from '@/primitives/Tooltip';
import { PlusIcon, MoreIcon, DeleteIcon } from '@/primitives/icons';
import { useConversations } from '@/hooks/useConversations';
import { useConversationStore } from '@/stores/conversationStore';
import { deleteConversation } from '@/api/conversation/conversation';
import type { Conversation } from '@/api/conversation/types';
import { CompassLogo } from '@/theme/logo';
import styles from './Sidebar.module.css';

type GroupKey = 'today' | 'yesterday' | 'earlier';

function groupKeyOf(iso: string, now: Date): GroupKey {
  const d = new Date(iso);
  const startOfDay = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
  const startOfYesterday = startOfDay - 24 * 60 * 60 * 1000;
  const t = d.getTime();
  if (t >= startOfDay) return 'today';
  if (t >= startOfYesterday) return 'yesterday';
  return 'earlier';
}

function groupConversations(
  items: Conversation[],
  now = new Date(),
): Array<{ key: GroupKey; items: Conversation[] }> {
  const groups: Record<GroupKey, Conversation[]> = { today: [], yesterday: [], earlier: [] };
  for (const c of items) {
    groups[groupKeyOf(c.updatedAt, now)].push(c);
  }
  return (Object.keys(groups) as GroupKey[]).map((key) => ({ key, items: groups[key] }));
}

export default function Sidebar() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { id } = useParams();
  const { data, mutate } = useConversations();
  const clearActive = useConversationStore((s) => s.clearActive);
  const [pendingDelete, setPendingDelete] = useState<Conversation | null>(null);

  const groups = useMemo(() => groupConversations(data?.items ?? []), [data]);

  function goHome() {
    clearActive();
    navigate('/');
  }

  async function remove(convId: string) {
    await deleteConversation(convId);
    await mutate();
    if (id === convId) goHome();
    setPendingDelete(null);
  }

  return (
    <div className={styles.root}>
      <div className={styles.logoRow}>
        <button type="button" className={styles.brand} onClick={goHome}>
          <CompassLogo size={24} />
          <span className={styles.brandText}>{t('sidebar.title')}</span>
        </button>
      </div>

      <Button
        variant="outline"
        block
        className={styles.newSession}
        icon={<PlusIcon size={14} />}
        onClick={goHome}
      >
        {t('sidebar.newConversation')}
      </Button>

      <div className={styles.regionArea}>
        {groups.map((group) => (
          <div key={group.key} className={styles.group}>
            <div className={styles.groupLabel}>{t(`sidebar.group.${group.key}`)}</div>
            {group.items.map((conv) => {
              const active = conv.id === id;
              return (
                <div
                  key={conv.id}
                  className={`${styles.item} ${active ? styles.active : ''}`.trim()}
                  data-active={active || undefined}
                  onClick={() => navigate(`/conversations/${conv.id}`)}
                  role="button"
                  tabIndex={0}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') navigate(`/conversations/${conv.id}`);
                  }}
                >
                  <span className={styles.itemTitle}>{conv.title}</span>
                  <Tooltip content={t('common.delete')}>
                    <button
                      type="button"
                      className={styles.moreBtn}
                      onClick={(e) => {
                        e.stopPropagation();
                        setPendingDelete(conv);
                      }}
                      aria-label={t('common.delete')}
                    >
                      <MoreIcon size={14} />
                    </button>
                  </Tooltip>
                </div>
              );
            })}
          </div>
        ))}
      </div>

      <Modal
        open={pendingDelete != null}
        title={t('conversation.list.deleteConfirm')}
        onClose={() => setPendingDelete(null)}
        footer={
          <>
            <Button variant="ghost" onClick={() => setPendingDelete(null)}>
              {t('common.cancel')}
            </Button>
            <Button
              variant="primary"
              icon={<DeleteIcon size={14} />}
              onClick={() => pendingDelete && void remove(pendingDelete.id)}
            >
              {t('common.confirm')}
            </Button>
          </>
        }
      />
    </div>
  );
}
```

创建 `frontend/src/layout/Sidebar.module.css`：
```css
.root {
  --dsh-sidebar-inline-padding: 12px;
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 6px var(--dsh-sidebar-inline-padding);
  box-sizing: border-box;
  background: var(--dsw-specific-sidebar-fill);
  color: var(--dsw-alias-label-primary);
  font-size: 14px;
}

.logoRow {
  flex: none;
  display: flex;
  align-items: center;
  height: 56px;
  padding: 8px 4px;
  box-sizing: border-box;
  overflow: hidden;
}

.brand {
  flex: 1;
  min-width: 0;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 0;
  border: none;
  background: transparent;
  color: inherit;
  cursor: pointer;
}

.brandText {
  font-size: 16px;
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.newSession {
  flex: none;
  height: 38px;
  border-radius: 12px;
  padding: 0 16px;
  margin-bottom: 12px;
  justify-content: center;
  font-weight: 500;
}

.regionArea {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.group {
  margin-bottom: 12px;
}

.groupLabel {
  padding: 4px 8px;
  font-size: 12px;
  font-weight: 500;
  color: var(--dsw-alias-label-tertiary);
}

.item {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 6px 8px;
  border-radius: 10px;
  cursor: pointer;
  color: var(--dsw-alias-label-primary);
  transition: background var(--ds-transition-duration) var(--ds-ease-in-out);
}

.item:hover {
  background: var(--dsw-alias-interactive-bg-hover);
}

.item.active {
  background: var(--dsw-specific-sidebar-nav-item-active);
}

.itemTitle {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
  line-height: 22px;
}

.moreBtn {
  flex: none;
  display: none;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: none;
  border-radius: 6px;
  background: transparent;
  cursor: pointer;
  color: var(--dsw-alias-label-tertiary);
}

.item:hover .moreBtn,
.item:focus-within .moreBtn {
  display: inline-flex;
}

.moreBtn:hover {
  background: var(--dsw-alias-interactive-bg-hover);
  color: var(--dsw-alias-label-secondary);
}
```

- [ ] **Step 2: 更新 Sidebar 测试**

修改 `frontend/src/layout/Sidebar.test.tsx`（保持原断言语义，仅调整导入路径与按钮查询）：

```tsx
import { describe, it, expect, beforeEach, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { SWRConfig } from 'swr';
import { __resetMocks } from '@/api/client';
import { installConversationMocks, __resetConversationData } from '@/api/mock/conversation';
import { installAuthMocks, __resetAuthData } from '@/api/mock/auth';
import { useAuthStore } from '@/stores/authStore';
import * as conversationApi from '@/api/conversation/conversation';
import Sidebar from './Sidebar';

async function login() {
  const { register } = await import('@/api/auth/auth');
  const r = await register({ email: 'a@b.com', password: 'pass1234', displayName: 'A' });
  useAuthStore.getState().login({ token: r.token, user: r.user });
  return r;
}

function renderSidebar() {
  return render(
    <SWRConfig value={{ provider: () => new Map(), dedupingInterval: 0 }}>
      <MemoryRouter initialEntries={['/conversations/abc']}>
        <Routes>
          <Route path="/" element={<div>Sidebar home</div>} />
          <Route path="/conversations/:id" element={<Sidebar />} />
        </Routes>
      </MemoryRouter>
    </SWRConfig>,
  );
}

describe('Sidebar', () => {
  beforeEach(() => {
    localStorage.clear();
    __resetMocks();
    installAuthMocks();
    installConversationMocks();
    __resetAuthData();
    __resetConversationData();
  });

  it('点击新建对话不创建会话，回到首页', async () => {
    await login();
    const createSpy = vi.spyOn(conversationApi, 'createConversation').mockResolvedValue({
      id: 'x',
      title: '',
      country: null,
      createdAt: '',
      updatedAt: '',
    });
    renderSidebar();
    await userEvent.click(screen.getByRole('button', { name: /新建对话/ }));
    await waitFor(() => {
      expect(screen.getByText('Sidebar home')).toBeInTheDocument();
    });
    expect(createSpy).not.toHaveBeenCalled();
    createSpy.mockRestore();
  });
});
```

- [ ] **Step 3: 运行测试确认通过**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/layout/Sidebar.test.tsx`
Expected: PASS。

- [ ] **Step 4: 提交**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git add frontend/src/layout/Sidebar.tsx frontend/src/layout/Sidebar.module.css frontend/src/layout/Sidebar.test.tsx && git commit -m "feat(frontend): 新增自研 Sidebar 组件"
```

---

### Task 13: DetailsPanel 组件

**Files:**
- Create: `frontend/src/layout/DetailsPanel.tsx`
- Create: `frontend/src/layout/DetailsPanel.module.css`
- Create: `frontend/src/layout/DetailsPanel.test.tsx`
- Modify: `frontend/src/stores/conversationStore.ts`

- [ ] **Step 1: 扩展 conversationStore**

编辑 `frontend/src/stores/conversationStore.ts`，加入选中消息与来源高亮状态：

```ts
import { create } from 'zustand';

type ConversationState = {
  activeId: string | null;
  selectedMessageId: string | null;
  highlightSource: number | null;
  setActive: (id: string | null) => void;
  clearActive: () => void;
  setSelectedMessageId: (id: string | null) => void;
  setHighlightSource: (n: number | null) => void;
};

export const useConversationStore = create<ConversationState>((set) => ({
  activeId: null,
  selectedMessageId: null,
  highlightSource: null,
  setActive: (id) => set({ activeId: id }),
  clearActive: () => set({ activeId: null, selectedMessageId: null, highlightSource: null }),
  setSelectedMessageId: (id) => set({ selectedMessageId: id, highlightSource: null }),
  setHighlightSource: (n) => set({ highlightSource: n }),
}));
```

- [ ] **Step 2: 写 DetailsPanel 测试**

创建 `frontend/src/layout/DetailsPanel.test.tsx`：
```tsx
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { DetailsPanel } from './DetailsPanel';
import { useConversationStore } from '@/stores/conversationStore';

describe('DetailsPanel', () => {
  it('无选中消息时显示空态', () => {
    render(<DetailsPanel message={null} />);
    expect(screen.getByText(/引用详情/)).toBeInTheDocument();
  });

  it('展示选中消息的来源与元信息', () => {
    const msg = {
      id: 'm1',
      role: 'assistant' as const,
      content: '回答',
      sources: [{ id: 'c1', title: '来源一', snippet: '片段一' }],
      tokensUsed: 123,
      createdAt: '2026-01-01T00:00:00Z',
    };
    render(<DetailsPanel message={msg} />);
    expect(screen.getByText('来源一')).toBeInTheDocument();
    expect(screen.getByText(/123/)).toBeInTheDocument();
  });
});
```

- [ ] **Step 3: 运行测试确认失败**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/layout/DetailsPanel.test.tsx`
Expected: FAIL（找不到 `./DetailsPanel`）。

- [ ] **Step 4: 实现 DetailsPanel**

创建 `frontend/src/layout/DetailsPanel.tsx`：
```tsx
import { useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import type { Message } from '@/api/conversation/types';
import { useConversationStore } from '@/stores/conversationStore';
import { Pill } from '@/primitives/Pill';
import { DocumentIcon, ClockIcon } from '@/primitives/icons';
import styles from './DetailsPanel.module.css';

export function DetailsPanel({ message }: { message: Message | null }) {
  const { t } = useTranslation();
  const highlightSource = useConversationStore((s) => s.highlightSource);
  const refs = useRef<(HTMLDivElement | null)[]>([]);

  useEffect(() => {
    if (highlightSource == null) return;
    const el = refs.current[highlightSource - 1];
    if (!el) return;
    el.scrollIntoView({ behavior: 'smooth', block: 'center' });
    el.classList.add(styles.highlight);
    const timer = setTimeout(() => el.classList.remove(styles.highlight), 2200);
    return () => clearTimeout(timer);
  }, [highlightSource]);

  if (!message || !message.sources || message.sources.length === 0) {
    return <div className={styles.empty}>{t('details.empty')}</div>;
  }

  const count = message.sources.length;

  return (
    <div className={styles.root}>
      <div className={styles.header}>
        <div className={styles.title}>{t('details.title')}</div>
        <div className={styles.meta}>
          {message.tokensUsed != null && <Pill>{`${t('details.tokens')}: ${message.tokensUsed}`}</Pill>}
          <Pill>{`${count} ${t('message.sources.title')}`}</Pill>
        </div>
      </div>
      <div className={styles.sourceList}>
        {message.sources.map((s, i) => (
          <div
            key={s.id}
            ref={(el) => {
              refs.current[i] = el;
            }}
            className={styles.sourceItem}
          >
            <div className={styles.sourceHead}>
              <span className={styles.sourceIndex}>
                <DocumentIcon size={12} />
                {i + 1}
              </span>
              {s.title && <span className={styles.sourceTitle}>{s.title}</span>}
            </div>
            <div className={styles.sourceSnippet}>{s.snippet}</div>
          </div>
        ))}
      </div>
      <div className={styles.foot}>
        <ClockIcon size={12} />
        <span>{new Date(message.createdAt).toLocaleString()}</span>
      </div>
    </div>
  );
}
```

创建 `frontend/src/layout/DetailsPanel.module.css`：
```css
.root {
  display: flex;
  flex-direction: column;
  height: 100%;
  padding: 16px;
  box-sizing: border-box;
  overflow-y: auto;
  background: var(--dsw-alias-bg-base);
}

.empty {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 16px;
  color: var(--dsw-alias-label-tertiary);
  font-size: 13px;
  text-align: center;
}

.header {
  flex: none;
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 16px;
}

.title {
  font-size: 15px;
  font-weight: 600;
  color: var(--dsw-alias-label-primary);
}

.meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.sourceList {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.sourceItem {
  border: 1px solid var(--dsw-alias-border-l1);
  border-radius: 12px;
  padding: 10px 12px;
  background: var(--dsw-alias-bg-layer-1);
  transition: background var(--ds-transition-duration) var(--ds-ease-in-out);
}

.sourceItem.highlight {
  background: var(--dsw-alias-interactive-bg-hover-accent);
}

.sourceHead {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 6px;
}

.sourceIndex {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 1px 6px;
  border-radius: 6px;
  background: rgba(97, 135, 216, 0.22);
  color: var(--dsw-alias-label-primary);
  font-size: 11px;
  font-weight: 600;
}

.sourceTitle {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 500;
  color: var(--dsw-alias-label-secondary);
}

.sourceSnippet {
  font-size: 13px;
  line-height: 20px;
  color: var(--dsw-alias-label-primary);
  display: -webkit-box;
  -webkit-line-clamp: 4;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.foot {
  flex: none;
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 12px;
  color: var(--dsw-alias-label-tertiary);
  font-size: 12px;
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/layout/DetailsPanel.test.tsx`
Expected: PASS。

- [ ] **Step 6: 提交**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git add frontend/src/layout/DetailsPanel.tsx frontend/src/layout/DetailsPanel.module.css frontend/src/layout/DetailsPanel.test.tsx frontend/src/stores/conversationStore.ts && git commit -m "feat(frontend): 新增 DetailsPanel 详情栏组件"
```

---

### Task 14: UserMenu 组件

**Files:**
- Create: `frontend/src/layout/UserMenu.tsx`（替换旧组件）
- Create: `frontend/src/layout/UserMenu.module.css`
- Modify: `frontend/src/layout/AppLayout.tsx`（重建为 AppFrame 编排）

- [ ] **Step 1: 实现 UserMenu**

创建 `frontend/src/layout/UserMenu.tsx`：
```tsx
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Dropdown } from '@/primitives/Dropdown';
import { LogoutIcon, MoonIcon, SunIcon } from '@/primitives/icons';
import { useAuthStore } from '@/stores/authStore';
import { useThemeStore } from '@/theme/themeStore';
import styles from './UserMenu.module.css';

export default function UserMenu() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const user = useAuthStore((s) => s.user);
  const logout = useAuthStore((s) => s.logout);
  const mode = useThemeStore((s) => s.mode);
  const toggleTheme = useThemeStore((s) => s.toggle);

  return (
    <Dropdown
      align="end"
      trigger={
        <button type="button" className={styles.trigger}>
          <span className={styles.avatar}>{(user?.displayName ?? '?').slice(0, 1)}</span>
          <span className={styles.name}>{user?.displayName}</span>
        </button>
      }
      items={[
        {
          key: 'theme',
          icon: mode === 'dark' ? <SunIcon size={14} /> : <MoonIcon size={14} />,
          label: t('sidebar.userMenu.theme'),
          onClick: toggleTheme,
        },
        {
          key: 'logout',
          icon: <LogoutIcon size={14} />,
          label: t('sidebar.userMenu.logout'),
          danger: true,
          onClick: () => {
            logout();
            navigate('/login', { replace: true });
          },
        },
      ]}
    />
  );
}
```

创建 `frontend/src/layout/UserMenu.module.css`：
```css
.trigger {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px;
  border: none;
  border-radius: 10px;
  background: transparent;
  cursor: pointer;
  color: var(--dsw-alias-label-primary);
  font-family: var(--dsw-font-family);
  font-size: 14px;
}

.trigger:hover {
  background: var(--dsw-alias-interactive-bg-hover);
}

.avatar {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--dsw-alias-button-info-fill);
  color: var(--dsw-alias-label-primary-foreground);
  font-size: 13px;
  font-weight: 600;
}

.name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-align: left;
}
```

- [ ] **Step 2: 重建 AppLayout**

创建 `frontend/src/layout/AppLayout.tsx`（覆盖旧组件）：
```tsx
import { Outlet, useParams } from 'react-router-dom';
import { useConversation } from '@/hooks/useConversation';
import { useConversationStore } from '@/stores/conversationStore';
import { AppFrame } from './AppFrame';
import Sidebar from './Sidebar';
import DetailsPanel from './DetailsPanel';
import UserMenu from './UserMenu';

export default function AppLayout() {
  const { id } = useParams();
  const selectedMessageId = useConversationStore((s) => s.selectedMessageId);
  const { conv, messages } = useConversation(id ?? null);

  const items = messages.data?.items ?? [];
  const selected = items.find((m) => m.id === selectedMessageId) ?? null;

  return (
    <AppFrame
      sidebar={
        <>
          <div className="min-h-0 flex-1">
            <Sidebar />
          </div>
          <div style={{ borderTop: '1px solid var(--dsw-alias-border-l1)', padding: 6 }}>
            <UserMenu />
          </div>
        </>
      }
      main={<Outlet context={{ conv, selected }} />}
      details={<DetailsPanel message={selected} />}
    />
  );
}
```

注意：此 AppLayout 使用 `useConversation` 在布局层获取消息，页面（ConversationPage）已有同样请求，SWR 会去重（同 key）。为遵循「功能逻辑不变」，ConversationPage 继续自管数据；布局层的 `messages` 仅用于详情栏选中消息查找。若担忧重复请求，可将 `useConversation` 保留在页面，详情栏通过 store 存整条 message（推荐方案见下 Step 3）。

- [ ] **Step 3: （推荐）改用 store 存整条选中消息，避免布局层重复请求**

调整 `conversationStore`（覆盖 Step 1 定义）：

```ts
import { create } from 'zustand';
import type { Message } from '@/api/conversation/types';

type ConversationState = {
  activeId: string | null;
  selectedMessage: Message | null;
  highlightSource: number | null;
  setActive: (id: string | null) => void;
  clearActive: () => void;
  setSelectedMessage: (m: Message | null) => void;
  setHighlightSource: (n: number | null) => void;
};

export const useConversationStore = create<ConversationState>((set) => ({
  activeId: null,
  selectedMessage: null,
  highlightSource: null,
  setActive: (id) => set({ activeId: id }),
  clearActive: () => set({ activeId: null, selectedMessage: null, highlightSource: null }),
  setSelectedMessage: (m) => set({ selectedMessage: m, highlightSource: null }),
  setHighlightSource: (n) => set({ highlightSource: n }),
}));
```

相应修改 `DetailsPanel` 与测试：`DetailsPanel` 内部通过 store 读取 `selectedMessage`，组件不再接收 `message` prop。AppLayout 简化为：

```tsx
import { Outlet } from 'react-router-dom';
import { AppFrame } from './AppFrame';
import Sidebar from './Sidebar';
import DetailsPanel from './DetailsPanel';
import UserMenu from './UserMenu';

export default function AppLayout() {
  return (
    <AppFrame
      sidebar={
        <>
          <div style={{ minHeight: 0, flex: 1 }}>
            <Sidebar />
          </div>
          <div style={{ borderTop: '1px solid var(--dsw-alias-border-l1)', padding: 6 }}>
            <UserMenu />
          </div>
        </>
      }
      main={<Outlet />}
      details={<DetailsPanel />}
    />
  );
}
```

（侧栏占位使用内联样式是为了跨组件共享 token；如需 CSS Modules 由 `AppLayout.module.css` 承载。）

修改 `DetailsPanel.tsx`：删除 `message` prop，改为：
```tsx
const selectedMessage = useConversationStore((s) => s.selectedMessage);
const message = selectedMessage;
```
其余渲染逻辑不变。同步修改 `DetailsPanel.test.tsx`：用 `useConversationStore.getState().setSelectedMessage(msg)` 设置选中后再渲染 `<DetailsPanel />`。

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/layout/DetailsPanel.test.tsx src/layout/Sidebar.test.tsx src/layout/AppFrame.test.tsx`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git add frontend/src/layout/UserMenu.tsx frontend/src/layout/UserMenu.module.css frontend/src/layout/AppLayout.tsx frontend/src/layout/DetailsPanel.tsx frontend/src/layout/DetailsPanel.test.tsx frontend/src/stores/conversationStore.ts && git commit -m "feat(frontend): 新增 UserMenu 并重构 AppLayout 为 AppFrame 编排"
```

---

### Task 15: 首页 Composer 与 HomePage

**Files:**
- Create: `frontend/src/features/home/HomeComposer.tsx`
- Create: `frontend/src/features/home/HomeComposer.module.css`
- Create: `frontend/src/pages/HomePage.tsx`（重写）
- Modify: `frontend/src/pages/HomePage.test.tsx`

- [ ] **Step 1: 实现 HomeComposer**

创建 `frontend/src/features/home/HomeComposer.tsx`：
```tsx
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button } from '@/primitives/Button';
import { Textarea } from '@/primitives/Textarea';
import { SendIcon, StopIcon } from '@/primitives/icons';
import { createConversation, sendMessage } from '@/api/conversation/conversation';
import { useConversationStore } from '@/stores/conversationStore';
import { useToast } from '@/primitives/ToastProvider';
import styles from './HomeComposer.module.css';

const MAX_LEN = 2000;

export function HomeComposer() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const toast = useToast();
  const setActive = useConversationStore((s) => s.setActive);
  const [value, setValue] = useState('');
  const [loading, setLoading] = useState(false);

  async function ask() {
    const content = value.trim();
    if (!content || loading) return;
    if (content.length > MAX_LEN) {
      toast.error(t('composer.tooLong'));
      return;
    }
    setLoading(true);
    try {
      const conv = await createConversation({});
      setActive(conv.id);
      const { messageId } = await sendMessage(conv.id, { content });
      navigate(`/conversations/${conv.id}`, { state: { pendingMessageId: messageId } });
    } catch {
      toast.error(t('error.generic'));
      setLoading(false);
    }
  }

  return (
    <div className={styles.card}>
      <Textarea
        value={value}
        onChange={(e) => setValue(e.target.value)}
        placeholder={t('composer.placeholder')}
        autoSize={{ minRows: 3, maxRows: 8 }}
        disabled={loading}
        onKeyDown={(e) => {
          if (e.key === 'Enter' && !e.shiftKey) {
            e.preventDefault();
            void ask();
          }
        }}
      />
      <div className={styles.actions}>
        {loading ? (
          <Button variant="primary" size="sm" icon={<StopIcon size={14} />} onClick={() => setLoading(false)}>
            {t('common.stop')}
          </Button>
        ) : (
          <Button
            variant="primary"
            size="sm"
            icon={<SendIcon size={14} />}
            disabled={!value.trim()}
            onClick={() => void ask()}
          >
            {t('common.send')}
          </Button>
        )}
      </div>
    </div>
  );
}
```

创建 `frontend/src/features/home/HomeComposer.module.css`：
```css
.card {
  width: 100%;
  border: 1px solid var(--dsw-alias-border-l2);
  border-radius: 16px;
  background: var(--dsw-specific-input-major);
  padding: 14px 16px;
  box-sizing: border-box;
  box-shadow: var(--dsw-shadow-lv1);
  transition:
    border-color var(--ds-transition-duration) var(--ds-ease-in-out),
    box-shadow var(--ds-transition-duration) var(--ds-ease-in-out);
}

.card:focus-within {
  border-color: var(--dsw-alias-brand-primary-new-colorprimary-new-color);
  box-shadow: 0 0 0 3px var(--dsw-alias-interactive-bg-hover-accent);
}

.actions {
  display: flex;
  justify-content: flex-end;
  padding-top: 10px;
}
```

- [ ] **Step 2: 重写 HomePage**

创建 `frontend/src/pages/HomePage.tsx`（覆盖）：
```tsx
import { useTranslation } from 'react-i18next';
import { CompassLogo } from '@/theme/logo';
import { HomeComposer } from '@/features/home/HomeComposer';
import styles from './HomePage.module.css';

export default function HomePage() {
  const { t } = useTranslation();
  return (
    <div className={styles.root}>
      <div className={styles.hero}>
        <div className={styles.logo}>
          <CompassLogo size={52} />
        </div>
        <h1 className={styles.title}>{t('home.welcome')}</h1>
        <p className={styles.subtitle}>{t('home.subtitle')}</p>
        <p className={styles.tagline}>{t('home.tagline')}</p>
      </div>
      <div className={styles.composer}>
        <HomeComposer />
      </div>
    </div>
  );
}
```

创建 `frontend/src/pages/HomePage.module.css`：
```css
.root {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  padding: 24px;
  box-sizing: border-box;
  gap: 28px;
}

.hero {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}

.logo {
  margin-bottom: 16px;
}

.title {
  margin: 0 0 8px;
  font-size: 28px;
  font-weight: 600;
  letter-spacing: -0.3px;
  color: var(--dsw-alias-label-primary);
}

.subtitle {
  margin: 0 0 4px;
  font-size: 15px;
  color: var(--dsw-alias-label-secondary);
}

.tagline {
  margin: 0;
  font-size: 13px;
  color: var(--dsw-alias-label-tertiary);
}

.composer {
  width: 100%;
  max-width: 720px;
}
```

- [ ] **Step 3: 更新 HomePage 测试**

修改 `frontend/src/pages/HomePage.test.tsx`，移除 antd `AntdApp` 包裹、改用 `ToastProvider`：

```tsx
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { ToastProvider } from '@/primitives/ToastProvider';
import HomePage from './HomePage';

const navigateMock = vi.fn();

vi.mock('react-router-dom', async (importOriginal) => {
  const actual = await importOriginal<typeof import('react-router-dom')>();
  return { ...actual, useNavigate: () => navigateMock };
});

vi.mock('@/api/conversation/conversation', () => ({
  createConversation: vi.fn(),
  sendMessage: vi.fn(),
}));

import { createConversation, sendMessage } from '@/api/conversation/conversation';

describe('HomePage', () => {
  beforeEach(() => {
    navigateMock.mockReset();
    vi.mocked(createConversation).mockReset();
    vi.mocked(sendMessage).mockReset();
  });

  it('从首页提问时携带 pendingMessageId 跳转到会话页', async () => {
    const user = userEvent.setup();
    vi.mocked(createConversation).mockResolvedValue({ id: 'conv-1' } as never);
    vi.mocked(sendMessage).mockResolvedValue({ messageId: 'msg-1' } as never);

    render(
      <ToastProvider>
        <MemoryRouter>
          <HomePage />
        </MemoryRouter>
      </ToastProvider>,
    );

    await user.type(screen.getByRole('textbox'), '我想去巴基斯坦投资');
    await user.click(screen.getByRole('button', { name: /发\s*送/ }));

    await vi.waitFor(() => {
      expect(createConversation).toHaveBeenCalledTimes(1);
    });
    expect(sendMessage).toHaveBeenCalledWith('conv-1', { content: '我想去巴基斯坦投资' });
    expect(navigateMock).toHaveBeenCalledWith('/conversations/conv-1', {
      state: { pendingMessageId: 'msg-1' },
    });
  });
});
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/pages/HomePage.test.tsx src/features/home/HomeComposer.tsx`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git add frontend/src/features/home frontend/src/pages/HomePage.tsx frontend/src/pages/HomePage.test.tsx && git commit -m "feat(frontend): 重写首页与 HomeComposer"
```

---

### Task 16: MarkdownRenderer 与 SourceChip 重绘

**Files:**
- Modify: `frontend/src/components/conversation/MarkdownRenderer.tsx`
- Modify: `frontend/src/components/conversation/SourceChip.tsx`
- Create: `frontend/src/components/conversation/SourceChip.module.css`
- Modify: `frontend/src/components/conversation/MarkdownRenderer.test.tsx`

- [ ] **Step 1: 重写 SourceChip（去掉 antd Popover）**

创建 `frontend/src/components/conversation/SourceChip.tsx`（覆盖）：
```tsx
import type { KnowledgeChunkRef } from '@/api/conversation/types';
import styles from './SourceChip.module.css';

type Props = {
  n: number;
  source: KnowledgeChunkRef | undefined;
  onSourceRef?: (n: number) => void;
};

export default function SourceChip({ n, source, onSourceRef }: Props) {
  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();
    onSourceRef?.(n);
  };

  return (
    <span
      className="source-chip"
      onClick={handleClick}
      role="button"
      tabIndex={0}
      title={source?.title ?? `来源 ${n}`}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          handleClick(e as unknown as React.MouseEvent);
        }
      }}
    >
      {n}
    </span>
  );
}
```

`SourceChip.module.css` 可为空骨架（芯片样式在 main.css 的 `.source-chip`）：
```css
/* 来源芯片样式位于 styles/main.css 的 .source-chip，此处保留模块命名空间 */
```

- [ ] **Step 2: 更新 MarkdownRenderer**

编辑 `frontend/src/components/conversation/MarkdownRenderer.tsx`，仅保留与 DeepSeek 一致的渲染（markup 不变，样式交给 main.css）。代码无需改动（它不依赖 antd）；确认 `a` 组件 `href` 匹配逻辑与 `linkSourceRefs` 不变。

- [ ] **Step 3: 更新测试**

`frontend/src/components/conversation/MarkdownRenderer.test.tsx` 中「传入 sources 时芯片可显示 Popover 来源内容」用例不再适用（已移除 Popover）。修改为仅断言芯片渲染与 `onSourceRef`：

```tsx
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { MarkdownRenderer } from './MarkdownRenderer';
import { linkSourceRefs } from '@/utils/sourceRefs';

describe('MarkdownRenderer', () => {
  it('linkSourceRefs 把片段[N]转义为指向来源的数字链接', () => {
    expect(linkSourceRefs('参考片段[1]与片段[2]内容')).toBe('参考[1](#src-1)与[2](#src-2)内容');
    expect(linkSourceRefs('无引用')).toBe('无引用');
  });

  it('linkSourceRefs 兼容【N】引用格式', () => {
    expect(linkSourceRefs('建议加强防范。【3】')).toBe('建议加强防范。[3](#src-3)');
    expect(linkSourceRefs('参考片段【1】')).toBe('参考[1](#src-1)');
  });

  it('linkSourceRefs 处理 "（来源：片段[N]）" 完整格式，并吃掉前置换行', () => {
    expect(linkSourceRefs('注意安全。（来源：片段[1]）')).toBe('注意安全。[1](#src-1)');
    expect(linkSourceRefs('做好评估。\n（来源：片段【2】）')).toBe('做好评估。[2](#src-2)');
  });

  it('渲染标题与列表', () => {
    const { container } = render(<MarkdownRenderer content={'# 标题\n- a\n- b'} />);
    expect(container.querySelector('h1')).not.toBeNull();
    expect(container.querySelectorAll('li')).toHaveLength(2);
  });

  it('不渲染内联 HTML / script', () => {
    const { container } = render(<MarkdownRenderer content={'<script>alert(1)</script>'} />);
    expect(container.querySelector('script')).toBeNull();
  });

  it('把「片段[N]」渲染为来源芯片', () => {
    const { container } = render(<MarkdownRenderer content={'参考片段[1]与片段[2]内容'} />);
    const chips = container.querySelectorAll('.source-chip');
    expect(chips).toHaveLength(2);
    expect(chips[0]).toHaveTextContent('1');
    expect(chips[1]).toHaveTextContent('2');
  });

  it('点击来源芯片触发 onSourceRef', () => {
    const onSourceRef = vi.fn();
    render(<MarkdownRenderer content={'参考片段[3]'} onSourceRef={onSourceRef} />);
    fireEvent.click(screen.getByText('3'));
    expect(onSourceRef).toHaveBeenCalledWith(3);
  });

  it('传入 sources 时芯片可渲染', () => {
    const sources = [{ id: 'c1', title: '测试来源', snippet: '这是测试片段内容' }];
    render(<MarkdownRenderer content={'参考片段[1]'} sources={sources} />);
    expect(screen.getByText('1')).toBeInTheDocument();
  });
});
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/components/conversation/MarkdownRenderer.test.tsx`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git add frontend/src/components/conversation/SourceChip.tsx frontend/src/components/conversation/SourceChip.module.css frontend/src/components/conversation/MarkdownRenderer.test.tsx && git commit -m "refactor(frontend): SourceChip 去除 antd Popover，MarkdownRenderer 保持渲染"
```

---

### Task 17: SourcesCard 重绘（DisclosureRow）

**Files:**
- Create: `frontend/src/components/conversation/SourcesCard.tsx`（覆盖）
- Create: `frontend/src/components/conversation/SourcesCard.module.css`
- Modify: `frontend/src/components/conversation/MessageBubble.tsx`

- [ ] **Step 1: 重写 SourcesCard**

创建 `frontend/src/components/conversation/SourcesCard.tsx`（覆盖，去掉 antd Collapse/Empty）：
```tsx
import { useTranslation } from 'react-i18next';
import { DisclosureRow } from '@/primitives/DisclosureRow';
import { DocumentIcon } from '@/primitives/icons';
import type { KnowledgeChunkRef } from '@/api/conversation/types';
import styles from './SourcesCard.module.css';

type Props = {
  sources: KnowledgeChunkRef[] | null;
  expanded?: boolean;
  onToggle?: (expanded: boolean) => void;
  registerSourceRef?: (index: number, el: HTMLDivElement | null) => void;
};

export default function SourcesCard({ sources, expanded, onToggle, registerSourceRef }: Props) {
  const { t } = useTranslation();
  if (!sources || sources.length === 0) {
    return <div className={styles.empty}>{t('message.sources.empty')}</div>;
  }
  return (
    <DisclosureRow
      title={
        <span className={styles.label}>
          <DocumentIcon size={13} />
          {t('message.sources.title')} · {sources[0]?.title ?? ''}
        </span>
      }
      expanded={expanded}
      onToggle={onToggle}
    >
      <div className={styles.list}>
        {sources.map((s, i) => (
          <div
            key={s.id}
            id={`src-${i + 1}`}
            ref={(el) => registerSourceRef?.(i, el)}
            className={styles.item}
          >
            <div className={styles.itemHead}>
              <span className={styles.index}>{i + 1}</span>
              {s.title && <span className={styles.itemTitle}>{s.title}</span>}
            </div>
            <div className={styles.snippet}>{s.snippet}</div>
          </div>
        ))}
      </div>
    </DisclosureRow>
  );
}
```

创建 `frontend/src/components/conversation/SourcesCard.module.css`：
```css
.empty {
  color: var(--dsw-alias-label-tertiary);
  font-size: 13px;
  padding: 8px 0;
}

.label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  color: var(--dsw-alias-label-secondary);
}

.list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 8px 0;
}

.item {
  border: 1px solid var(--dsw-alias-border-l1);
  border-radius: 10px;
  padding: 8px 10px;
  background: var(--dsw-alias-bg-layer-1);
  transition: background var(--ds-transition-duration) var(--ds-ease-in-out);
}

.itemHead {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}

.index {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding: 0 4px;
  border-radius: 6px;
  background: rgba(97, 135, 216, 0.22);
  color: var(--dsw-alias-label-primary);
  font-size: 11px;
  font-weight: 600;
}

.itemTitle {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 13px;
  font-weight: 500;
  color: var(--dsw-alias-label-secondary);
}

.snippet {
  font-size: 13px;
  line-height: 20px;
  color: var(--dsw-alias-label-primary);
}
```

- [ ] **Step 2: 更新 MessageBubble**

编辑 `frontend/src/components/conversation/MessageBubble.tsx`：
- 移除对 `handleSourceRef` 的滚动逻辑中对 `SourcesCard` 内联展开的强制依赖，改为调用 store 的 `setSelectedMessage` + `setHighlightSource`（联动右栏）：
- 保留 `sourcesExpanded` 本地状态用于内联卡展开；当右栏未展开时点击芯片回退展开内联卡。

```tsx
import { useCallback, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { SparkleIcon } from '@/primitives/icons';
import { useConversationStore } from '@/stores/conversationStore';
import type { Message } from '@/api/conversation/types';
import { MarkdownRenderer } from './MarkdownRenderer';
import SourcesCard from './SourcesCard';
import styles from './MessageBubble.module.css';

export default function MessageBubble({
  message,
  streaming,
}: {
  message: Pick<Message, 'id' | 'role' | 'content' | 'sources'>;
  streaming?: boolean;
}) {
  const { t } = useTranslation();
  const isUser = message.role === 'user';
  const [sourcesExpanded, setSourcesExpanded] = useState(false);
  const sourceRefs = useRef<(HTMLDivElement | null)[]>([]);
  const setSelectedMessage = useConversationStore((s) => s.setSelectedMessage);
  const setHighlightSource = useConversationStore((s) => s.setHighlightSource);

  const handleSourceRef = useCallback(
    (n: number) => {
      setSelectedMessage(message as Message);
      setHighlightSource(n);
      setSourcesExpanded(true);
      requestAnimationFrame(() => {
        const el = sourceRefs.current[n - 1];
        if (!el) return;
        el.scrollIntoView({ behavior: 'smooth', block: 'center' });
        el.classList.add('source-highlight');
        setTimeout(() => el.classList.remove('source-highlight'), 2200);
      });
    },
    [message, setSelectedMessage, setHighlightSource],
  );

  if (isUser) {
    return (
      <div className={styles.userRow}>
        <div className={styles.userBubble}>{message.content}</div>
      </div>
    );
  }

  return (
    <div className={styles.assistantRow}>
      <div className={styles.avatar}>
        <SparkleIcon size={15} />
      </div>
      <div className={styles.assistantBody}>
        <div className="md-body">
          <MarkdownRenderer
            content={message.content + (streaming ? t('message.streaming.cursor') : '')}
            sources={message.sources}
            onSourceRef={handleSourceRef}
          />
        </div>
        {message.sources && message.sources.length > 0 && (
          <div className={styles.sources}>
            <SourcesCard
              sources={message.sources}
              expanded={sourcesExpanded}
              onToggle={setSourcesExpanded}
              registerSourceRef={(i, el) => {
                sourceRefs.current[i] = el;
              }}
            />
          </div>
        )}
      </div>
    </div>
  );
}
```

创建 `frontend/src/components/conversation/MessageBubble.module.css`：
```css
.userRow {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
}

.userBubble {
  max-width: min(525px, 82%);
  background: var(--dsw-specific-bubble);
  border-radius: 22px;
  padding: 10px 16px;
  font-size: 16px;
  line-height: 24px;
  color: var(--dsw-alias-label-primary);
  white-space: pre-wrap;
  word-break: break-word;
}

.assistantRow {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.avatar {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  margin-top: 2px;
  border-radius: 10px;
  background: linear-gradient(135deg, var(--dsw-static-deepseek-400), var(--dsw-static-deepseek-500));
  color: #fff;
}

.assistantBody {
  flex: 1;
  min-width: 0;
}

.sources {
  margin-top: 12px;
}
```

- [ ] **Step 3: 运行测试确认通过**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/components/conversation/MessageList.test.tsx`
Expected: PASS（MessageList 依赖 MessageBubble/SourcesCard 的「引用来源」文本，仍由 i18n `message.sources.title` 提供）。

- [ ] **Step 4: 提交**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git add frontend/src/components/conversation/SourcesCard.tsx frontend/src/components/conversation/SourcesCard.module.css frontend/src/components/conversation/MessageBubble.tsx frontend/src/components/conversation/MessageBubble.module.css && git commit -m "refactor(frontend): SourcesCard 改用 DisclosureRow，MessageBubble 重绘"
```

---

### Task 18: MessageList 重绘

**Files:**
- Create: `frontend/src/components/conversation/MessageList.tsx`（覆盖）
- Create: `frontend/src/components/conversation/MessageList.module.css`
- Create: `frontend/src/features/conversation/TurnStatus.tsx`（流式状态行）

- [ ] **Step 1: 实现 MessageList**

创建 `frontend/src/components/conversation/MessageList.tsx`（覆盖，去掉 Tailwind 类）：
```tsx
import { useEffect, useRef } from 'react';
import type { Message } from '@/api/conversation/types';
import MessageBubble from './MessageBubble';
import styles from './MessageList.module.css';

export default function MessageList({
  messages,
  streamingId,
}: {
  messages: Message[];
  streamingId?: string;
}) {
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
    <div ref={containerRef} onScroll={onScroll} className={styles.scroll}>
      <div className={styles.column}>
        {messages.map((m) => (
          <MessageBubble key={m.id} message={m} streaming={m.id === streamingId} />
        ))}
      </div>
      <div className={styles.column} ref={bottomRef} />
    </div>
  );
}
```

创建 `frontend/src/components/conversation/MessageList.module.css`：
```css
.scroll {
  flex: 1 1 auto;
  min-height: 0;
  overflow-y: auto;
  padding: 16px 24px;
}

.column {
  max-width: var(--dsh-chat-content-width);
  width: 100%;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 20px;
}
```

- [ ] **Step 2: 实现 TurnStatus**

创建 `frontend/src/features/conversation/TurnStatus.tsx`：
```tsx
import { useTranslation } from 'react-i18next';
import styles from './TurnStatus.module.css';

export function TurnStatus() {
  const { t } = useTranslation();
  return <div className={styles.status}>{t('message.streaming.pending')}</div>;
}
```

创建 `frontend/src/features/conversation/TurnStatus.module.css`：
```css
.status {
  align-self: flex-start;
  flex: none;
  display: inline-flex;
  align-items: center;
  height: 26px;
  font-size: 14px;
  font-weight: 600;
  white-space: nowrap;
  background: linear-gradient(
    90deg,
    var(--dsw-static-deepseek-500) 0%,
    var(--dsw-static-deepseek-500) 40%,
    var(--dsw-static-deepseek-200) 50%,
    var(--dsw-static-deepseek-500) 60%,
    var(--dsw-static-deepseek-500) 100%
  );
  background-position: 100% 0;
  background-size: 250% 100%;
  background-clip: text;
  color: transparent;
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  animation: dsh-turn-shimmer 1.8s linear infinite;
}

@keyframes dsh-turn-shimmer {
  to {
    background-position: 0 0;
  }
}

@media (prefers-reduced-motion: reduce) {
  .status {
    background-position: 0 0;
    background-size: 100% 100%;
    animation: none;
  }
}
```

- [ ] **Step 3: 运行测试确认通过**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/components/conversation/MessageList.test.tsx`
Expected: PASS。

- [ ] **Step 4: 提交**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git add frontend/src/components/conversation/MessageList.tsx frontend/src/components/conversation/MessageList.module.css frontend/src/features/conversation/TurnStatus.tsx frontend/src/features/conversation/TurnStatus.module.css && git commit -m "refactor(frontend): 重绘 MessageList 并新增流式状态行"
```

---

### Task 19: MessageComposer 重绘

**Files:**
- Create: `frontend/src/components/conversation/MessageComposer.tsx`（覆盖）
- Create: `frontend/src/components/conversation/MessageComposer.module.css`
- Modify: `frontend/src/components/conversation/MessageComposer.test.tsx`

- [ ] **Step 1: 重写 MessageComposer**

创建 `frontend/src/components/conversation/MessageComposer.tsx`（覆盖，逻辑不变、替换 antd 控件）：
```tsx
import { useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/primitives/Button';
import { Textarea } from '@/primitives/Textarea';
import { SendIcon, StopIcon } from '@/primitives/icons';
import { sendMessage as apiSendMessage } from '@/api/conversation/conversation';
import { useSSEStream, type SSEEvent } from '@/hooks/useSSEStream';
import type { Message } from '@/api/conversation/types';
import styles from './MessageComposer.module.css';

type MessagesLike = { items: Message[] };
type MutateFn = (
  updater?: ((cur?: MessagesLike) => MessagesLike) | MessagesLike,
  opts?: { revalidate?: boolean },
) => Promise<unknown>;

type Props = {
  conversationId: string;
  messages: MessagesLike;
  mutateMessages: MutateFn;
  onStreamingChange?: (id: string | null) => void;
  initialStreamMessageId?: string | null;
};

const MAX_LEN = 2000;

export function MessageComposer({
  conversationId,
  messages,
  mutateMessages,
  onStreamingChange,
  initialStreamMessageId,
}: Props) {
  const { t } = useTranslation();
  const [value, setValue] = useState('');
  const [activeMessageId, setActiveMessageId] = useState<string | null>(
    initialStreamMessageId ?? null,
  );
  const [errorReason, setErrorReason] = useState<string | null>(null);
  const lastUserContent = useRef('');

  function setActive(id: string | null) {
    setActiveMessageId(id);
    onStreamingChange?.(id);
  }

  function onEvent(e: SSEEvent) {
    if (e.type === 'heartbeat') return;
    if (e.type === 'sources') {
      void mutateMessages(
        (cur) => {
          const items = cur?.items ?? [];
          const idx = items.findIndex((m) => m.id === activeMessageId);
          if (idx === -1) return cur ?? { items };
          const copy = [...items];
          copy[idx] = { ...copy[idx]!, sources: e.chunks };
          return { items: copy };
        },
        { revalidate: false },
      );
      return;
    }
    if (e.type === 'message') {
      void mutateMessages(
        (cur) => {
          const items = cur?.items ?? [];
          const idx = items.findIndex((m) => m.id === activeMessageId);
          if (idx === -1) return cur ?? { items };
          const copy = [...items];
          copy[idx] = { ...copy[idx]!, content: copy[idx]!.content + e.delta };
          return { items: copy };
        },
        { revalidate: false },
      );
    } else if (e.type === 'done') {
      setActive(null);
    } else if (e.type === 'error') {
      setErrorReason(e.message);
      setActive(null);
    }
  }

  const streamReady = !!activeMessageId && messages.items.some((m) => m.id === activeMessageId);
  const { state, stop } = useSSEStream({
    convId: conversationId,
    messageId: activeMessageId ?? '',
    enabled: streamReady,
    onEvent,
  });

  async function sendContent(content: string) {
    if (!content) return;
    if (content.length > MAX_LEN) {
      setErrorReason(t('composer.tooLong'));
      return;
    }
    setValue('');
    setErrorReason(null);
    lastUserContent.current = content;

    const tempUserId = 'pending_user_' + Date.now();
    const tempAsstId = 'pending_asst_' + Date.now();
    await mutateMessages(
      (cur) => ({
        items: [
          ...(cur?.items ?? messages.items),
          {
            id: tempUserId,
            role: 'user',
            content,
            sources: null,
            tokensUsed: null,
            createdAt: new Date().toISOString(),
          },
          {
            id: tempAsstId,
            role: 'assistant',
            content: '',
            sources: null,
            tokensUsed: null,
            createdAt: new Date().toISOString(),
          },
        ],
      }),
      { revalidate: false },
    );

    try {
      const { messageId } = await apiSendMessage(conversationId, { content });
      await mutateMessages(
        (cur) => {
          const items = (cur?.items ?? []).map((m) =>
            m.id === tempAsstId ? { ...m, id: messageId } : m,
          );
          return { items };
        },
        { revalidate: false },
      );
      setActive(messageId);
    } catch {
      await mutateMessages(
        (cur) => ({
          items: (cur?.items ?? []).filter((m) => m.id !== tempUserId && m.id !== tempAsstId),
        }),
        { revalidate: false },
      );
      setErrorReason(t('composer.error.reason'));
      setActive(null);
    }
  }

  function send() {
    const content = value.trim();
    if (!content) return;
    if (content.length > MAX_LEN) return;
    void sendContent(content);
  }

  function resend() {
    void sendContent(lastUserContent.current);
  }

  const streaming = state === 'streaming';

  return (
    <div className={styles.wrap}>
      {errorReason && (
        <div className={styles.errorRow}>
          <span>{t('composer.error.reason')}: {errorReason}</span>
          <Button size="sm" variant="outline" onClick={() => void resend()}>
            {t('composer.error.retry')}
          </Button>
        </div>
      )}
      <div className={styles.card}>
        <Textarea
          value={value}
          onChange={(e) => setValue(e.target.value)}
          placeholder={t('composer.placeholder')}
          autoSize={{ minRows: 1, maxRows: 6 }}
          disabled={streaming}
          onKeyDown={(e) => {
            if (e.key === 'Enter' && !e.shiftKey) {
              e.preventDefault();
              send();
            }
          }}
        />
        <div className={styles.actions}>
          {streaming ? (
            <Button danger variant="ghost" icon={<StopIcon size={14} />} onClick={stop}>
              {t('composer.stop')}
            </Button>
          ) : (
            <Button
              variant="primary"
              icon={<SendIcon size={14} />}
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
}
```

创建 `frontend/src/components/conversation/MessageComposer.module.css`：
```css
.wrap {
  flex: none;
  padding: 8px 24px 16px;
}

.errorRow {
  display: flex;
  align-items: center;
  gap: 10px;
  max-width: var(--dsh-chat-content-width);
  margin: 0 auto 8px;
  padding: 8px 12px;
  border-radius: 10px;
  background: var(--dsw-alias-interactive-bg-hover-danger);
  color: var(--dsw-alias-state-error-primary);
  font-size: 13px;
}

.errorRow span {
  flex: 1;
  min-width: 0;
}

.card {
  max-width: calc(var(--dsh-chat-content-width) + 16px);
  margin: 0 auto;
  border: 1px solid var(--dsw-alias-border-l2);
  border-radius: 16px;
  background: var(--dsw-specific-input-major);
  padding: 10px 12px;
  box-sizing: border-box;
  box-shadow: var(--dsw-shadow-lv1);
  transition:
    border-color var(--ds-transition-duration) var(--ds-ease-in-out),
    box-shadow var(--ds-transition-duration) var(--ds-ease-in-out);
}

.card:focus-within {
  border-color: var(--dsw-alias-brand-primary-new-colorprimary-new-color);
  box-shadow: 0 0 0 3px var(--dsw-alias-interactive-bg-hover-accent);
}

.actions {
  display: flex;
  justify-content: flex-end;
  padding-top: 6px;
}
```

注意：`Button` 需支持 `danger` prop（当前未定义）。在 Task 6 的 `Button.tsx` 中补充 `danger`：

在 `ButtonProps` 增加 `danger?: boolean;`，在 `classes` 数组加入 `danger ? styles.danger : ''`，并在 `Button.module.css` 追加：
```css
.danger {
  color: var(--dsw-alias-state-error-primary);
}
```

- [ ] **Step 2: 更新 MessageComposer 测试**

`frontend/src/components/conversation/MessageComposer.test.tsx` 无需修改（查询仍用 `textbox` 与「发送」按钮）。若「发送」文本被 `composer.send` 键提供，保持 i18n 不变即可。

- [ ] **Step 3: 运行测试确认通过**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/components/conversation/MessageComposer.test.tsx src/pages/ConversationPage.test.tsx`
Expected: PASS（若 ConversationPage 仍引 antd，会失败——需先完成 Task 20）。

- [ ] **Step 4: 提交**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git add frontend/src/components/conversation/MessageComposer.tsx frontend/src/components/conversation/MessageComposer.module.css frontend/src/primitives/Button.tsx frontend/src/primitives/Button.module.css && git commit -m "refactor(frontend): 重绘 MessageComposer，Button 支持 danger"
```

---

### Task 20: ConversationPage 重绘

**Files:**
- Create: `frontend/src/pages/ConversationPage.tsx`（覆盖）
- Create: `frontend/src/pages/ConversationPage.module.css`
- Modify: `frontend/src/pages/ConversationPage.test.tsx`

- [ ] **Step 1: 重写 ConversationPage**

创建 `frontend/src/pages/ConversationPage.tsx`（覆盖，逻辑不变、去掉 antd）：
```tsx
import { useEffect, useState } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button } from '@/primitives/Button';
import { PlusIcon } from '@/primitives/icons';
import { useConversation } from '@/hooks/useConversation';
import { useConversationStore } from '@/stores/conversationStore';
import MessageList from '@/components/conversation/MessageList';
import { MessageComposer } from '@/components/conversation/MessageComposer';
import { TurnStatus } from '@/features/conversation/TurnStatus';
import type { Message } from '@/api/conversation/types';
import type { Paginated } from '@/api/types';
import styles from './ConversationPage.module.css';

type MessagesLike = { items: Message[] };
type ComposerMutate = (
  updater?: ((cur?: MessagesLike) => MessagesLike) | MessagesLike,
  opts?: { revalidate?: boolean },
) => Promise<unknown>;

export default function ConversationPage() {
  const { id } = useParams();
  const { t } = useTranslation();
  const navigate = useNavigate();
  const location = useLocation();
  const { conv, messages, mutateMessages } = useConversation(id ?? null);
  const setActive = useConversationStore((s) => s.setActive);
  const clearActive = useConversationStore((s) => s.clearActive);
  const [streamingId, setStreamingId] = useState<string | null>(null);

  useEffect(() => {
    if (location.state?.pendingMessageId) {
      window.history.replaceState({}, '', location.pathname);
    }
  }, [location]);

  useEffect(() => {
    if (id) setActive(id);
  }, [id, setActive]);

  if (conv.error) return <div className={styles.error}>{t('error.generic')}</div>;
  if (!conv.data) return <div className={styles.loading}>{t('common.loading')}</div>;

  const composerMutate: ComposerMutate = (updater, opts) => {
    if (typeof updater === 'function') {
      return mutateMessages((cur?: Paginated<Message>) => {
        const next = updater(cur);
        return {
          items: next.items,
          total: cur?.total ?? next.items.length,
          hasMore: cur?.hasMore ?? false,
        };
      }, opts);
    }
    if (updater) {
      return mutateMessages(
        { items: updater.items, total: updater.items.length, hasMore: false },
        opts,
      );
    }
    return mutateMessages();
  };

  return (
    <div className={styles.root}>
      <header className={styles.header}>
        <span className={styles.title}>{conv.data.title}</span>
        <Button
          variant="ghost"
          size="sm"
          icon={<PlusIcon size={14} />}
          onClick={() => {
            clearActive();
            navigate('/');
          }}
        >
          {t('home.newButton')}
        </Button>
      </header>
      <MessageList messages={messages.data?.items ?? []} streamingId={streamingId ?? undefined} />
      {streamingId && <TurnStatus />}
      <MessageComposer
        key={id}
        conversationId={id!}
        messages={{ items: messages.data?.items ?? [] }}
        mutateMessages={composerMutate}
        onStreamingChange={setStreamingId}
        initialStreamMessageId={
          (location.state as { pendingMessageId?: string } | null)?.pendingMessageId ?? null
        }
      />
    </div>
  );
}
```

创建 `frontend/src/pages/ConversationPage.module.css`：
```css
.root {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.header {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 52px;
  padding: 0 20px;
  border-bottom: 1px solid var(--dsw-alias-border-l1);
  box-sizing: border-box;
}

.title {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 15px;
  font-weight: 600;
  color: var(--dsw-alias-label-primary);
}

.error {
  padding: 24px;
  color: var(--dsw-alias-state-error-primary);
}

.loading {
  padding: 24px;
  color: var(--dsw-alias-label-tertiary);
}
```

- [ ] **Step 2: 确认 ConversationPage 测试可运行**

`frontend/src/pages/ConversationPage.test.tsx` 不依赖 antd 渲染细节（用 `textbox`、`发\s*送`、`/针对「越南税收」/`、来源标题文本），无需修改。

- [ ] **Step 3: 运行测试确认通过**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/pages/ConversationPage.test.tsx src/components/conversation/MessageComposer.test.tsx`
Expected: PASS。

- [ ] **Step 4: 提交**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git add frontend/src/pages/ConversationPage.tsx frontend/src/pages/ConversationPage.module.css && git commit -m "refactor(frontend): 重绘 ConversationPage"
```

---

### Task 21: 登录/注册页重绘

**Files:**
- Create: `frontend/src/pages/LoginPage.tsx`（覆盖）
- Create: `frontend/src/pages/LoginPage.module.css`
- Create: `frontend/src/pages/RegisterPage.tsx`（覆盖）
- Create: `frontend/src/pages/RegisterPage.module.css`
- Create: `frontend/src/features/auth/fields.tsx`（表单字段抽取）

- [ ] **Step 1: 实现登录/注册公共字段**

创建 `frontend/src/features/auth/fields.tsx`：
```tsx
import { Input } from '@/primitives/Input';
import { MailIcon } from '@/primitives/icons';
import styles from './fields.module.css';

export function Field({ label, error, children }: { label: string; error?: string; children: React.ReactNode }) {
  return (
    <label className={styles.field}>
      <span className={styles.label}>{label}</span>
      {children}
      {error && <span className={styles.error}>{error}</span>}
    </label>
  );
}
```

创建 `frontend/src/features/auth/fields.module.css`：
```css
.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 14px;
}

.label {
  font-size: 13px;
  color: var(--dsw-alias-label-secondary);
}

.error {
  font-size: 12px;
  color: var(--dsw-alias-state-error-primary);
}
```

（如不需要 `MailIcon` 则删除 import。）

- [ ] **Step 2: 重写 LoginPage**

创建 `frontend/src/pages/LoginPage.tsx`（覆盖，去掉 antd Form）：
```tsx
import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button } from '@/primitives/Button';
import { Input } from '@/primitives/Input';
import { useToast } from '@/primitives/ToastProvider';
import { CompassLogo } from '@/theme/logo';
import { login as apiLogin } from '@/api/auth/auth';
import { useAuthStore } from '@/stores/authStore';
import type { LoginRequest } from '@/api/auth/types';
import { Field } from '@/features/auth/fields';
import styles from './LoginPage.module.css';

type LoginFormValues = LoginRequest & { remember?: boolean };

export default function LoginPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const toast = useToast();
  const token = useAuthStore((s) => s.token);
  const loginStore = useAuthStore((s) => s.login);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [remember, setRemember] = useState(true);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (token) navigate('/', { replace: true });
  }, [token, navigate]);

  async function onLogin(e: React.FormEvent) {
    e.preventDefault();
    if (loading) return;
    setLoading(true);
    try {
      const res = await apiLogin({ email, password });
      loginStore({ token: res.token, user: res.user, remember: remember !== false });
      navigate('/', { replace: true });
    } catch (err: unknown) {
      const code = (err as { code?: string })?.code;
      toast.error(code === 'UNAUTHORIZED' ? t('auth.error.invalid') : t('error.generic'));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className={styles.root}>
      <div className={styles.card}>
        <div className={styles.head}>
          <CompassLogo size={44} />
          <h1 className={styles.title}>{t('auth.login.title')}</h1>
          <p className={styles.subtitle}>{t('auth.login.subtitle')}</p>
        </div>
        <form onSubmit={(e) => void onLogin(e)}>
          <Field label={t('auth.field.email')}>
            <Input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder={t('auth.field.email')}
              required
            />
          </Field>
          <Field label={t('auth.field.password')}>
            <Input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder={t('auth.field.password')}
              required
              minLength={8}
              maxLength={72}
            />
          </Field>
          <label className={styles.remember}>
            <input
              type="checkbox"
              checked={remember}
              onChange={(e) => setRemember(e.target.checked)}
            />
            <span>{t('auth.login.remember')}</span>
          </label>
          <Button type="submit" variant="primary" block size="md" loading={loading}>
            {t('auth.login.submit')}
          </Button>
        </form>
        <div className={styles.switch}>
          <Button variant="ghost" size="sm" onClick={() => navigate('/register')}>
            {t('auth.login.toRegister')}
          </Button>
        </div>
      </div>
    </div>
  );
}
```

创建 `frontend/src/pages/LoginPage.module.css`：
```css
.root {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 24px;
  box-sizing: border-box;
  background: var(--dsw-alias-bg-base);
}

.card {
  width: 100%;
  max-width: 400px;
  padding: 28px;
  border: 1px solid var(--dsw-alias-border-l1);
  border-radius: 20px;
  background: var(--dsw-alias-bg-layer-1);
  box-shadow: var(--dsw-shadow-lv2);
  box-sizing: border-box;
}

.head {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 24px;
  text-align: center;
}

.title {
  margin: 12px 0 4px;
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.2px;
  color: var(--dsw-alias-label-primary);
}

.subtitle {
  margin: 0;
  font-size: 13px;
  color: var(--dsw-alias-label-secondary);
}

.remember {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 4px 0 16px;
  font-size: 13px;
  color: var(--dsw-alias-label-secondary);
  cursor: pointer;
}

.switch {
  margin-top: 16px;
  text-align: center;
}
```

- [ ] **Step 3: 重写 RegisterPage**

创建 `frontend/src/pages/RegisterPage.tsx`（覆盖）：
```tsx
import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Button } from '@/primitives/Button';
import { Input } from '@/primitives/Input';
import { useToast } from '@/primitives/ToastProvider';
import { CompassLogo } from '@/theme/logo';
import { register as apiRegister } from '@/api/auth/auth';
import { useAuthStore } from '@/stores/authStore';
import { Field } from '@/features/auth/fields';
import styles from './RegisterPage.module.css';

type FormValues = {
  displayName: string;
  email: string;
  password: string;
  confirmPassword: string;
};

export default function RegisterPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const toast = useToast();
  const token = useAuthStore((s) => s.token);
  const loginStore = useAuthStore((s) => s.login);
  const [form, setForm] = useState<FormValues>({
    displayName: '',
    email: '',
    password: '',
    confirmPassword: '',
  });
  const [loading, setLoading] = useState(false);
  const [mismatch, setMismatch] = useState(false);

  useEffect(() => {
    if (token) navigate('/', { replace: true });
  }, [token, navigate]);

  function set<K extends keyof FormValues>(key: K, value: string) {
    setForm((f) => ({ ...f, [key]: value }));
    if (key === 'confirmPassword' || key === 'password') {
      setMismatch(false);
    }
  }

  async function onRegister(e: React.FormEvent) {
    e.preventDefault();
    if (loading) return;
    if (form.password !== form.confirmPassword) {
      setMismatch(true);
      return;
    }
    setLoading(true);
    try {
      const req = {
        email: form.email,
        password: form.password,
        displayName: form.displayName,
      };
      const res = await apiRegister(req);
      loginStore({ token: res.token, user: res.user, remember: true });
      navigate('/', { replace: true });
    } catch (err: unknown) {
      const code = (err as { code?: string })?.code;
      toast.error(code === 'CONFLICT' ? t('auth.error.conflict') : t('error.generic'));
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className={styles.root}>
      <div className={styles.card}>
        <div className={styles.head}>
          <CompassLogo size={44} />
          <h1 className={styles.title}>{t('auth.register.title')}</h1>
          <p className={styles.subtitle}>{t('auth.register.subtitle')}</p>
        </div>
        <form onSubmit={(e) => void onRegister(e)}>
          <Field label={t('auth.field.displayName')}>
            <Input
              value={form.displayName}
              onChange={(e) => set('displayName', e.target.value)}
              placeholder={t('auth.field.displayName')}
              required
            />
          </Field>
          <Field label={t('auth.field.email')}>
            <Input
              type="email"
              value={form.email}
              onChange={(e) => set('email', e.target.value)}
              placeholder={t('auth.field.email')}
              required
            />
          </Field>
          <Field label={t('auth.field.password')}>
            <Input
              type="password"
              value={form.password}
              onChange={(e) => set('password', e.target.value)}
              placeholder={t('auth.field.password')}
              required
              minLength={8}
              maxLength={72}
            />
          </Field>
          <Field label={t('auth.field.confirmPassword')} error={mismatch ? t('auth.error.passwordMismatch') : undefined}>
            <Input
              type="password"
              value={form.confirmPassword}
              onChange={(e) => set('confirmPassword', e.target.value)}
              placeholder={t('auth.field.confirmPassword')}
              required
            />
          </Field>
          <Button type="submit" variant="primary" block size="md" loading={loading}>
            {t('auth.register.submit')}
          </Button>
        </form>
        <div className={styles.switch}>
          <Button variant="ghost" size="sm" onClick={() => navigate('/login')}>
            {t('auth.register.toLogin')}
          </Button>
        </div>
      </div>
    </div>
  );
}
```

创建 `frontend/src/pages/RegisterPage.module.css`（与 LoginPage 相同结构，直接复制）：
```css
.root {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 24px;
  box-sizing: border-box;
  background: var(--dsw-alias-bg-base);
}

.card {
  width: 100%;
  max-width: 400px;
  padding: 28px;
  border: 1px solid var(--dsw-alias-border-l1);
  border-radius: 20px;
  background: var(--dsw-alias-bg-layer-1);
  box-shadow: var(--dsw-shadow-lv2);
  box-sizing: border-box;
}

.head {
  display: flex;
  flex-direction: column;
  align-items: center;
  margin-bottom: 24px;
  text-align: center;
}

.title {
  margin: 12px 0 4px;
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.2px;
  color: var(--dsw-alias-label-primary);
}

.subtitle {
  margin: 0;
  font-size: 13px;
  color: var(--dsw-alias-label-secondary);
}

.switch {
  margin-top: 16px;
  text-align: center;
}
```

- [ ] **Step 4: 更新登录/注册测试**

`frontend/src/pages/LoginPage.test.tsx` 与 `RegisterPage.test.tsx`：
- 测试用 `render(...)` 直接渲染页面，但页面现在调用 `useToast()`，必须包 `ToastProvider`。
- 修改 `renderAt` 包裹：

```tsx
import { ToastProvider } from '@/primitives/ToastProvider';

function renderAt(path: string) {
  return render(
    <ToastProvider>
      <MemoryRouter initialEntries={[path]}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/register" element={<div>Register page</div>} />
          <Route path="/" element={<div>Home</div>} />
        </Routes>
      </MemoryRouter>
    </ToastProvider>,
  );
}
```

RegisterPage 同理。其余断言（placeholder、`登\s*录`、`注\s*册`、密码不一致文本、checkbox「记住我」）在新 markup 中仍成立（`<input type="checkbox">` + 相邻文本、label 内文本）。

- [ ] **Step 5: 运行测试确认通过**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run test src/pages/LoginPage.test.tsx src/pages/RegisterPage.test.tsx src/pages/HomePage.test.tsx`
Expected: PASS。

- [ ] **Step 6: 提交**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git add frontend/src/pages/LoginPage.tsx frontend/src/pages/LoginPage.module.css frontend/src/pages/RegisterPage.tsx frontend/src/pages/RegisterPage.module.css frontend/src/features/auth && git commit -m "refactor(frontend): 重绘登录/注册页"
```

---

### Task 22: App 根组件与入口接线

**Files:**
- Create: `frontend/src/App.tsx`（覆盖）
- Modify: `frontend/src/main.tsx`
- Modify: `frontend/src/components/ErrorBoundary.tsx`
- Create: `frontend/src/theme/ThemeProvider.test.tsx`

- [ ] **Step 1: 重写 App.tsx**

创建 `frontend/src/App.tsx`（覆盖，移除 antd ConfigProvider）：
```tsx
import { RouterProvider } from 'react-router-dom';
import { SWRConfig } from 'swr';
import './i18n/config';
import { router } from './router';
import { ErrorBoundary } from './components/ErrorBoundary';
import { ThemeProvider } from './theme/ThemeProvider';
import { ToastProvider } from './primitives/ToastProvider';

export default function App() {
  return (
    <ThemeProvider>
      <ToastProvider>
        <ErrorBoundary>
          <SWRConfig
            value={{
              onError: (err) => {
                if (err?.status !== 401 && import.meta.env.DEV) {
                  console.error('SWR error', err);
                }
              },
            }}
          >
            <RouterProvider router={router} />
          </SWRConfig>
        </ErrorBoundary>
      </ToastProvider>
    </ThemeProvider>
  );
}
```

- [ ] **Step 2: 更新 main.tsx**

创建 `frontend/src/main.tsx`（覆盖，去掉 antd-override.css）：
```tsx
import './api';
import { createRoot } from 'react-dom/client';
import './styles/main.css';
import App from './App';

const root = document.getElementById('root');
if (root) {
  createRoot(root).render(<App />);
}
```

- [ ] **Step 3: 更新 ErrorBoundary**

编辑 `frontend/src/components/ErrorBoundary.tsx`，去掉 antd `Result`：
```tsx
import { Component, type ErrorInfo, type ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '@/primitives/Button';
import styles from './ErrorBoundary.module.css';

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
    <div className={styles.root}>
      <div className={styles.card}>
        <p>{t('error.generic')}</p>
        <Button variant="primary" onClick={onReload}>
          {t('common.retry')}
        </Button>
      </div>
    </div>
  );
}
```

创建 `frontend/src/components/ErrorBoundary.module.css`：
```css
.root {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  padding: 24px;
  box-sizing: border-box;
  background: var(--dsw-alias-bg-base);
}

.card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 16px;
  padding: 32px;
  border: 1px solid var(--dsw-alias-border-l1);
  border-radius: 20px;
  background: var(--dsw-alias-bg-layer-1);
  color: var(--dsw-alias-label-primary);
}
```

- [ ] **Step 4: 写 ThemeProvider 测试**

创建 `frontend/src/theme/ThemeProvider.test.tsx`：
```tsx
import { describe, it, expect, beforeEach } from 'vitest';
import { render } from '@testing-library/react';
import { act } from 'react';
import { ThemeProvider } from './ThemeProvider';
import { useThemeStore } from './themeStore';

describe('ThemeProvider', () => {
  beforeEach(() => {
    localStorage.clear();
    document.body.removeAttribute('data-ds-dark-theme');
  });

  it('挂载后按 store 模式写入 body 属性', () => {
    act(() => {
      useThemeStore.getState().setMode('dark');
    });
    render(
      <ThemeProvider>
        <div>child</div>
      </ThemeProvider>,
    );
    expect(document.body.hasAttribute('data-ds-dark-theme')).toBe(true);
  });
});
```

- [ ] **Step 5: 运行测试确认通过**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run lint && bunx tsc --noEmit && bun run test`
Expected: 全部通过（antd 引用已全部移除）。若仍有 antd 引用，用 `rg "antd" frontend/src` 定位并清理（重点：`styles/antd-override.css` 待 Task 23 删除，此时可先删除其 import）。

- [ ] **Step 6: 提交**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git add frontend/src/App.tsx frontend/src/main.tsx frontend/src/components/ErrorBoundary.tsx frontend/src/components/ErrorBoundary.module.css frontend/src/theme/ThemeProvider.test.tsx && git commit -m "refactor(frontend): 根组件切换为 ThemeProvider+ToastProvider，入口清理"
```

---

### Task 23: 清理旧样式与废弃文件

**Files:**
- Delete: `frontend/src/styles/antd-override.css`
- Delete: `frontend/src/components/layout/`（旧 AppLayout/Sidebar/UserMenu 已被 layout/ 替代）
- Modify: `frontend/src/router.tsx`（更新 layout 导入路径）
- Modify: `frontend/src/i18n/zh-CN.json`、`frontend/src/i18n/en-US.json`（新增 i18n key）

- [ ] **Step 1: 删除废弃文件**

Run:
```bash
rm /home/hunter/code/invest-guide-workspace/invest-guide/frontend/src/styles/antd-override.css
rm -rf /home/hunter/code/invest-guide-workspace/invest-guide/frontend/src/components/layout
```
Expected: 文件删除成功。

- [ ] **Step 2: 更新 router 导入**

编辑 `frontend/src/router.tsx`，把 `import AppLayout from './components/layout/AppLayout';` 改为 `import AppLayout from './layout/AppLayout';`。其余逻辑不变。

- [ ] **Step 3: 补充 i18n key**

在 `frontend/src/i18n/zh-CN.json` 与 `en-US.json` 中新增：

zh-CN.json:
```json
"sidebar": {
  "title": "Invest Guide",
  "newConversation": "新建对话",
  "userMenu": { "logout": "登出", "theme": "切换主题" },
  "toggle": "折叠侧边栏",
  "group": { "today": "今天", "yesterday": "昨天", "earlier": "更早" }
},
"details": {
  "title": "引用详情",
  "empty": "选择一条回复查看引用详情",
  "tokens": "Tokens"
}
```

en-US.json:
```json
"sidebar": {
  "title": "Invest Guide",
  "newConversation": "New chat",
  "userMenu": { "logout": "Sign out", "theme": "Toggle theme" },
  "toggle": "Toggle sidebar",
  "group": { "today": "Today", "yesterday": "Yesterday", "earlier": "Earlier" }
},
"details": {
  "title": "Citation details",
  "empty": "Select a reply to view citation details",
  "tokens": "Tokens"
}
```

- [ ] **Step 4: 全量验证**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run lint && bunx tsc --noEmit && bun run test`
Expected: 全绿。若 `router.test.tsx` 失败（首页文案由 `home.welcome` 提供，未变），检查具体断言。

- [ ] **Step 5: 提交**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git add frontend/src/router.tsx frontend/src/i18n && git commit -m "chore(frontend): 清理废弃样式/组件，补充 i18n key"
```

---

### Task 24: 文档同步（AGENT.md / ARCHITECTURE.md）

**Files:**
- Modify: `AGENT.md`
- Modify: `ARCHITECTURE.md`

- [ ] **Step 1: 更新 AGENT.md 前端规范**

编辑 `AGENT.md`：
- 「UI 库与图标（仅前端）」一节改为：不再使用 antd，改为自研基础组件 `frontend/src/primitives/`（Button/Input/Modal/Menu/Tooltip/Toast/DisclosureRow/Pill/Icon），图标用 `primitives/icons.tsx` 内联 SVG；不在业务组件中直接使用原生交互 HTML（统一封装在 primitives）。
- 「CSS（前端）」一节改为：使用 CSS Modules（`ComponentName.module.css`）；所有颜色/间距用 `frontend/src/styles/` 定义的 `--dsw-*` 设计 token（浅色 `body` / 暗色 `body[data-ds-dark-theme]`）；禁止硬编码色值；全局样式只在 `styles/` 下。
- 移除 Tailwind 相关规则（`@theme`、`dark:` 变体、Preflight 禁用说明），替换为 CSS Modules 说明与 `--dsw-*` token 使用约定。
- 暗黑模式：`body[data-ds-dark-theme]` 为唯一切换点，由 `theme/ThemeProvider` 维护。

- [ ] **Step 2: 更新 ARCHITECTURE.md**

编辑 `ARCHITECTURE.md` 前端部分：
- 更新前端分层描述（`primitives/layout/pages/features/stores/hooks/api/i18n/theme/styles`）。
- 更新依赖说明（移除 antd/tailwind，保留 CSS Modules + zustand + swr + react-router + i18next + react-markdown）。
- 更新设计 token 说明（`--dsw-*`，浅/暗双主题）。

- [ ] **Step 3: 验证无遗漏**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide && rg -n "antd|tailwind|@theme|Tailwind" AGENT.md ARCHITECTURE.md`
Expected: 前端相关描述不再引用 antd/tailwind（后端/历史记录除外）。

- [ ] **Step 4: 提交**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git add AGENT.md ARCHITECTURE.md && git commit -m "docs(frontend): 同步 AGENT.md/ARCHITECTURE.md 前端设计规范"
```

---

### Task 25: 端到端验证

**Files:**
- 无（仅验证）

- [ ] **Step 1: 全量质量门禁**

Run:
```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run lint && bun run format && bunx tsc --noEmit && bun run test
```
Expected: 全部通过，无 TypeScript 错误，无 lint 告警。

- [ ] **Step 2: 构建验证**

Run: `cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && bun run build`
Expected: 构建成功，无 CSS/模块解析错误。

- [ ] **Step 3: 启动 dev server 目检**

Run:
```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide/frontend && VITE_USE_MOCK=true bun run dev
```
手动验证（浏览器）：
- 登录/注册页渲染正常，明暗切换（用户菜单）生效
- 首页 logo + 输入卡
- 新建会话 → 发送问题 → 流式回答（shimmer 状态行）→ 来源卡可展开 → 点击 [N] 芯片联动右栏高亮
- 会话列表分组、删除确认弹窗
- 右栏拖拽、侧栏折叠动画
Expected: 全部符合设计。

- [ ] **Step 4: 提交最终状态**

```bash
cd /home/hunter/code/invest-guide-workspace/invest-guide && git status --short
```
如有未提交改动，按前面任务的分组规则提交；无改动则跳过。

---

## Self-Review

**1. Spec coverage（对照 `docs/frontend/design-dsh-rewrite.md`）：**
- 技术栈照搬（CSS Modules + primitives + token）→ Task 1,2,5-10 ✓
- 保留全部功能（登录/注册/首页/会话/来源/流式/删除）→ Task 12,15,16,17,18,19,20,21 ✓
- 来源卡+内联芯片 → Task 16,17 ✓
- 双主题+切换 → Task 3,14,22 ✓
- 品牌蓝+指南针 logo → Task 3,15,21 ✓
- react-markdown 保留 → Task 16 ✓
- 三栏可拖拽 → Task 11,13,14 ✓
- 右侧详情栏（选中消息来源+元信息）→ Task 13,14,17 ✓
- 移除 antd/tailwind → Task 1,23 ✓
- 文档同步 → Task 24 ✓

**2. Placeholder scan:** 无 TBD/TODO；所有代码步骤含完整实现。

**3. Type consistency:**
- `conversationStore` 在 Task 13 定义为 `selectedMessageId`，Task 14 改为 `selectedMessage`——Task 14 的 Step 3 明确标注「覆盖 Step 1 定义」，需执行者按推荐方案选用其一并同步 DetailsPanel 测试。
- `Button` 的 `danger` prop 在 Task 19 补充，Task 6 已预留扩展点 ✓
- `DetailsPanel` 的 `message` prop 与 store 读取在 Task 14 二选一，已在步骤内说明。

**风险提示：** Task 14 存在两套实现（布局层 useConversation vs store 存整条消息）。推荐选「store 存整条消息」，避免布局层重复请求。执行者必须同步 Task 13 的 DetailsPanel 定义，避免类型不一致。
