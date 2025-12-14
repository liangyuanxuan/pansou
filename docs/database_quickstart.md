# MySQL 数据库功能快速开始

## 5分钟快速启动

### 方式一：Docker Compose 一键启动（最简单）

1. **创建配置文件**

创建 `docker-compose-db.yml`：

```yaml
version: '3.8'

services:
  mysql:
    image: mysql:8.0
    container_name: pansou-mysql
    restart: always
    environment:
      MYSQL_ROOT_PASSWORD: pansou123456
      MYSQL_DATABASE: pansou
    volumes:
      - mysql_data:/var/lib/mysql
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
      DB_ENABLED: "true"
      DB_HOST: mysql
      DB_USER: root
      DB_PASSWORD: pansou123456
      DB_NAME: pansou
      ENABLED_PLUGINS: "labi,zhizhen"
    depends_on:
      mysql:
        condition: service_healthy

volumes:
  mysql_data:
```

2. **启动服务**

```bash
docker-compose -f docker-compose-db.yml up -d
```

3. **测试搜索**

```bash
# 首次搜索（会使用插件并保存到数据库）
curl "http://localhost:8888/api/search?kw=电影"

# 再次搜索（直接从数据库返回，超快！）
curl "http://localhost:8888/api/search?kw=电影"
```

### 方式二：使用现有 MySQL

如果你已经有 MySQL 服务，只需启动 pansou 容器：

```bash
docker run -d --name pansou \
  -p 8888:8888 \
  -e DB_ENABLED=true \
  -e DB_HOST=你的MySQL地址 \
  -e DB_USER=root \
  -e DB_PASSWORD=你的密码 \
  -e DB_NAME=pansou \
  -e ENABLED_PLUGINS=labi,zhizhen \
  ghcr.io/fish2018/pansou:latest
```

### 方式三：本地开发环境

1. **准备 MySQL**

```sql
CREATE DATABASE pansou CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

2. **配置环境变量**

```bash
export DB_ENABLED=true
export DB_HOST=localhost
export DB_PASSWORD=你的密码
export ENABLED_PLUGINS=labi,zhizhen
```

3. **启动服务**

```bash
./pansou
```

## 验证数据库功能

### 1. 首次搜索

```bash
curl "http://localhost:8888/api/search?kw=速度与激情"
```

观察日志：
```
ℹ️ 数据库中没有找到结果，将使用插件搜索
✅ 成功保存 15 条结果到数据库
```

### 2. 再次搜索（速度对比）

```bash
# 相同关键词 - 超快返回
curl "http://localhost:8888/api/search?kw=速度与激情"
```

观察日志：
```
✅ 从数据库找到 15 条结果
```

响应时间从 2-5秒 降低到 20-50毫秒！

### 3. 模糊匹配测试

```bash
# 这些搜索都可能命中数据库
curl "http://localhost:8888/api/search?kw=速度"
curl "http://localhost:8888/api/search?kw=激情"
```

### 4. 强制刷新

```bash
# 跳过数据库，强制使用插件搜索
curl "http://localhost:8888/api/search?kw=速度与激情&refresh=true"
```

## 查看数据库

```bash
# 进入 MySQL 容器
docker exec -it pansou-mysql mysql -uroot -ppansou123456 pansou

# 或使用任何 MySQL 客户端连接
# 主机：localhost
# 端口：3306
# 用户：root
# 密码：pansou123456
# 数据库：pansou
```

查询示例：

```sql
-- 查看所有搜索记录
SELECT id, keyword, title, datetime FROM search_records ORDER BY created_at DESC LIMIT 10;

-- 查看网盘链接
SELECT type, url, note, datetime FROM link_records 
JOIN search_records ON link_records.search_id = search_records.id 
LIMIT 10;

-- 统计网盘类型
SELECT type, COUNT(*) as count FROM link_records GROUP BY type;
```

## 常见问题

### Q1: 如何禁用数据库？

设置 `DB_ENABLED=false` 或不设置该变量。

### Q2: 数据库会自动清理吗？

目前不会自动清理，可以手动执行清理或等待后续版本的自动清理功能。

### Q3: 数据库对性能有影响吗？

**没有负面影响，反而提升性能：**
- 首次搜索：与不使用数据库相同（2-5秒）
- 后续搜索：大幅提升（20-50ms vs 100-200ms）
- 数据持久化：不依赖内存缓存

### Q4: 可以使用其他数据库吗？

目前只支持 MySQL。使用 GORM 框架，未来可以轻松扩展支持 PostgreSQL、SQLite 等。

### Q5: 数据库需要多大空间？

根据搜索频率不同：
- 低频使用（<100次/天）：~100MB/月
- 中频使用（100-1000次/天）：~500MB/月
- 高频使用（>1000次/天）：~2GB/月

## 性能对比

| 场景 | 不使用数据库 | 使用数据库 | 提升 |
|------|------------|----------|------|
| 首次搜索 | 2-5秒 | 2-5秒 | - |
| 命中缓存 | 100-200ms | 20-50ms | **4-10倍** |
| 缓存失效 | 2-5秒 | 20-50ms | **40-250倍** |
| 关联搜索 | 2-5秒 | 20-50ms | **40-250倍** |

## 下一步

- 查看[完整数据库配置文档](database.md)
- 查看[使用示例和场景](database_example.md)
- 探索更多高级功能和调优选项

## 技术支持

遇到问题？
1. 查看服务日志：`docker logs pansou-service`
2. 查看 MySQL 日志：`docker logs pansou-mysql`
3. 提交 Issue：[GitHub Issues](https://github.com/fish2018/pansou/issues)
