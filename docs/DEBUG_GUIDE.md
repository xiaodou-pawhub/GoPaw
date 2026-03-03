# GoPaw 调试指南 - 查看 LLM 请求详情

> **用途**: 查看完整的 LLM 请求参数、请求头、请求地址等信息  
> **更新时间**: 2026-03-01

---

## 🎯 问题诊断

你遇到的错误：
```json
{
  "error": "agent: llm call step 0: llm: api error [invalid_request_error]: 
  Incorrect API key provided."
}
```

这说明：
1. ✅ 请求已发送到阿里云
2. ❌ API Key 验证失败

---

## 🔍 查看完整请求信息

### 方法一：使用调试脚本（推荐）

```bash
cd /Users/luoxiaodou/workspace/projects/claw/GoPaw

# 1. 运行调试脚本（自动设置 debug=true）
./debug_llm.sh

# 2. 在新终端发送测试请求
curl -X POST http://localhost:8088/api/agent/chat \
  -H "Content-Type: application/json" \
  -d '{"session_id":"test","content":"你好"}'

# 3. 查看调试脚本输出的日志
```

---

### 方法二：手动设置 Debug 模式

#### 1. 修改配置文件

```bash
vim config.yaml
```

找到 `app.debug` 设置为 `true`：

```yaml
app:
  name: "GoPaw"
  language: zh
  debug: true  # ← 改为 true
```

#### 2. 重新启动服务

```bash
./gopaw start
```

#### 3. 发送测试请求

```bash
curl -X POST http://localhost:8088/api/agent/chat \
  -H "Content-Type: application/json" \
  -d '{"session_id":"test","content":"你好"}'
```

#### 4. 查看日志输出

服务会输出类似这样的日志：

```
2026-03-01T12:00:00.000Z    DEBUG    LLM Request Details
    method: POST
    url: https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions
    model: qwen-turbo
    api_key_prefix: sk-abc1234...
    payload: {
        "model": "qwen-turbo",
        "messages": [
            {"role": "system", "content": "你是一个智能助理..."},
            {"role": "user", "content": "你好"}
        ],
        "max_tokens": 4096,
        "stream": false
    }
    raw_body: {"model":"qwen-turbo","messages":[...]}

2026-03-01T12:00:00.100Z    DEBUG    HTTP Request Headers
    Content-Type: application/json
    Authorization: Bearer sk-abc1234...

2026-03-01T12:00:00.500Z    DEBUG    HTTP Response
    status_code: 401
    status: 401 Unauthorized

2026-03-01T12:00:00.500Z    DEBUG    Raw Response Body
    {"error":{"message":"Incorrect API key provided","type":"invalid_request_error"}}
```

---

## 📋 关键信息说明

### 1. 请求地址

查看 `url` 字段，应该是阿里云 DashScope 的地址：
```
https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions
```

如果不是，检查 `config.yaml` 中的 `llm.base_url`。

---

### 2. 请求参数

查看 `payload` 字段，包含：
- `model`: 使用的模型（如 `qwen-turbo`）
- `messages`: 对话历史
- `max_tokens`: 最大 token 数
- `stream`: 是否流式输出

---

### 3. 请求头

查看 `Authorization` 字段：
```
Authorization: Bearer sk-xxxxxxxxxxxxxxx
```

确认：
- ✅ 有 `Bearer ` 前缀
- ✅ API Key 完整

---

### 4. 响应内容

查看 `Raw Response Body`：
- `401 Unauthorized`: API Key 错误
- `400 Bad Request`: 请求参数错误
- `200 OK`: 成功

---

## 🔧 常见问题解决

### 问题 1: API Key 格式错误

**阿里云 DashScope API Key 格式**:
```
sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

**检查**:
```bash
# 查看环境变量
echo $OPENAI_API_KEY

# 或查看配置文件
cat config.yaml | grep api_key
```

**解决**:
```bash
# 重新设置（替换为你的真实 Key）
export OPENAI_API_KEY=sk-xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx

# 或在 config.yaml 中直接填写
vim config.yaml
```

---

### 问题 2: Base URL 错误

**阿里云 DashScope 兼容模式地址**:
```yaml
llm:
  base_url: https://dashscope.aliyuncs.com/compatible-mode/v1
```

**检查**:
```bash
cat config.yaml | grep base_url
```

**解决**:
```bash
vim config.yaml
# 修改为正确的 base_url
```

---

### 问题 3: 模型名称错误

**阿里云支持的模型**:
- `qwen-turbo`
- `qwen-plus`
- `qwen-max`

**检查**:
```bash
cat config.yaml | grep model
```

**解决**:
```bash
vim config.yaml
# 修改 model 为 qwen-turbo 或其他支持的模型
```

---

## 🎯 快速验证

### 1. 使用 curl 直接测试阿里云 API

```bash
# 替换为你的 API Key
API_KEY="sk-your-key-here"

curl -X POST https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $API_KEY" \
  -H "X-DashScope-SSE: disable" \
  -d '{
    "model": "qwen-turbo",
    "messages": [
      {"role": "user", "content": "你好"}
    ]
  }'
```

**预期响应**:
```json
{
  "choices": [
    {
      "message": {
        "content": "你好！有什么可以帮助你的吗？"
      }
    }
  ]
}
```

如果这个直接调用也失败，说明是 API Key 或配置问题。

---

### 2. 查看 GoPaw 配置

```bash
# 查看当前配置
./gopaw version
cat config.yaml
```

---

## 📝 调试日志示例

### 成功日志

```
DEBUG    LLM Request Details
    url: https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions
    model: qwen-turbo
    api_key_prefix: sk-abcd1234...
    payload: {"model":"qwen-turbo","messages":[...]}

DEBUG    HTTP Response
    status_code: 200

DEBUG    Raw Response Body
    {"choices":[{"message":{"content":"你好！..."}}]}
```

### 失败日志（API Key 错误）

```
DEBUG    LLM Request Details
    url: https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions
    model: qwen-turbo
    api_key_prefix: sk-wrong...

DEBUG    HTTP Response
    status_code: 401

DEBUG    Raw Response Body
    {"error":{"message":"Incorrect API key provided","type":"invalid_request_error"}}

ERROR    agent: llm call step 0: llm: api error [invalid_request_error]: Incorrect API key provided
```

---

## 🆘 获取帮助

如果调试后仍有问题，请提供：

1. **调试日志输出**（完整）
2. **config.yaml 内容**（隐藏真实 API Key）
3. **直接 curl 测试的结果**

这样可以快速定位问题！

---

*文档结束*
