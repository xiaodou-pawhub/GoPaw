# Docker Root 权限配置指南

## 📋 概述

GoPaw Docker 容器默认配置了必要的 Linux capabilities 和 root 启动权限，用于初始化时自动修复挂载目录权限，完成后会自动降权为普通用户运行服务。

---

## 🔧 当前配置

### 默认配置（已启用）

```yaml
services:
  gopaw:
    # 必要的 Linux capabilities，用于文件操作等系统任务
    cap_add:
      - CHOWN        # 允许修改文件所有者
      - SETUID       # 允许设置用户 ID
      - SETGID       # 允许设置组 ID
      - DAC_OVERRIDE # 允许读写任何文件（系统任务需要）
    
    # 容器运行用户
    # 0:0 表示以 root 启动，初始化时会自动修复挂载目录权限
    # 初始化完成后会自动降权为 node 用户运行服务
    user: "0:0"
```

---

## 📝 Linux Capabilities 说明

### 已启用的能力

| 能力 | 说明 | 使用场景 |
|------|------|---------|
| **CHOWN** | 修改文件所有者 | 初始化时修复挂载目录权限 |
| **SETUID** | 设置用户 ID | 进程降权运行需要 |
| **SETGID** | 设置组 ID | 进程降权运行需要 |
| **DAC_OVERRIDE** | 绕过文件权限检查 | 系统任务需要读写文件 |

### 安全说明

这些 capabilities 是**最小必要权限**，用于：
1. ✅ 启动时修复挂载目录的所有权和权限
2. ✅ 完成初始化后自动降权为普通用户
3. ✅ 符合最小权限原则

---

## 🚀 使用方式

### 本地自建模式

```bash
# 启动服务（自动应用 root 权限配置）
docker compose up -d

# 查看容器运行用户
docker compose exec gopaw id
# 输出：uid=0(root) gid=0(root) groups=0(root)
```

### 在线拉取模式

```bash
# 使用在线镜像启动
docker compose -f docker-compose.online.yml up -d
```

---

## 🔍 验证配置

### 1. 检查运行用户

```bash
docker compose exec gopaw id
```

**期望输出**：
```
uid=0(root) gid=0(root) groups=0(root)
```

### 2. 检查 capabilities

```bash
docker compose exec gopaw cat /proc/1/status | grep Cap
```

**期望输出**（部分）：
```
CapInh: 0000000000000000
CapPrm: 0000000000000031  # 包含 CHOWN, SETUID, SETGID, DAC_OVERRIDE
CapEff: 0000000000000031
```

### 3. 检查挂载目录权限

```bash
# 查看容器内数据目录
docker compose exec gopaw ls -la /app/data

# 查看宿主机数据目录
ls -la data/
```

---

## ⚠️ 安全说明

### 为什么需要 root 启动？

1. **挂载目录权限修复**
   - 首次启动时，挂载的 `./data` 和 `./logs` 目录可能属于宿主机的 root 用户
   - 容器内需要 root 权限来修复这些目录的所有权

2. **自动降权机制**
   - 初始化完成后，服务会自动降权为普通用户（如 node）运行
   - 符合安全最佳实践

### 安全性保障

| 措施 | 说明 |
|------|------|
| **最小 capabilities** | 只授予必要的 4 个能力 |
| **自动降权** | 初始化后自动切换为普通用户 |
| **只读配置** | config.yaml 以只读方式挂载 |
| **隔离运行** | 容器内运行，与宿主机隔离 |

---

## 🔧 自定义配置

### 如果需要更多权限

根据实际需求添加 capabilities：

```yaml
cap_add:
  - CHOWN
  - SETUID
  - SETGID
  - DAC_OVERRIDE
  - NET_ADMIN      # 网络管理（如需要）
  - SYS_PTRACE     # 进程调试（如需要）
```

### 如果不需要 root 启动

**前提条件**：
- 挂载目录的权限已经正确设置
- 或者使用命名卷（named volumes）

**配置方法**：

```yaml
services:
  gopaw:
    user: "1000:1000"  # 使用普通用户
    # 移除 cap_add 或保留必要的 capabilities
```

---

## 🐛 故障排查

### 问题 1：容器启动失败

**错误信息**：
```
permission denied
```

**可能原因**：
- Docker 守护进程权限不足
- SELinux/AppArmor 限制

**解决方案**：
```bash
# 检查 Docker 状态
sudo systemctl status docker

# 临时禁用 SELinux（测试用）
setenforce 0

# 或者添加 SELinux 标签
volumes:
  - ./data:/app/data:z
```

### 问题 2：挂载目录权限错误

**错误信息**：
```
EACCES: permission denied, open '/app/data/gopaw.db'
```

**解决方案**：
```bash
# 在宿主机上修复权限
sudo chown -R 1000:1000 data/ logs/

# 或者让容器自动修复（使用 root 启动）
docker compose restart gopaw
```

### 问题 3：capabilities 未生效

**验证方法**：
```bash
docker compose exec gopaw cat /proc/1/status | grep Cap
```

**如果 capabilities 不正确**：
1. 检查 `docker-compose.yml` 配置是否正确
2. 重启容器：`docker compose up -d --force-recreate`
3. 检查 Docker 版本（需要 17.09+）

---

## 📚 相关资源

- [Linux Capabilities 详解](https://man7.org/linux/man-pages/man7/capabilities.7.html)
- [Docker 安全最佳实践](https://docs.docker.com/engine/security/)
- [Docker user 命名空间](https://docs.docker.com/engine/security/userns-remap/)

---

**最后更新**: 2026-03-07
