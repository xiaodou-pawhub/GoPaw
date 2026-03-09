# GoPaw API 验证清单

**验证时间**: 2026 年 3 月 8 日
**验证范围**: 核心 API 端点完整性

---

## ✅ P0 - 核心 API（必须验证）

### 模型管理 API

| API 端点 | 方法 | 状态 | 备注 |
|---------|------|------|------|
| `/api/settings/providers` | GET | ⏳ 待测试 | 获取模型列表（按优先级） |
| `/api/settings/providers` | POST | ⏳ 待测试 | 创建/更新模型 |
| `/api/settings/providers/:id` | DELETE | ⏳ 待测试 | 删除模型 |
| `/api/settings/providers/:id/toggle` | POST | ⏳ 待测试 | 启用/禁用模型 |
| `/api/settings/providers/reorder` | POST | ⏳ 待测试 | 批量调整优先级 |
| `/api/settings/providers/capable/:capability` | GET | ⏳ 待测试 | 按能力筛选模型 |
| `/api/settings/providers/health` | GET | ⏳ 待测试 | 健康状态检查 |

### 聊天 API

| API 端点 | 方法 | 状态 | 备注 |
|---------|------|------|------|
| `/api/agent/chat` | POST | ⏳ 待测试 | 发送消息 |
| `/api/agent/chat/stream` | GET | ⏳ 待测试 | 流式响应 |
| `/api/agent/sessions` | GET | ⏳ 待测试 | 获取会话列表 |
| `/api/agent/sessions/:id` | DELETE | ⏳ 待测试 | 删除会话 |

### 频道 API

| API 端点 | 方法 | 状态 | 备注 |
|---------|------|------|------|
| `/api/channels/health` | GET | ⏳ 待测试 | 频道健康状态 |
| `/api/channels/:name/test` | POST | ⏳ 待测试 | 测试频道连接 |

---

## 🔧 测试方法

### 使用 curl 测试

```bash
# 1. 获取模型列表
curl -X GET http://localhost:8088/api/settings/providers \
  -H "Authorization: Bearer YOUR_TOKEN"

# 2. 测试模型启用/禁用
curl -X POST http://localhost:8088/api/settings/providers/PROVIDER_ID/toggle \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"enabled": true}'

# 3. 测试优先级调整
curl -X POST http://localhost:8088/api/settings/providers/reorder \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"provider_ids": ["id1", "id2", "id3"]}'

# 4. 测试能力筛选
curl -X GET http://localhost:8088/api/settings/providers/capable/vision \
  -H "Authorization: Bearer YOUR_TOKEN"

# 5. 测试健康状态
curl -X GET http://localhost:8088/api/settings/providers/health \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### 使用 Postman 测试

导入以下集合：
- GoPaw API Collection (待创建)

---

## 📋 验证步骤

### Step 1: 启动服务
```bash
cd GoPaw
make dev
# 或
./gopaw start
```

### Step 2: 获取 Token
```bash
# 从日志中获取 admin_token
docker logs gopaw | grep "Admin token"
```

### Step 3: 执行测试
按顺序测试每个 API 端点

### Step 4: 记录结果
更新上方表格状态
- ✅ 通过
- ❌ 失败（注明原因）
- ⏳ 待测试

---

## ⚠️ 常见问题

### 401 Unauthorized
- 检查 Token 是否正确
- 检查 Authorization header 格式

### 404 Not Found
- 检查 API 路径是否正确
- 检查路由是否注册

### 500 Internal Server Error
- 查看后端日志
- 检查数据库连接

---

**验证完成时间**: _______
**验证人**: _______
**总体状态**: ✅ 通过 / ⚠️ 部分通过 / ❌ 失败
