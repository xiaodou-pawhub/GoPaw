# GoPaw 技能市场 产品设计文档

> 版本：v0.1（MVP）
> 状态：设计阶段
> 最后更新：2026-03-07

---

## 目录

1. [产品概述](#1-产品概述)
2. [MVP 范围](#2-mvp-范围)
3. [安装协议](#3-安装协议)
4. [技术架构](#4-技术架构)
5. [技术选型](#5-技术选型)
6. [数据库设计](#6-数据库设计)
7. [API 设计](#7-api-设计)
8. [页面设计](#8-页面设计)
9. [用户流程](#9-用户流程)
10. [部署方案](#10-部署方案)
11. [开发任务清单](#11-开发任务清单)

---

## 1. 产品概述

### 1.1 是什么

GoPaw 技能市场（SkillHub）是一个托管、分发 GoPaw 技能（Skill）的网站平台。用户可以在这里发现、预览、一键安装社区或官方提供的技能包，扩展本地 GoPaw 的能力。

类比：VS Code 的扩展市场、Homebrew 的公式库。

### 1.2 目标用户

| 用户类型 | 需求 |
|---------|------|
| GoPaw 普通用户 | 找到好用的技能，一键安装，无需自己编写 |
| GoPaw 开发者 | 发布自己的技能，让更多人使用 |
| GoPaw 团队 | 运营精选技能、建立生态、积累用户认知 |

### 1.3 核心价值

- **发现**：集中展示所有可用技能，分类清晰，可搜索
- **信任**：官方精选 + 社区评分，降低选择成本
- **便捷**：一键安装，无需手动下载复制文件
- **习惯**：让用户形成"找技能去 SkillHub"的认知

### 1.4 阶段规划

| 阶段 | 目标 | 特征 |
|------|------|------|
| MVP | 让用户认识这个网站 | 官方精选技能，浏览+安装，GitHub 登录 |
| V1 | 社区参与 | 用户提交技能，审核上架 |
| V2 | 商业化 | 付费技能，创作者分成 |

---

## 2. MVP 范围

### 2.1 包含功能

- 技能列表页（浏览、分类筛选、搜索）
- 技能详情页（介绍、安装方式、版本历史）
- GitHub OAuth 登录
- 一键安装（触发 `gopaw://` 协议或展示 CLI 命令）
- 安装统计（记录安装次数）
- 管理后台（团队内部上架、下架技能）

### 2.2 不包含（留到后续版本）

- 用户自主提交技能
- 付费技能 / 分成系统
- 评分 / 评论
- 个人主页 / 收藏
- 技能依赖管理

---

## 3. 安装协议

### 3.1 URL Scheme

桌面端 GoPaw 注册系统级 URL Scheme，网站点击"安装"时唤起本地应用：

```
gopaw://skill/install?name=translator&source=market&version=1.0.0
```

| 参数 | 必填 | 说明 |
|------|------|------|
| `name` | 是 | 技能唯一标识符（英文，小写，连字符） |
| `source` | 否 | 来源，默认 `market` |
| `version` | 否 | 指定版本，不填默认 `latest` |

**触发逻辑：**

```
用户点击"安装"按钮
  ├─ 检测本地是否安装 GoPaw 桌面端
  │   ├─ 已安装 → window.location = "gopaw://skill/install?..."
  │   └─ 未安装 → 展示备用方案弹窗
  │       ├─ CLI 命令：gopaw skill install translator
  │       └─ 手动下载链接（zip 包）
```

### 3.2 CLI 命令（备用）

适用于 Docker 部署、命令行用户：

```bash
gopaw skill install translator
gopaw skill install translator --version 1.0.0
gopaw skill list --installed
gopaw skill remove translator
```

### 3.3 Web UI 安装（Docker 用户）

GoPaw Web 界面内嵌市场入口，直接调用本地 API 安装，不依赖 URL Scheme：

```
POST http://localhost:8088/api/skills/install
Body: { "name": "translator", "source": "market", "version": "latest" }
```

### 3.4 技能包格式

每个技能是一个 zip 包，解压后结构：

```
translator/
├── manifest.yaml      # 技能元信息（必须）
├── prompt.md          # Level 1 技能提示词
├── workflow.yaml      # Level 2 工作流（可选）
└── skill.go           # Level 3 代码技能（可选）
```

---

## 4. 技术架构

### 4.1 总体架构

```
用户浏览器
    │
    ▼
Next.js 应用（前端 + API Routes）
    │
    ├─ 静态页面 / SSR（技能列表、详情）
    ├─ API Routes（/api/*）
    │       │
    │       ├─ PostgreSQL（用户、技能元数据、安装记录）
    │       └─ 对象存储（技能 zip 包）
    │
    └─ GitHub OAuth（NextAuth.js）
```

### 4.2 项目结构

```
gopaw-market/
├── app/                    # Next.js App Router
│   ├── (site)/             # 公开页面
│   │   ├── page.tsx        # 首页
│   │   ├── skills/         # 技能列表
│   │   │   ├── page.tsx
│   │   │   └── [name]/     # 技能详情
│   │   │       └── page.tsx
│   │   └── layout.tsx
│   ├── admin/              # 管理后台（需鉴权）
│   │   ├── skills/
│   │   └── layout.tsx
│   └── api/                # API Routes
│       ├── auth/           # NextAuth
│       ├── skills/         # 技能 CRUD
│       └── install/        # 安装统计
├── components/             # UI 组件
├── lib/                    # 工具函数、DB 客户端
│   ├── db.ts               # Prisma 客户端
│   ├── auth.ts             # NextAuth 配置
│   └── storage.ts          # 对象存储工具
├── prisma/
│   └── schema.prisma       # 数据库 Schema
└── public/
```

---

## 5. 技术选型

### 5.1 推荐栈

| 层级 | 技术 | 理由 |
|------|------|------|
| 框架 | Next.js 14（App Router） | SSR 利于 SEO，前后端一体，部署简单 |
| UI | Tailwind CSS + shadcn/ui | 组件齐全，风格简洁，开发效率高 |
| 数据库 | PostgreSQL | 成熟稳定，支持全文搜索 |
| ORM | Prisma | 类型安全，迁移管理方便 |
| 认证 | NextAuth.js v5 | GitHub OAuth 开箱即用 |
| 对象存储 | Cloudflare R2 | 免费额度大，无出流量费，兼容 S3 API |
| 部署 | Vercel | 与 Next.js 无缝集成，自动 CI/CD |
| 数据库托管 | Neon（PostgreSQL Serverless） | 免费套餐够 MVP 用，与 Vercel 集成好 |

### 5.2 开发依赖

```json
{
  "dependencies": {
    "next": "^14",
    "react": "^18",
    "next-auth": "^5",
    "@prisma/client": "^5",
    "@aws-sdk/client-s3": "^3",
    "tailwindcss": "^3",
    "zod": "^3"
  },
  "devDependencies": {
    "prisma": "^5",
    "typescript": "^5"
  }
}
```

---

## 6. 数据库设计

### 6.1 Schema（Prisma）

```prisma
// prisma/schema.prisma

generator client {
  provider = "prisma-client-js"
}

datasource db {
  provider = "postgresql"
  url      = env("DATABASE_URL")
}

// 用户（GitHub OAuth）
model User {
  id        String   @id @default(cuid())
  githubId  String   @unique
  username  String
  avatarUrl String?
  email     String?
  role      Role     @default(USER)
  createdAt DateTime @default(now())

  installs  Install[]
}

enum Role {
  USER
  ADMIN
}

// 技能
model Skill {
  id           String      @id @default(cuid())
  name         String      @unique  // 唯一标识，如 translator
  displayName  String                // 展示名，如"多语言翻译助手"
  description  String
  longDesc     String?     @db.Text  // 详细介绍（Markdown）
  author       String                // 作者名
  authorUrl    String?               // 作者主页
  category     Category    @relation(fields: [categoryId], references: [id])
  categoryId   String
  tags         String[]              // 标签数组
  level        Int                   // 1=Prompt / 2=Config / 3=Code
  featured     Boolean     @default(false)   // 精选
  published    Boolean     @default(false)   // 是否上架
  installCount Int         @default(0)
  createdAt    DateTime    @default(now())
  updatedAt    DateTime    @updatedAt

  versions     SkillVersion[]
  installs     Install[]
}

// 技能版本
model SkillVersion {
  id          String   @id @default(cuid())
  skill       Skill    @relation(fields: [skillId], references: [id])
  skillId     String
  version     String                // 如 1.0.0
  changelog   String?               // 更新说明
  packageUrl  String                // zip 包的对象存储 URL
  isLatest    Boolean  @default(false)
  createdAt   DateTime @default(now())

  @@unique([skillId, version])
}

// 分类
model Category {
  id    String  @id @default(cuid())
  name  String  @unique   // 如 productivity、language、dev-tools
  label String            // 展示名，如"效率工具"
  icon  String?           // 图标名（可选）
  order Int     @default(0)

  skills Skill[]
}

// 安装记录
model Install {
  id        String   @id @default(cuid())
  skill     Skill    @relation(fields: [skillId], references: [id])
  skillId   String
  user      User?    @relation(fields: [userId], references: [id])
  userId    String?            // 未登录用户为 null
  source    String             // protocol / cli / webui
  createdAt DateTime @default(now())
}
```

### 6.2 初始分类数据

| name | label |
|------|-------|
| `productivity` | 效率工具 |
| `language` | 语言 & 翻译 |
| `dev-tools` | 开发辅助 |
| `writing` | 写作助手 |
| `data` | 数据处理 |
| `automation` | 自动化 |

---

## 7. API 设计

所有 API 均为 Next.js Route Handlers（`app/api/`）。

### 7.1 技能接口

#### 获取技能列表

```
GET /api/skills
```

Query 参数：

| 参数 | 类型 | 说明 |
|------|------|------|
| `q` | string | 关键词搜索（名称、描述） |
| `category` | string | 分类筛选 |
| `featured` | boolean | 只看精选 |
| `page` | number | 页码，默认 1 |
| `pageSize` | number | 每页数量，默认 20 |

响应：

```json
{
  "skills": [
    {
      "name": "translator",
      "displayName": "多语言翻译助手",
      "description": "支持 50+ 种语言互译",
      "category": { "name": "language", "label": "语言 & 翻译" },
      "tags": ["翻译", "多语言"],
      "level": 1,
      "featured": true,
      "installCount": 1280,
      "latestVersion": "1.2.0"
    }
  ],
  "total": 42,
  "page": 1,
  "pageSize": 20
}
```

#### 获取技能详情

```
GET /api/skills/:name
```

响应：在列表字段基础上增加 `longDesc`、`versions`（版本列表）、`author`、`authorUrl`。

#### 记录安装

```
POST /api/skills/:name/install
```

请求体：

```json
{
  "version": "latest",
  "source": "protocol"   // protocol | cli | webui
}
```

响应：

```json
{
  "ok": true,
  "packageUrl": "https://r2.example.com/skills/translator-1.2.0.zip"
}
```

### 7.2 分类接口

```
GET /api/categories
```

响应：分类列表（含每类技能数量）。

### 7.3 管理接口（需 ADMIN 角色）

```
POST   /api/admin/skills              # 创建技能
PUT    /api/admin/skills/:name        # 更新技能信息
POST   /api/admin/skills/:name/publish    # 上架
POST   /api/admin/skills/:name/unpublish  # 下架
POST   /api/admin/skills/:name/versions   # 上传新版本（附 zip 包）
DELETE /api/admin/skills/:name        # 删除
```

---

## 8. 页面设计

### 8.1 首页 `/`

**布局：**

```
Header（Logo + 搜索框 + 登录按钮）
  ↓
Hero 区（一句话介绍 + 搜索框）
  ↓
精选技能（3~6 个卡片，horizontal scroll）
  ↓
分类导航（图标 + 分类名 + 数量）
  ↓
最新上架（列表，按时间倒序）
  ↓
Footer（GitHub 链接、文档链接、协议）
```

### 8.2 技能列表页 `/skills`

**布局：**

```
左侧边栏：分类筛选 + 标签筛选
主内容区：搜索栏 + 排序（安装数/最新）+ 技能卡片网格
```

**技能卡片包含：**
- 技能名称 + 简短描述
- 分类标签
- 安装次数
- Level 标识（Prompt / Config / Code）
- "安装"按钮

### 8.3 技能详情页 `/skills/:name`

**布局：**

```
顶部：技能名 + 作者 + 安装次数 + [安装] 按钮
  ↓
Tab 切换：
  ├─ 介绍（Markdown 渲染）
  ├─ 版本历史
  └─ 安装说明（三种方式：协议 / CLI / 手动）
```

**安装弹窗（点击"安装"后出现）：**

```
------- 安装 translator -------

如果你安装了 GoPaw 桌面版：
  [立即安装]  ← 触发 gopaw:// 协议

如果你使用 CLI 或 Docker：
  $ gopaw skill install translator
  [复制命令]

手动安装：
  [下载 zip 包] → 解压到 {workspace}/skills/ 目录
-------------------------------
```

### 8.4 管理后台 `/admin`

简单的内部管理界面，功能：
- 技能列表（含上架/下架状态）
- 创建/编辑技能（表单填写元信息）
- 上传技能 zip 包（新版本）
- 一键上架/下架

风格保持简单，不需要精心设计，内部使用即可。

---

## 9. 用户流程

### 9.1 首次访问用户

```
访问首页
  → 浏览精选技能
  → 点击感兴趣的技能
  → 查看详情
  → 点击"安装"
  → 弹窗提示三种安装方式
  → 用户选择适合自己的方式安装
  → （可选）GitHub 登录，安装记录被保存
```

### 9.2 已登录用户

```
GitHub 登录
  → 安装记录自动关联账号
  → （V1）可查看已安装技能列表
  → （V1）可收藏技能
```

### 9.3 管理员上架技能

```
登录后台 /admin
  → 填写技能元信息（名称、描述、分类、标签等）
  → 上传 zip 包（触发存储到 R2）
  → 填写版本号和更新说明
  → 点击"上架"
  → 技能出现在公开列表
```

---

## 10. 部署方案

### 10.1 推荐部署架构

```
Vercel（Next.js 应用）
  ├─ 自动 CI/CD（推送 main 分支触发部署）
  ├─ 环境变量管理
  └─ Edge Network（全球加速）

Neon（PostgreSQL Serverless）
  └─ 免费套餐：3GB 存储，足够 MVP

Cloudflare R2（技能包存储）
  └─ 免费额度：10GB 存储，1000 万次请求/月
```

### 10.2 环境变量

```env
# 数据库
DATABASE_URL=postgresql://...

# GitHub OAuth
AUTH_GITHUB_ID=xxx
AUTH_GITHUB_SECRET=xxx
NEXTAUTH_SECRET=xxx
NEXTAUTH_URL=https://market.gopaw.dev

# Cloudflare R2
R2_ACCOUNT_ID=xxx
R2_ACCESS_KEY_ID=xxx
R2_SECRET_ACCESS_KEY=xxx
R2_BUCKET_NAME=gopaw-skills
R2_PUBLIC_URL=https://skills.gopaw.dev

# 管理员 GitHub 用户名（逗号分隔）
ADMIN_GITHUB_USERNAMES=xiaodou997
```

### 10.3 域名规划

| 地址 | 说明 |
|------|------|
| `market.gopaw.dev` | 技能市场网站 |
| `skills.gopaw.dev` | R2 公开访问（技能 zip 包 CDN） |

---

## 11. 开发任务清单

### Phase 0：项目初始化（0.5 天）

- [ ] 使用 `create-next-app` 初始化项目，配置 TypeScript + Tailwind
- [ ] 安装并配置 shadcn/ui
- [ ] 配置 Prisma + 连接 Neon 数据库
- [ ] 配置 NextAuth.js（GitHub OAuth）
- [ ] 配置 Cloudflare R2 客户端（使用 `@aws-sdk/client-s3`）
- [ ] 执行 `prisma db push` 创建数据表
- [ ] 插入初始分类数据

### Phase 1：核心 API（1 天）

- [ ] `GET /api/skills` — 列表 + 搜索 + 分类筛选 + 分页
- [ ] `GET /api/skills/:name` — 详情（含版本列表）
- [ ] `POST /api/skills/:name/install` — 记录安装，返回下载链接
- [ ] `GET /api/categories` — 分类列表
- [ ] 管理接口：创建技能、上架/下架、上传版本

### Phase 2：前端页面（2 天）

- [ ] 公共组件：Header（Logo + 搜索 + 登录）、Footer、SkillCard
- [ ] 首页：Hero + 精选区 + 分类导航 + 最新上架
- [ ] 技能列表页：侧边栏筛选 + 网格 + 搜索
- [ ] 技能详情页：介绍 Tab + 版本历史 Tab + 安装说明 Tab
- [ ] 安装弹窗：三种安装方式（协议 / CLI 命令 / 手动下载）
- [ ] GitHub 登录页

### Phase 3：管理后台（1 天）

- [ ] 后台布局 + 鉴权中间件（仅 ADMIN 角色可访问）
- [ ] 技能列表管理页（上架/下架开关）
- [ ] 创建/编辑技能表单
- [ ] 上传技能 zip 包（上传到 R2，记录版本）

### Phase 4：内容 + 上线（0.5 天）

- [ ] 录入首批精选技能（建议 10~20 个）
- [ ] 配置 Vercel 部署（绑定域名、环境变量）
- [ ] 配置 R2 公开访问 + 自定义域名
- [ ] 基础 SEO（`metadata`、`og:image`）
- [ ] 测试三种安装方式（协议唤起 / CLI / 手动下载）

### 总工时估算

| Phase | 工时 |
|-------|------|
| Phase 0 | 0.5 天 |
| Phase 1 | 1 天 |
| Phase 2 | 2 天 |
| Phase 3 | 1 天 |
| Phase 4 | 0.5 天 |
| **合计** | **5 天** |

---

## 附录 A：技能元信息示例

上架一个技能时，需填写以下元信息：

```yaml
name: translator
displayName: 多语言翻译助手
description: 支持 50+ 种语言互译，自动识别源语言，输出地道译文
author: xiaodou997
authorUrl: https://github.com/xiaodou997
category: language
tags: [翻译, 多语言, 英语, 日语]
level: 1
```

## 附录 B：与 GoPaw 主仓库的接口约定

技能市场与 GoPaw 本体的交互点：

| 交互方式 | 说明 |
|---------|------|
| `gopaw://skill/install?name=xxx` | 桌面端注册此 URL Scheme 并处理安装 |
| `gopaw skill install xxx` | CLI 命令，从市场 API 下载 zip 并解压到 skills 目录 |
| `POST /api/skills/install`（本地） | Web UI 调用本地 GoPaw API，触发从市场下载安装 |
| 安装目录 | `{workspace}/skills/{name}/`（与现有 skills 目录一致） |

GoPaw 主仓库需实现：
- `gopaw skill install <name>` CLI 子命令
- 桌面端 URL Scheme 注册与处理（gopaw-desk 仓库负责）
- Web UI 的"技能市场入口"入口页（可 iframe 嵌入或跳转）
