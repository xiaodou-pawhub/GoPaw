# GoPaw Unified v2 — 项目合并说明文档

> 分支：`feature/unified-v2`
> 文档日期：2026-03-21
> 状态：**Step 1~3 已完成，Step 4 待实现**

---

## 背景与目标

原本存在三个相互独立的项目：

| 项目 | 定位 | 问题 |
|------|------|------|
| `GoPaw` | Server 版，Web UI，主功能库 | 功能最完整，但只支持单用户 token 认证 |
| `gopaw-desk` | Wails 桌面应用 | 与 GoPaw 功能高度重叠，独立维护成本高 |
| `gopaw-cloud` | SaaS 多用户版 | 代码与 GoPaw 分叉，难以同步 |

**合并目标**：以 GoPaw 为主干，在 `feature/unified-v2` 分支上叠加能力，形成一个支持「个人本地 → 小团队 → 未来云服务」的统一项目，**不删除任何现有功能代码**。

---

## 三种运行模式

```
gopaw start --mode solo    # 个人本地（默认）
gopaw start --mode team    # 小团队
gopaw start --mode cloud   # SaaS（Phase 4 实现）
```

| 行为 | solo（默认） | team | cloud |
|------|------------|------|-------|
| 认证 | **无需登录**，直接访问 | JWT（用户名+密码） | JWT + 邀请码 |
| 用户 | 单用户概念 | Admin 创建账号 | 开放/受邀注册 |
| 用户上限 | 1 | ≤ 50（可配） | 无限 |
| Admin 面板 | 隐藏 | 显示用户管理 | 完整 SaaS 面板 |
| 数据隔离 | 无 user_id 区分 | user_id 过滤 | user_id 过滤 |
| 桌面托盘 | `--tray` 可选 | — | — |

---

## 已完成工作

### Step 1：数据库驱动统一 + Mode 配置

#### 数据库驱动替换

原项目混用两个 SQLite 驱动：

- `modernc.org/sqlite`（纯 Go）：用于 memory/store、ltm_store
- `mattn/go-sqlite3`（**CGo**）：用于 trace、skill、agent、migrate 工具

**改动**：将所有 `mattn/go-sqlite3` 替换为 `modernc.org/sqlite`，统一驱动名 `"sqlite3"` → `"sqlite"`。

