# 数据库配置说明

## 功能说明

本项目现已支持 MySQL 数据库集成，实现以下功能：

1. **智能搜索**：优先从数据库搜索，数据库没有数据时才使用插件搜索
2. **自动缓存**：插件搜索的结果会自动保存到数据库
3. **关联查询**：支持关键词模糊匹配，可以查找相关的历史搜索结果

## 数据库配置

### 环境变量配置

在项目根目录创建 `.env` 文件或直接设置环境变量：

```bash
# 启用数据库功能（必需）
DB_ENABLED=true

# 数据库连接配置
DB_HOST=localhost        # 数据库主机地址，默认：localhost
DB_PORT=3306            # 数据库端口，默认：3306
DB_USER=root            # 数据库用户名，默认：root
DB_PASSWORD=your_password  # 数据库密码（必需）
DB_NAME=pansou          # 数据库名称，默认：pansou

# 连接池配置（可选）
DB_MAX_IDLE=10          # 最大空闲连接数，默认：10
DB_MAX_OPEN=100         # 最大打开连接数，默认：100
```

### 数据库初始化

1. **创建数据库**

```sql
CREATE DATABASE IF NOT EXISTS pansou DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

2. **自动创建表结构**

程序启动时会自动创建必要的表结构，无需手动执行 SQL。包括：
- `search_records` - 搜索记录表
- `link_records` - 网盘链接表

## 数据库表结构

### search_records 表

存储搜索结果的主记录：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INT | 主键 |
| keyword | VARCHAR(255) | 搜索关键词（支持多个，逗号分隔） |
| message_id | VARCHAR(100) | 消息ID |
| unique_id | VARCHAR(255) | 全局唯一ID（唯一索引） |
| channel | VARCHAR(100) | 频道名称 |
| datetime | DATETIME | 数据时间 |
| title | TEXT | 标题 |
| content | LONGTEXT | 内容 |
| tags | TEXT | 标签（JSON格式） |
| images | TEXT | 图片链接（JSON格式） |
| created_at | DATETIME | 创建时间 |
| updated_at | DATETIME | 更新时间 |

### link_records 表

存储网盘链接记录：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | INT | 主键 |
| search_id | INT | 关联 search_records 的 ID |
| type | VARCHAR(50) | 网盘类型 |
| url | VARCHAR(500) | 网盘链接（唯一索引） |
| password | VARCHAR(100) | 提取码 |
| datetime | DATETIME | 链接时间 |
| work_title | TEXT | 作品标题 |
| created_at | DATETIME | 创建时间 |
| updated_at | DATETIME | 更新时间 |

## 使用说明

### 搜索流程

1. **首次搜索**：数据库为空，使用插件搜索并自动保存结果
2. **后续搜索**：优先从数据库查询，找到结果直接返回
3. **强制刷新**：使用 `refresh=true` 参数强制使用插件搜索并更新数据库

### API 使用

搜索 API 不需要任何改动，数据库功能会自动生效：

```bash
# 普通搜索（优先使用数据库）
GET /search?kw=电影名称

# 强制刷新（跳过数据库，使用插件搜索）
GET /search?kw=电影名称&refresh=true
```

### 数据管理

程序会自动维护数据库，包括：
- 自动去重：相同的 unique_id 不会重复存储
- 关键词关联：同一条记录可以关联多个搜索关键词
- 链接去重：相同的 URL 不会重复存储

## 注意事项

1. **数据库字符集**：必须使用 `utf8mb4`，以支持表情符号等特殊字符
2. **磁盘空间**：根据搜索频率，数据库会逐渐增长，建议定期清理旧数据
3. **性能优化**：已在关键字段建立索引，支持高效查询
4. **数据安全**：建议定期备份数据库

## 故障排查

### 数据库连接失败

检查：
1. 数据库服务是否启动
2. 用户名密码是否正确
3. 数据库是否已创建
4. 防火墙是否允许连接

### 数据未保存

检查：
1. `DB_ENABLED` 是否设置为 `true`
2. 查看控制台是否有错误日志
3. 检查数据库用户是否有写入权限

## 性能建议

1. **索引优化**：程序已自动在关键字段创建索引
2. **连接池**：合理设置 `DB_MAX_IDLE` 和 `DB_MAX_OPEN`
3. **定期清理**：可以定期清理超过一定时间的旧记录
4. **数据库调优**：根据实际情况调整 MySQL 配置参数

## 未来扩展

可以考虑添加的功能：
- 定时清理旧数据的 API
- 数据统计和分析接口
- 导出/导入数据功能
- 全文搜索优化
