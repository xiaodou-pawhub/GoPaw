# GoPaw 部署指南

本文档说明如何在本地完成构建，并将 GoPaw 部署到 Linux 服务器（`linux/amd64`）。

---

## 前置要求

| 环境 | 要求 |
|------|------|
| 本地开发机 | Go 1.24+、Node.js 18+、pnpm、Docker |
| 服务器 | Docker 24+、docker compose plugin |

---

## 一、本地构建

GoPaw 的镜像构建策略是**本地交叉编译 → 上传镜像**，服务器不需要 Go 源码和开发环境。

### 1.1 一步构建镜像

```bash
# 在 GoPaw 项目根目录执行
make docker-build
```

此命令依次完成：
1. `pnpm build` 构建前端，嵌入 Go 二进制
2. `CGO_ENABLED=0 GOOS=linux GOARCH=amd64` 交叉编译 Linux 二进制
3. `docker build --platform linux/amd64` 打包镜像

> **Apple Silicon（M 系列芯片）用户**：`--platform linux/amd64` 已写入 Makefile，构建结果是 `linux/amd64` 镜像，与服务器架构匹配。

### 1.2 导出镜像为文件

```bash
docker save gopaw:latest | gzip > gopaw.tar.gz
```

---

## 二、准备服务器部署文件

需要上传到服务器的文件共 **3 个**：

```
gopaw.tar.gz          # Docker 镜像（上一步导出）
docker-compose.yml    # 编排文件（项目根目录）
config.yaml           # 启动配置（见下方说明）
```

### 2.1 准备 config.yaml

`config.yaml` 只包含**系统启动参数**，不需要填写任何 API Key 或平台凭据（这些均通过 Web UI 配置）。

从示例文件复制并按需修改：

```bash
cp config.yaml.example config.yaml
```

**通常只需关注以下字段：**

```yaml
workspace:
  dir: /app/data      # Docker 部署固定使用此路径，无需改动

app:
  timezone: Asia/Shanghai   # 按需修改时区

server:
  host: 0.0.0.0
  port: 8088          # 如需修改端口，docker-compose.yml 中也要同步修改

log:
  level: info         # 生产环境建议 info；排查问题时改为 debug
  format: json        # 生产环境建议 json（结构化日志）
  output: stdout      # Docker 场景使用 stdout，日志由 docker 管理
```

> **不需要在 config.yaml 里填写的内容：**
> LLM 提供商配置（API Key、模型）、飞书/钉钉频道凭据、Agent 设定、工具配置等，
> 均通过 GoPaw Web UI 配置，存储在 SQLite 数据库中，重启不丢失。

---

## 三、配置 Web UI 访问密码（admin_token）

GoPaw Web UI 通过 `admin_token` 验证访问权限。有两种配置方式：

### 方式 A：通过 .env 文件固定设置（推荐）

```bash
# 复制示例文件
cp .env.example .env

# 编辑 .env，设置你的密码
# GOPAW_ADMIN_TOKEN=your-admin-token-here
```

docker compose 启动时自动读取 `.env`，将 `GOPAW_ADMIN_TOKEN` 注入容器，
GoPaw 内部通过 `APP_ADMIN_TOKEN` 环境变量读取（Viper 自动映射 `app.admin_token`）。

> **为什么不写在 config.yaml 里？**
> `.env` 不应提交到 Git，适合存放密码等敏感值；config.yaml 通常会进入版本管理，不适合写密码。

### 方式 B：每次重启自动生成（简便但不适合生产）

`docker-compose.yml` 中 `APP_ADMIN_TOKEN` 留空（默认），GoPaw 启动时自动生成随机 token 并打印到日志：

```bash
# 查看自动生成的 token
docker compose logs gopaw | grep "admin token"
```

重启后 token 会变化，需要重新查日志。适合临时测试，**不建议生产使用**。

---

## 四、上传到服务器

```bash
# 在服务器上创建部署目录
ssh user@your-server "mkdir -p ~/gopaw"

# 上传文件（含 .env）
scp gopaw.tar.gz docker-compose.yml config.yaml .env user@your-server:~/gopaw/
```

---

## 四、服务器端操作

```bash
# SSH 登录服务器
ssh user@your-server
cd ~/gopaw

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

首次访问需要完成初始化设置（管理员账号、LLM 提供商配置等）。

---

## 五、启用浏览器控制工具（可选）

`browser_control` 工具需要 Headless Chrome，通过 Docker Sidecar 方式提供。

```bash
# 含浏览器控制的启动方式
docker compose --profile browser up -d
```

这会额外启动 `chromedp/headless-shell` 容器，GoPaw 通过 CDP WebSocket 连接它。
不需要浏览器功能时，不加 `--profile browser` 即可，`browser_control` 工具调用时会返回连接错误。

---

## 六、技能目录（可选）

自定义技能放在服务器的 `~/.gopaw/skills/` 目录，已在 `docker-compose.yml` 中挂载：

```bash
# 在服务器上创建技能目录
mkdir -p ~/.gopaw/skills

# 上传技能（示例）
scp -r skills/translator user@your-server:~/.gopaw/skills/
```

修改技能文件后需重启 GoPaw 生效：

```bash
docker compose restart gopaw
```

---

## 七、常用运维命令

```bash
# 停止服务
docker compose down

# 更新镜像并重启（重复执行第一步 ~ 第四步后）
docker load < gopaw.tar.gz
docker compose up -d --force-recreate

# 查看实时日志
docker compose logs -f gopaw

# 进入容器排查问题
docker compose exec gopaw sh

# 查看数据 volume 位置
docker volume inspect gopaw_gopaw-data
```

---

## 八、数据持久化说明

| 数据类型 | 存储位置 | 说明 |
|---------|---------|------|
| SQLite 数据库（设置、记忆） | Docker volume `gopaw-data` | 随 volume 持久化，`docker compose down` 不会删除 |
| 应用日志 | Docker volume `gopaw-logs` | 同上 |
| 技能文件 | 服务器 `~/.gopaw/skills/` | bind-mount，直接编辑文件即可 |
| 配置文件 | `~/gopaw/config.yaml` | bind-mount，修改后重启生效 |

> **备份数据**：备份 `gopaw-data` volume 即可保留所有运行时数据：
> ```bash
> docker run --rm -v gopaw_gopaw-data:/data -v $(pwd):/backup alpine \
>   tar czf /backup/gopaw-data-backup.tar.gz -C /data .
> ```