好处：
- 整个项目**无 CGo 依赖**（托盘除外），可直接 `CGO_ENABLED=0` 交叉编译 Linux 二进制
- 为将来迁移到 [libsql/Turso](https://turso.tech) 奠定基础（接口兼容，改驱动名即可）

涉及文件：
```
internal/trace/storage.go
internal/skill/usage_store.go
internal/agent/manager.go
internal/agent/router.go
cmd/migrate-db/main.go
cmd/migrate-team/main.go
go.mod（移除 mattn/go-sqlite3）
```

#### Mode 配置字段

`config.yaml` / `internal/config/config.go`：

```yaml
app:
  mode: solo   # solo | team | cloud
```

新建 `internal/mode/mode.go`：

```go
type Mode string
const (Solo Mode = "solo"; Team Mode = "team"; Cloud Mode = "cloud")

func (m Mode) RequireAuth() bool   { return m == Team || m == Cloud }
func (m Mode) IsMultiUser() bool   { return m == Team || m == Cloud }
func (m Mode) RequireInvite() bool { return m == Cloud }
```

---

### Step 2：Solo 模式桌面体验

#### 认证中间件 Mode 感知

`internal/server/middleware.go` — `WebAuth` 函数：

```
solo  → 直接通过，无需任何 token
team  → 检查 JWT Bearer header 或 session cookie
cloud → 同 team
```

#### 系统托盘（可选，-tags tray）

新增 `internal/tray/`，采用 build tag 分离：

```
tray_desktop.go  // go:build tray    — 使用 getlantern/systray（含 CGo）
tray_stub.go     // go:build !tray   — 纯 Go no-op，服务器/Docker 默认
```

启动方式：
```bash
# 需要先用 make build-desktop（-tags tray）编译
gopaw start --tray --mode solo
```

功能：系统托盘图标 → "Open GoPaw"（打开浏览器）/ "Quit GoPaw"（优雅退出）

#### 浏览器自动打开

solo 模式或 `--tray` 启动后，延迟 800ms 自动调用系统 `open` / `xdg-open` 打开浏览器。

新增 `--no-browser` 标志可禁用（`make dev` 下自动传入，避免 dev 模式打开错误端口）。

#### 新 API 端点

| 端点 | 说明 |
|------|------|
| `GET /api/mode` | 公开，返回当前模式和认证要求，供前端初始化判断 |
| `GET /api/auth/me` | 返回当前登录用户信息（team/cloud 模式） |

---

### Step 3：多用户支持（team 模式）

#### internal/user/ 包

新建三个文件：

| 文件 | 内容 |
|------|------|
| `user.go` | `User` 结构体，`Role`（admin/member） |
| `store.go` | SQLite CRUD：Create/GetByUsername/GetByID/List/Update/Delete/Count |
| `service.go` | 业务逻辑：CreateUser、Authenticate（bcrypt）、EnsureAdminExists、SetPassword、SetActive |

**数据库表**（自动迁移，写入 gopaw.db）：

```sql
CREATE TABLE IF NOT EXISTS users (
    id            TEXT PRIMARY KEY,
    username      TEXT NOT NULL UNIQUE,
    email         TEXT,
    role          TEXT NOT NULL DEFAULT 'member',  -- admin | member
    password_hash TEXT NOT NULL,                    -- bcrypt
    is_active     INTEGER NOT NULL DEFAULT 1,
    created_at    TIMESTAMP NOT NULL,
    updated_at    TIMESTAMP NOT NULL
);
```

#### JWT 认证集成

复用现有 `internal/auth` 包（已有 `GenerateToken` / `ValidateToken`）。

team/cloud 模式登录流程：
```
POST /api/auth/login {username, password}
  → user.Service.Authenticate()
  → auth.Service.GenerateToken()
  → Set-Cookie: gopaw_session=<JWT>
  → 返回 {user_id, username, role, access_token}
```

#### 用户管理 API

新建 `internal/server/handlers/users.go`，仅在 `IsMultiUser()` 时注册路由：

```
GET    /api/users          列出所有用户（admin）
POST   /api/users          创建用户（admin）
DELETE /api/users/:id      删除用户（admin）
PUT    /api/users/:id/active  启用/禁用账号（admin）
```

#### 首次启动初始化

team/cloud 模式启动时，若数据库中无用户，自动创建：

```
username: admin
password: <与 admin_token 相同>
role:     admin
```

---

### 端口与开发体验

#### 端口调整（防止与其他项目冲突）

| 用途 | 旧端口 | 新端口 |
|------|--------|--------|
| Go 后端 | 8088 | **16688** |
| Vite dev server | 5173 | **6673** |

#### Makefile 重构

```bash
make dev            # 双进程 HMR（Vite:6673 + Go:16688），最常用
make dev-go         # 只起 Go 后端（另开终端跑前端）
make dev-embedded   # 单进程：先 bun build，再 Go 内嵌提供，无 HMR
make run-solo       # 生产 solo 模式，自动打开浏览器
make run-team       # 生产 team 模式，JWT 多用户
make run-tray       # 系统托盘模式（build-desktop 版本）
make build          # 服务器版：嵌入前端，无 CGo，适合 Linux/Docker
make build-desktop  # 桌面版：-tags tray，含系统托盘
make build-linux    # CGO_ENABLED=0 交叉编译 Linux amd64
```

#### 单进程架构说明

```
生产（make build）      → 单文件，前端 embed 到 Go 二进制     ✅ 单进程
Docker（make build-linux）→ 单文件，纯 Go，无 CGo              ✅ 单进程
开发（make dev-embedded）→ 先 bun build，再 Go 内嵌运行       ✅ 单进程
开发（make dev）         → Vite(HMR) + Go 双进程，热更新必须   ⚡ 双进程
```

#### 前端构建修复（vite.config.ts 副本问题）

**根本原因**：`tsconfig.node.json` 的 `"composite": true` + `vue-tsc -b` 会强制 TypeScript 输出 `vite.config.js`，导致 Vite 优先读取旧的 `.js` 文件而非 `.ts`。

**修复**：
- 删除 `tsconfig.node.json`（composite 模式不再需要）
- `tsconfig.json` 直接 include `vite.config.ts`
- `package.json` 构建脚本 `vue-tsc -b` → `vue-tsc --noEmit`（只做类型检查，不输出文件）

---

## 当前目录结构（新增部分）

```
GoPaw/
├── internal/
│   ├── mode/
│   │   └── mode.go              # Mode 类型定义
│   ├── tray/
│   │   ├── tray_desktop.go      # go:build tray（getlantern/systray）
│   │   └── tray_stub.go         # go:build !tray（no-op，服务器默认）
│   ├── user/
│   │   ├── user.go              # User 结构体 + Role 常量
│   │   ├── store.go             # SQLite CRUD
│   │   └── service.go           # 业务逻辑（bcrypt 密码）
│   └── server/
│       └── handlers/
│           └── users.go         # 用户管理 HTTP handlers
└── Makefile                     # 重构，新增多个开发目标
```

---

## 待实现（Step 4）

| 功能 | 说明 |
|------|------|
| 前端登录页 | team/cloud 模式的登录界面（`/login` 路由） |
| 前端用户管理页 | Admin 管理用户，仅 `isMultiUser` 时侧边栏显示 |
| cloud 邀请码 | `internal/invite/` 包，注册时校验 |
| 打包脚本 | macOS DMG：`go build` + `create-dmg` |
| 移植 collaboration | gopaw-desk 的多 Agent 协作能力 |
| libsql/Turso 迁移 | 现在是 modernc.org/sqlite，未来可切换到 libsql 支持云端数据库 |

---

## 快速上手

```bash
# 安装前端依赖
make web-install

# 开发模式（双进程 HMR）
make dev

# 单进程开发（无 HMR，调试后端用）
make dev-embedded

# 生产构建并运行
make run-solo          # 个人模式
make run-team          # 团队模式

# Docker 构建
make build-linux       # 生成 gopaw-linux 二进制
make docker-build      # 构建镜像
```

### Solo 模式（默认，无需任何配置）

```bash
./gopaw start               # 浏览器自动打开 http://localhost:16688
./gopaw start --tray        # 系统托盘模式（需 build-desktop 编译）
./gopaw start --mode solo   # 显式指定
```

### Team 模式（多用户）

```bash
./gopaw start --mode team
# 首次启动自动创建 admin 账号，密码与 config.yaml admin_token 相同
# 访问 http://localhost:16688 → 登录页 → 用户名 admin + 密码
```

---

## 依赖变化

| 依赖 | 变化 | 原因 |
|------|------|------|
| `mattn/go-sqlite3` | **移除** | 替换为纯 Go 驱动 |
| `modernc.org/sqlite` | 已有，统一全部使用 | 纯 Go，无 CGo |
| `github.com/getlantern/systray` | 新增（仅 -tags tray） | 系统托盘 |
| `golang.org/x/crypto` | 已有，新增使用 | bcrypt 密码哈希 |
| `github.com/golang-jwt/jwt/v5` | 已有，新增使用 | team/cloud JWT |
