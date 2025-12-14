# 数据库功能使用示例

## Docker 部署示例（推荐）

### 1. 使用 Docker Compose 部署（包含 MySQL）

创建 `docker-compose-with-db.yml` 文件：

```yaml
version: '3.8'

services:
  mysql:
    image: mysql:8.0
    container_name: pansou-mysql
    restart: always
    environment:
      MYSQL_ROOT_PASSWORD: your_root_password
      MYSQL_DATABASE: pansou
      MYSQL_USER: pansou
      MYSQL_PASSWORD: pansou_password
    volumes:
      - mysql_data:/var/lib/mysql
    ports:
      - "3306:3306"
    command: --character-set-server=utf8mb4 --collation-server=utf8mb4_unicode_ci
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 10s
      timeout: 5s
      retries: 5

  pansou:
    image: ghcr.io/fish2018/pansou:latest
    container_name: pansou-service
    restart: always
    ports:
      - "8888:8888"
    environment:
      # 数据库配置
      DB_ENABLED: "true"
      DB_HOST: mysql
      DB_PORT: "3306"
      DB_USER: pansou
      DB_PASSWORD: pansou_password
      DB_NAME: pansou
      # 应用配置
      PORT: "8888"
      CHANNELS: "tgsearchers4"
      ENABLED_PLUGINS: "labi,zhizhen,shandian"
      # 缓存配置
      CACHE_ENABLED: "true"
      CACHE_TTL: "60"
    depends_on:
      mysql:
        condition: service_healthy
    volumes:
      - cache_data:/app/cache

volumes:
  mysql_data:
  cache_data:
```

启动服务：

```bash
# 启动所有服务
docker-compose -f docker-compose-with-db.yml up -d

# 查看日志
docker-compose -f docker-compose-with-db.yml logs -f

# 停止服务
docker-compose -f docker-compose-with-db.yml down
```

### 2. 连接到外部 MySQL 数据库

如果已有 MySQL 数据库，可以直接连接：

```bash
docker run -d --name pansou \
  -p 8888:8888 \
  -e DB_ENABLED=true \
  -e DB_HOST=192.168.1.100 \
  -e DB_PORT=3306 \
  -e DB_USER=pansou \
  -e DB_PASSWORD=your_password \
  -e DB_NAME=pansou \
  -e CHANNELS=tgsearchers4 \
  -e ENABLED_PLUGINS=labi,zhizhen \
  ghcr.io/fish2018/pansou:latest
```

## 本地开发环境示例

### 1. 准备数据库

```sql
-- 创建数据库
CREATE DATABASE IF NOT EXISTS pansou DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- 创建用户（可选）
CREATE USER IF NOT EXISTS 'pansou'@'localhost' IDENTIFIED BY 'pansou_password';
GRANT ALL PRIVILEGES ON pansou.* TO 'pansou'@'localhost';
FLUSH PRIVILEGES;
```

### 2. 配置环境变量

创建 `.env` 文件：

```bash
# 数据库配置
DB_ENABLED=true
DB_HOST=localhost
DB_PORT=3306
DB_USER=pansou
DB_PASSWORD=pansou_password
DB_NAME=pansou

# 应用配置
PORT=8888
CHANNELS=tgsearchers4
ENABLED_PLUGINS=labi,zhizhen,shandian

# 缓存配置
CACHE_ENABLED=true
CACHE_TTL=60
```

### 3. 启动应用

```bash
# 安装依赖
go mod tidy

# 运行
go run main.go
```

## 使用场景示例

### 场景1：首次搜索（数据库为空）

```bash
# 第一次搜索某个关键词
curl "http://localhost:8888/api/search?kw=速度与激情"
```

**流程**：
1. 检查数据库，未找到相关记录
2. 使用插件进行实时搜索
3. 返回搜索结果给用户
4. 后台异步将结果保存到数据库

**日志输出**：
```
ℹ️ 数据库中没有找到结果，将使用插件搜索
✅ 成功保存 15 条结果到数据库
```

### 场景2：再次搜索（数据库有数据）

```bash
# 再次搜索相同或相关关键词
curl "http://localhost:8888/api/search?kw=速度与激情10"
```

**流程**：
1. 检查数据库，找到相关记录（关键词模糊匹配）
2. 直接从数据库返回结果（毫秒级响应）
3. 无需调用插件搜索

**日志输出**：
```
✅ 从数据库找到 15 条结果
```

### 场景3：强制刷新

```bash
# 强制使用插件搜索，更新数据库
curl "http://localhost:8888/api/search?kw=速度与激情&refresh=true"
```

**流程**：
1. 跳过数据库检查
2. 直接使用插件搜索
3. 返回最新结果
4. 更新数据库记录

### 场景4：关联搜索

由于使用模糊匹配，以下搜索都可能命中同一批数据库记录：

```bash
# 这些搜索可能返回相同或重叠的结果
curl "http://localhost:8888/api/search?kw=速度与激情"
curl "http://localhost:8888/api/search?kw=速度"
curl "http://localhost:8888/api/search?kw=激情"
```

## 数据库维护

### 查看数据库统计

当前版本已实现统计功能（在 DAO 层），可以通过扩展 API 暴露：

```go
// 可以添加一个管理接口来查看统计
stats, _ := searchDAO.GetStatistics()
// 返回：
// {
//   "search_records": 1234,
//   "link_records": 5678,
//   "latest_datetime": "2023-12-13T10:30:00Z"
// }
```

### 清理旧数据

当前版本已实现清理功能，可以通过定时任务或手动调用：

```go
// 清理90天前的数据
searchDAO.CleanOldRecords(90)
```

### 手动查询数据库

```sql
-- 查看搜索记录
SELECT * FROM search_records ORDER BY created_at DESC LIMIT 10;

-- 查看链接记录
SELECT * FROM link_records ORDER BY created_at DESC LIMIT 10;

-- 统计各网盘类型的数量
SELECT type, COUNT(*) as count FROM link_records GROUP BY type;

-- 查找包含某关键词的记录
SELECT * FROM search_records WHERE keyword LIKE '%速度与激情%';
```

## 性能对比

### 不启用数据库
- 首次搜索：2-5秒（取决于插件响应速度）
- 后续搜索：100-200ms（内存缓存）
- 缓存失效后：2-5秒（重新搜索）

### 启用数据库
- 首次搜索：2-5秒（插件搜索 + 数据库写入）
- 后续搜索：20-50ms（数据库查询）
- 关联搜索：20-50ms（模糊匹配命中）
- 数据持久化：无需担心缓存失效

## 注意事项

1. **首次部署**：数据库为空，所有搜索都会使用插件，逐渐积累数据
2. **磁盘空间**：根据搜索频率，数据库会逐渐增长，建议定期清理
3. **性能优化**：已在关键字段建立索引，支持快速查询
4. **数据备份**：建议定期备份数据库，避免数据丢失

## 故障排查

### 数据库连接失败

检查事项：
- MySQL 服务是否正常运行
- 数据库用户名密码是否正确
- 网络连接是否正常
- 防火墙设置是否允许连接

查看日志：
```bash
# Docker 环境
docker logs pansou-service

# 本地环境
# 查看控制台输出
```

### 数据未保存

可能原因：
- `DB_ENABLED` 未设置为 `true`
- 数据库用户权限不足
- 磁盘空间不足

### 搜索结果不准确

解决方案：
- 使用 `refresh=true` 参数强制更新
- 检查关键词是否正确
- 确认数据库记录是否存在
