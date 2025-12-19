# Gying 搜索插件

## 📖 简介

Gying是PanSou的搜索插件，用于从 www.gying.net 网站搜索影视资源。支持多用户登录并配置账户，在搜索时自动聚合所有用户的搜索结果。

## ✨ 核心特性

- ✅ **多用户支持** - 每个用户独立配置，互不干扰
- ✅ **用户名密码登录** - 支持使用用户名和密码登录
- ✅ **智能去重** - 多用户搜索时自动去重
- ✅ **负载均衡** - 任务均匀分配，避免单用户限流
- ✅ **内存缓存** - 用户数据缓存到内存，搜索性能极高
- ✅ **持久化存储** - Cookie和用户配置自动保存，重启不丢失
- ✅ **Web管理界面** - 一站式配置，简单易用
- ✅ **RESTful API** - 支持程序化调用
- ✅ **默认账户自动登录** - 插件启动时自动使用默认账户登录

## 🚀 快速开始

### 步骤1: 启动服务

```bash
cd /Users/macbookpro/Desktop/fish2018/pansou
go run main.go

# 或者编译后运行
go build -o pansou main.go
./pansou
```

### 步骤2: 访问管理页面

如果需要添加更多账户或管理现有账户，可以访问管理页面：

```
http://localhost:8889/gying/你的用户名
```

**示例**：
```
http://localhost:8889/gying/myusername
```

系统会自动：
1. 根据用户名生成专属64位hash（不可逆）
2. 重定向到专属管理页面：`http://localhost:8889/gying/{hash}`
3. 显示登录表单供手动登录

**📌 提示**：请收藏hash后的URL（包含你的专属hash），方便下次访问。

### 步骤3: 手动登录

在"登录状态"区域输入：
- 用户名
- 密码

点击"**登录**"按钮。

### 步骤4: 开始搜索

在PanSou主页搜索框输入关键词，系统会**自动聚合所有用户**的Gying搜索结果！

```bash
# 通过API搜索
curl "http://localhost:8889/api/search?kw=遮天"

# 只搜索插件（包括gying）
curl "http://localhost:8889/api/search?kw=遮天&src=plugin"
```

## 📡 API文档

### 统一接口

所有操作通过统一的POST接口：

```
POST /gying/{hash}
Content-Type: application/json

{
  "action": "操作类型",
  ...其他参数
}
```

### API列表

| Action | 说明 | 需要登录 |
|--------|------|---------|
| `get_status` | 获取状态 | ❌ |
| `login` | 登录 | ❌ |
| `logout` | 退出登录 | ✅ |
| `test_search` | 测试搜索 | ✅ |

---

### 1️⃣ get_status - 获取用户状态

**请求**：
```bash
curl -X POST "http://localhost:8889/gying/{hash}" \
  -H "Content-Type: application/json" \
  -d '{"action": "get_status"}'
```

**成功响应（已登录）**：
```json
{
  "success": true,
  "message": "获取成功",
  "data": {
    "hash": "abc123...",
    "logged_in": true,
    "status": "active",
    "username_masked": "pa****ou",
    "login_time": "2025-10-28 12:00:00",
    "expire_time": "2026-02-26 12:00:00",
    "expires_in_days": 121
  }
}
```

**成功响应（未登录）**：
```json
{
  "success": true,
  "message": "获取成功",
  "data": {
    "hash": "abc123...",
    "logged_in": false,
    "status": "pending"
  }
}
```

---

### 2️⃣ login - 登录

**请求**：
```bash
curl -X POST "http://localhost:8889/gying/{hash}" \
  -H "Content-Type: application/json" \
  -d '{"action": "login", "username": "xxx", "password": "xxx"}'
```

**成功响应**：
```json
{
  "success": true,
  "message": "登录成功",
  "data": {
    "status": "active",
    "username_masked": "pa****ou"
  }
}
```

**失败响应**：
```json
{
  "success": false,
  "message": "登录失败: 用户名或密码错误"
}
```

---

### 3️⃣ logout - 退出登录

**请求**：
```bash
curl -X POST "http://localhost:8889/gying/{hash}" \
  -H "Content-Type: application/json" \
  -d '{"action": "logout"}'
```

**成功响应**：
```json
{
  "success": true,
  "message": "已退出登录",
  "data": {
    "status": "pending"
  }
}
```

---

### 4️⃣ test_search - 测试搜索

**请求**：
```bash
curl -X POST "http://localhost:8889/gying/{hash}" \
  -H "Content-Type: application/json" \
  -d '{"action": "test_search", "keyword": "遮天"}'
```

**成功响应**：
```json
{
  "success": true,
  "message": "找到 5 条结果",
  "data": {
    "keyword": "遮天",
    "total_results": 5,
    "results": [
      {
        "title": "遮天：禁区",
        "links": [
          {
            "type": "quark",
            "url": "https://pan.quark.cn/s/89f7aeef9681",
            "password": ""
          }
        ]
      }
    ]
  }
}
```

---

## 🔧 配置说明

### 环境变量（可选）

```bash
# Hash Salt（推荐自定义，增强安全性）
export GYING_HASH_SALT="your-custom-salt-here"

# Cookie加密密钥（32字节，推荐自定义）
export GYING_ENCRYPTION_KEY="your-32-byte-key-here!!!!!!!!!!"
```

### 代码内配置

在 `gying.go` 第20-24行修改：

```go
const (
    MaxConcurrentUsers   = 10    // 最多使用的用户数（搜索时）
    MaxConcurrentDetails = 50    // 最大并发详情请求数
    DebugLog             = false // 调试日志开关
)
```

### 默认账户配置

在 `gying.go` 第27-32行修改默认账户：

```go
var DefaultAccounts = []struct {
    Username string
    Password string
}{
    // 可以添加更多默认账户
    // {"user2", "password2"},
}
```

**参数说明**：

| 参数 | 默认值 | 说明 | 建议 |
|------|--------|------|------|
| `MaxConcurrentUsers` | 10 | 单次搜索最多使用的用户数 | 10-20足够 |
| `MaxConcurrentDetails` | 50 | 最大并发详情请求数 | 50-100 |
| `DebugLog` | false | 是否开启调试日志 | 生产环境false |

## 📂 数据存储

### 存储位置

```
cache/gying_users/{hash}.json
```

### 数据结构

```json
{
  "hash": "abc123...",
  "username": "pansou",
  "username_masked": "pa****ou",
  "cookie": "BT_auth=xxx; BT_cookietime=xxx",
  "status": "active",
  "created_at": "2025-10-28T12:00:00+08:00",
  "login_at": "2025-10-28T12:00:00+08:00",
  "expire_at": "2026-02-26T12:00:00+08:00",
  "last_access_at": "2025-10-28T13:00:00+08:00"
}
```

**字段说明**：
- `hash`: 用户唯一标识（SHA256，不可逆推用户名）
- `username`: 原始用户名（存储）
- `username_masked`: 脱敏用户名（如`pa****ou`）
- `cookie`: 登录Cookie（明文存储，建议配置加密）
- `status`: 用户状态（`pending`/`active`/`expired`）
- `expire_at`: Cookie过期时间（121天）