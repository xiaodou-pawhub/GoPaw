# GoPaw 部署指南

本文档说明如何在本地完成构建，并将 GoPaw 部署到 Linux 服务器（`linux/amd64`）。

---

## 前置要求

| 环境 | 要求 |
|------|------|
| 本地开发机 | Go 1.24+、Node.js 18+、pnpm、Docker |
| 服务器 | Docker 24+、docker compose plugin |

---

## 一、本地构建镜像

GoPaw 的构建策略是**本地交叉编译 → 导出镜像文件 → 上传服务器**，服务器不需要 Go 源码和开发环境。

### 1.1 一步构建镜像

```bash
# 在 GoPaw 项目根目录执行
make docker-build
```

此命令依次完成：
1. `pnpm build` 构建前端并嵌入 Go 二进制
2. `CGO_ENABLED=0 GOOS=linux GOARCH=amd64` 交叉编译 Linux 二进制
3. `docker build --platform linux/amd64` 打包镜像

> **Apple Silicon（M 系列芯片）用户**：`--platform linux/amd64` 已写入 Makefile，构建结果是 `linux/amd64` 镜像，与服务器架构匹配。

### 1.2 导出镜像为文件

```bash
docker save gopaw:latest | gzip > gopaw.tar.gz
```

---

## 二、准备部署文件

在服务器上新建一个部署目录（如 `~/gopaw`），以下文件都放在这个目录里：

```
~/gopaw/
├── gopaw.tar.gz        # Docker 镜像（第一步导出）
├── docker-compose.yml  # 编排文件（从项目根目录复制）
├── config.yaml         # 启动配置（见 2.1）
└── .env                # 访问密码等敏感配置（见 2.2）
```

运行后会自动创建：

```
~/gopaw/
├── data/               # SQLite 数据库、记忆文件等持久化数据
│   └── skills/         # 用户技能目录（自动创建，放入技能后调用 reload API 即可生效）
└── logs/               # 应用日志
```

### 2.1 准备 config.yaml

`config.yaml` 只包含**系统启动参数**，不需要填写任何 API Key 或平台凭据（这些均通过 Web UI 配置，存储在 SQLite）。

从示例文件复制并修改：

```bash
cp config.yaml.example config.yaml
```

**Docker 部署只需关注以下字段：**

```yaml
workspace:
  dir: /app/data        # 容器内固定路径，勿改

app:
  timezone: Asia/Shanghai   # 按需修改时区

server:
  host: 0.0.0.0
  port: 8088            # 如需改端口，docker-compose.yml 的 ports 也要同步

log:
  level: info           # 排查问题时改为 debug
  format: json          # 生产环境推荐 json（结构化日志）
  output: stdout        # Docker 场景用 stdout，日志由 Docker 统一管理
```

> LLM 提供商（API Key、模型）、飞书/钉钉频道凭据、Agent 设定等，均通过 Web UI 配置，无需写入 config.yaml。

### 2.2 准备 .env（配置访问密码）

GoPaw Web UI 通过 `admin_token` 验证访问权限，建议通过 `.env` 文件固定设置。

```bash
# 复制示例文件
cp .env.example .env

# 编辑 .env，将占位值替换为你的密码
# GOPAW_ADMIN_TOKEN=your-admin-token-here
```

`.env` 由 docker compose 自动读取，`GOPAW_ADMIN_TOKEN` 会注入到容器的 `APP_ADMIN_TOKEN` 环境变量。

**如果 `.env` 中留空**，GoPaw 每次启动时自动生成随机 token 并打印到日志，重启后 token 会变化：

```bash
docker compose logs gopaw | grep "Admin token"
```

> `.env` 包含敏感信息，已加入 `.gitignore`，不会提交到 Git。

---

## 三、上传到服务器

```bash
# 在服务器上创建部署目录
ssh user@your-server "mkdir -p ~/gopaw"

# 上传所有部署文件
scp gopaw.tar.gz docker-compose.yml config.yaml .env.example user@your-server:~/gopaw/
```

---

## 四、服务器端操作

```bash
# SSH 登录服务器
ssh user@your-server
cd ~/gopaw

# 从示例创建 .env 并设置访问密码
cp .env.example .env
# 编辑 .env，填写 GOPAW_ADMIN_TOKEN
nano .env

# 导入镜像
docker load < gopaw.tar.gz

# 启动服务（基础模式，不含浏览器控制）
docker compose up -d

# 查看服务状态
docker compose ps

# 查看日志
docker compose logs -f gopaw
```

服务启动后访问：`http://your-server:8088`

首次访问需完成初始化向导（设置 LLM 提供商等），之后正常使用。

---

## 五、启用浏览器控制工具（可选）

`browser_control` 工具依赖 Headless Chrome，通过 Docker Sidecar 方式提供：

```bash
# 含浏览器控制的启动方式
docker compose --profile browser up -d
```

这会额外拉取并启动 `chromedp/headless-shell` 容器，GoPaw 通过 CDP WebSocket 连接它。
不需要浏览器功能时不加 `--profile browser` 即可，`browser_control` 工具调用时会返回连接错误。

---

## 六、技能目录（可选）

技能统一存放在工作区的 `skills/` 子目录，Docker 部署中对应宿主机的 `./data/skills/`，随 `./data` 挂载自动生效，无需额外配置。

```bash
# 在服务器上创建技能目录（首次启动后 data/ 已自动创建，此步骤可选）
mkdir -p ~/gopaw/data/skills

# 上传技能
scp -r skills/translator user@your-server:~/gopaw/data/skills/
```

上传后通过 API 动态加载，**无需重启**：

```bash
curl -X POST http://your-server:8088/api/skills/reload \
  -H "Authorization: Bearer <admin-token>"
```

或在 Web UI `设置 → 技能` 页面点击「重新加载」按钮。

---

## 七、更新镜像

```bash
# 本地重新构建并导出
make docker-build
docker save gopaw:latest | gzip > gopaw.tar.gz

# 上传新镜像
scp gopaw.tar.gz user@your-server:~/gopaw/

# 服务器端更新
ssh user@your-server "cd ~/gopaw && docker load < gopaw.tar.gz && docker compose up -d --force-recreate"
```

---

## 八、常用运维命令

```bash
# 停止服务（数据目录保留）
docker compose down

# 查看实时日志
docker compose logs -f gopaw

# 进入容器排查问题
docker compose exec gopaw sh

# 重启单个服务
docker compose restart gopaw
```

---

## 九、数据持久化说明

所有数据存在部署目录的子目录下，清晰可见，直接备份文件夹即可。

| 数据类型 | 路径 | 说明 |
|---------|------|------|
| SQLite 数据库（设置、记忆） | `./data/` | `docker compose down` 不会删除 |
| 应用日志 | `./logs/` | 同上 |
| 技能文件 | `./skills/` | 直接编辑文件，重启生效 |
| 启动配置 | `./config.yaml` | 修改后重启生效 |
| 访问密码 | `./.env` | 修改后重启生效 |

**备份：**

```bash
# 备份所有持久化数据（在部署目录执行）
tar czf gopaw-backup-$(date +%Y%m%d).tar.gz data/ config.yaml .env
```
