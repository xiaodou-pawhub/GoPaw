# GitHub Actions 自动同步到 Gitee 配置指南

## 📋 概述

此工作流会在每次推送到 GitHub `main` 分支时，自动将代码同步到 Gitee 仓库。

---

## 🔧 配置步骤

### 1. 在 Gitee 创建仓库

1. 访问 https://gitee.com
2. 创建新仓库（建议与 GitHub 仓库同名）
3. 复制仓库 SSH 地址，格式：`git@gitee.com:username/repo.git`

### 2. 生成 SSH 密钥对

在本地终端执行：

```bash
# 生成 SSH 密钥（邮箱替换为你的 Gitee 注册邮箱）
ssh-keygen -t rsa -b 4096 -C "your_email@example.com" -f gitee_actions_key

# 按提示操作，可以直接回车跳过密码
```

### 3. 添加公钥到 Gitee

1. 查看公钥内容：
   ```bash
   cat gitee_actions_key.pub
   ```

2. 复制公钥内容（以 `ssh-rsa` 开头的整行）

3. 访问 Gitee → 设置 → SSH 公钥 → 添加公钥
   - 标题：`GitHub Actions Sync`
   - 粘贴公钥内容
   - 保存

### 4. 配置 GitHub Secrets

在 GitHub 仓库中：

1. 进入 **Settings** → **Secrets and variables** → **Actions**

2. 添加以下 Secrets：

| Secret 名称 | 值 | 说明 |
|------------|-----|------|
| `GITEE_RSA_PRIVATE_KEY` | `gitee_actions_key` 文件内容 | SSH 私钥（完整内容，包含 BEGIN/END 行） |
| `GITEE_REPO` | `git@gitee.com:username/repo.git` | Gitee 仓库 SSH 地址 |

**添加示例**：

```
# GITEE_RSA_PRIVATE_KEY
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA...（你的私钥内容）
...
-----END RSA PRIVATE KEY-----

# GITEE_REPO
git@gitee.com:xiaodou997/gopaw.git
```

---

## 🚀 使用方式

### 自动同步

每次 `git push` 到 `main` 分支时自动触发。

### 手动触发

1. 访问 GitHub 仓库 → **Actions** 标签
2. 选择 **Sync to Gitee** 工作流
3. 点击 **Run workflow** 按钮
4. 选择分支（默认 main）
5. 点击 **Run workflow**

---

## 📝 注意事项

### 1. 工作流文件本身

- ✅ 工作流文件（`.github/workflows/sync-to-gitee.yml`）会同步到 Gitee
- ✅ 这是期望行为，保持配置一致
- ℹ️ Gitee 也有 Actions 功能（需单独开启）

### 2. 敏感信息

- ⚠️ **永远不要**将私钥提交到代码库
- ✅ 使用 GitHub Secrets 存储敏感信息
- ✅ 私钥文件 `gitee_actions_key` 应添加到 `.gitignore`

### 3. 首次同步

如果是空仓库首次同步：

```bash
# 在本地执行完整同步
git remote add gitee git@gitee.com:username/repo.git
git push -f gitee main
git push --tags gitee
```

---

## 🔍 故障排查

### 查看同步日志

1. GitHub → Actions → Sync to Gitee
2. 点击最近的运行记录
3. 查看每一步的输出

### 常见错误

| 错误 | 原因 | 解决方案 |
|------|------|---------|
| `Permission denied (publickey)` | SSH 密钥配置错误 | 检查私钥是否正确，公钥是否添加到 Gitee |
| `Could not resolve hostname` | 网络问题 | 检查 Gitee 地址是否正确 |
| `remote: Repository not found` | 仓库不存在或无权限 | 检查 GITEE_REPO 地址，确认有写入权限 |

### 手动测试

```bash
# 在本地测试 SSH 连接
ssh -T git@gitee.com

# 应该显示：
# Hi username! You've successfully authenticated, but Gitee does not provide shell access.
```

---

## 📊 同步策略

### 同步内容

- ✅ `main` 分支代码
- ✅ 所有 Git 标签
- ✅ 工作流配置文件

### 不同步内容

- ❌ GitHub Actions 运行日志
- ❌ GitHub Secrets
- ❌ GitHub 特定的 CI/CD 配置

### 分支策略

- 仅同步 `main` 分支
- 开发分支保留在 GitHub
- Gitee 作为镜像仓库

---

## 🔗 相关资源

- [Gitee SSH 密钥配置](https://gitee.com/help/articles/4181)
- [GitHub Actions Secrets](https://docs.github.com/en/actions/security-guides/encrypted-secrets)
- [Git 远程仓库管理](https://git-scm.com/book/en/v2/Git-Basics-Working-with-Remotes)

---

**配置完成后，每次 push 到 main 分支都会自动同步到 Gitee！** 🎉
