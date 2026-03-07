# GoPaw 部署指南

本文档说明如何将 GoPaw 部署到 Linux 服务器（`linux/amd64`）。
部署策略：**本地构建镜像 → 导出文件 → 上传服务器运行**，服务器无需 Go 源码和开发环境。

---

## 前置要求

| 环境 | 要求 |
|------|------|
| 本地开发机 | Go 1.24+、Node.js 18+、pnpm、Docker |
| 服务器 | Docker 24+、docker compose plugin |

---

## 一、本地构建

在 GoPaw 项目根目录执行：

```bash
make docker-build
```

此命令依次完成：
1. 构建前端并嵌入 Go 二进制
2. 交叉编译 Linux amd64 二进制（`CGO_ENABLED=0 GOOS=linux GOARCH=amd64`）
3. 打包 Docker 镜像（`--platform linux/amd64`，Apple Silicon 也可正常构建）

导出镜像为文件：

```bash
docker save gopaw:latest | gzip > gopaw.tar.gz
```

---

## 二、准备配置文件

从示例文件复制并修改：

```bash
cp config.yaml.example config.yaml
```

**Docker 部署必须修改的字段：**

```yaml
workspace:
  dir: /app/data        # 容器内固定路径，不要修改

app:
  admin_token: "your-password-here"   # Web UI 访问密码，留空则每次启动随机生成
  timezone: Asia/Shanghai             # 按需修改时区

log:
  format: json          # 生产环境推荐 json
  output: stdout        # Docker 场景固定用 stdout
```

> LLM 提供商（API Key、模型）、频道凭据、Agent 设定等均通过 Web UI 配置，无需写入 config.yaml。

---

## 三、上传到服务器

```bash
# 在服务器上创建部署目录
ssh user@your-server "mkdir -p ~/gopaw"

# 上传部署文件
scp gopaw.tar.gz docker-compose.yml config.yaml user@your-server:~/gopaw/
```

---

## 四、服务器端启动

```bash
ssh user@your-server
cd ~/gopaw

# 导入镜像
docker load < gopaw.tar.gz

# 启动服务
docker compose up -d

# 查看状态
docker compose ps

# 查看日志
docker compose logs -f gopaw
```

服务启动后访问：`http://your-server:8088`
首次访问需完成初始化向导（配置 LLM 提供商等）。

> **忘记 admin_token？** 留空时系统自动生成，查看方式：
> ```bash
> docker compose logs gopaw | grep -i "admin token"
> ```

---

## 五、启用浏览器控制工具（可选）

`browser_control` 工具需要 Headless Chrome sidecar，**首次使用需构建镜像**（已预装中文字体）：

```bash
# 构建 Chrome 镜像（仅首次，或更新时重新构建）
docker compose --profile browser build

# 启动（含浏览器）
docker compose --profile browser up -d
```

不需要浏览器功能时，直接 `docker compose up -d` 即可，`browser_control` 工具调用时会返回连接错误。

---

## 六、添加技能

技能存放在工作区的 `skills/` 子目录，对应服务器上的 `~/gopaw/data/skills/`，随 `./data` 挂载自动生效。

```bash
# 上传技能
scp -r skills/translator user@your-server:~/gopaw/data/skills/
```

上传后无需重启，通过对话触发加载：

```
你：重新加载技能
```

或在 Web UI `设置 → 技能` 页面点击「重新加载」按钮。

---

## 七、更新版本

```bash
# 本地重新构建
make docker-build
docker save gopaw:latest | gzip > gopaw.tar.gz

# 上传并更新
scp gopaw.tar.gz user@your-server:~/gopaw/
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

# 重启服务
docker compose restart gopaw
```

---

## 九、数据说明

所有数据存在部署目录的子目录下，`docker compose down` 不会删除。

| 路径 | 内容 |
|------|------|
| `./data/` | SQLite 数据库、记忆文件、技能目录 |
| `./logs/` | 应用日志（如需写文件，config.yaml 中 `log.output: both`）|
| `./config.yaml` | 启动配置，修改后 `docker compose restart gopaw` 生效 |

**备份：**

```bash
tar czf gopaw-backup-$(date +%Y%m%d).tar.gz data/ config.yaml
```
