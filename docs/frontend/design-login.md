# InvestGuide 登录页改版设计（antd Pro 标准登录模板）

> 落点：`docs/frontend/design-login.md`
> 日期：2026-08-04
> 状态：已批准（终端确认）
> 范围：登录页 + 新增注册页，参照 [antd 表单页设计规范](https://ant.design/docs/spec/research-form-cn) 的「登录」模板（Ant Design Pro 标准登录）。

## 1. 背景与目标

当前登录页为 antd 默认蓝色风格，登录/注册共用 `Tabs` 双 Tab。参照 antd 表单页规范的
「登录」模板，将其改造为 Pro 标准登录结构：

- 居中单列表单卡（研究显示单列自上而下录入效率最高）
- 输入框 prefix 图标、记住我 + 全宽主按钮、注册链接
- 登录与注册拆分为独立页面，各自一个表单任务

**已确认决策**：
1. 采用 antd Pro 标准登录模板结构（居中卡片、无 Tabs）
2. 注册入口为独立路由 `/register`
3. 「记住我」实现真实语义（localStorage / sessionStorage 分流）
4. 视觉沿用项目当前 token（不引入 DeepSeek 配色，属于后续全局改版范围）
5. 不加「忘记密码」链接（后端无该端点，避免死链接）

## 2. 页面结构

### LoginPage（`/login`）

```
居中卡片（max-w-[360px]）
├── 品牌区：Compass logo + 标题 + 副标题（沿用现有文案 key）
├── Form（layout="vertical"）
│   ├── 邮箱    — Input prefix=MailOutlined
│   ├── 密码    — Input.Password prefix=LockOutlined
│   ├── 记住我  — Checkbox（默认勾选），与提交按钮同一区域
│   └── 提交    — Button block type="primary" loading
└── 「还没有账号？注册」— Button type="link" → /register
```

- 校验规则沿用现有：邮箱 `required + type:email`；密码 `required + min:8 + max:72`
- 提交逻辑不变：`apiLogin` → `loginStore({ token, user, remember })` → `navigate('/', replace)`
- 错误反馈不变：`UNAUTHORIZED` → `auth.error.invalid`，其余 → `error.generic`
- 已登录访问 `/login` → 重定向 `/`

### RegisterPage（`/register`，新页面）

```
居中卡片（同 Login 风格）
├── 品牌区：标题 + 副标题
├── Form（layout="vertical"）
│   ├── 昵称       — Input
│   ├── 邮箱       — Input prefix=MailOutlined
│   ├── 密码       — Input.Password prefix=LockOutlined
│   ├── 确认密码   — Input.Password，validator 比对两次输入
│   └── 提交       — Button block type="primary" loading
└── 「已有账号？去登录」— Button type="link" → /login
```

- 提交逻辑不变：`apiRegister` → `loginStore({ token, user, remember: true })` → `navigate('/', replace)`
- 错误反馈不变：`CONFLICT` → `auth.error.conflict`，其余 → `error.generic`
- 确认密码仅前端校验，不进入 `RegisterRequest`（API 契约不变）

## 3. 记住我语义（authStore）

改造 `login` / `hydrate` / `logout` 的存储行为：

| 场景 | 存储位置 | 效果 |
|------|----------|------|
| 勾选「记住我」 | `localStorage` | 浏览器重启仍保持登录 |
| 未勾选 | `sessionStorage` | 关闭浏览器会话即失效 |

- `login(p: { token, user, remember })`：`remember === false` 写 sessionStorage，否则 localStorage（向后兼容：默认 true）
- `hydrate()`：先读 localStorage，再读 sessionStorage
- `logout()`：同时清除两个存储中的 token
- 新增常量 `SESSION_TOKEN_KEY`（如 `investguide.token.session`）；`TOKEN_KEY` 语义不变
- 认证成功响应后（SSE/API 401 处理等）不受影响——它们调用 `logout()` 无需感知存储位置

## 4. 路由与守卫

`router.tsx`：

- 新增 `{ path: '/register', element: <RegisterPage /> }`（与 `/login` 同级，不要求登录）
- 登录态访问 `/login`、`/register` 均重定向 `/`（LoginPage 已有 useEffect 逻辑，RegisterPage 同样实现）

## 5. i18n 变更

新增 key：

| key | zh-CN | en-US |
|-----|-------|-------|
| `auth.login.remember` | 记住我 | Remember me |
| `auth.login.toRegister` | 还没有账号？注册 | New here? Sign up |
| `auth.register.toLogin` | 已有账号？去登录 | Already have an account? Sign in |
| `auth.register.subtitle` | 创建账号，开始你的投资研究 | Create an account to start |
| `auth.field.confirmPassword` | 确认密码 | Confirm password |
| `auth.error.passwordMismatch` | 两次输入的密码不一致 | The two passwords do not match |

删除不再使用的 key：`auth.login.tab`、`auth.register.tab`（Tabs 移除）。

## 6. 明确不动的内容

- 后端 API / 契约 / mock：`LoginRequest`、`RegisterRequest`、`AuthResponse` 均不变
- 其余页面（Home / Conversation / Sidebar）与全局样式
- antd 主题 token（本轮不引入 DeepSeek 配色）
- `body[data-theme=dark]` 钩子

## 7. 测试

| 文件 | 变更 |
|------|------|
| `LoginPage.test.tsx` | 移除 Tabs 断言；改为断言登录表单（邮箱/密码/记住我/提交）与注册链接；保留登录成功写 token 用例（改 `loginStore` 带 remember 参数） |
| `RegisterPage.test.tsx` | 新增：渲染各字段、密码不一致校验提示、成功注册写 token、错误反馈 |
| `authStore.test.ts` | 更新 login 持久化断言（默认 localStorage）；新增 remember=false 走 sessionStorage、hydrate 读 sessionStorage、logout 清两处 |
| `router.test.tsx` | 断言从「tab」改为登录表单（如 `getByLabelText(/邮箱/)`） |

## 8. 验证

```bash
cd frontend && bun run lint && bunx tsc --noEmit && bun run test
```

门禁：lint 0 errors、tsc clean、全部测试通过。

## 9. 不在本轮范围

- 深色模式、响应式、全局配色改版（DeepSeek 风格）
- 忘记密码流程
- 第三方登录 / 短信验证码
