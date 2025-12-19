package config

import (
	_ "database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"time"

	_ "github.com/go-sql-driver/mysql" // 驱动
	"gopkg.in/yaml.v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Config 应用配置结构
type Config struct {
	DefaultChannels    []string `yaml:"default_channels"`
	DefaultConcurrency int      `yaml:"default_concurrency"`
	Port               string   `yaml:"port"`
	ProxyURL           string   `yaml:"proxy_url"`
	UseProxy           bool     `yaml:"use_proxy"`
	HTTPProxyURL       string   `yaml:"http_proxy_url"`
	HTTPSProxyURL      string   `yaml:"https_proxy_url"`
	// 缓存相关配置
	CacheEnabled    bool   `yaml:"cache_enabled"`
	CachePath       string `yaml:"cache_path"`
	CacheMaxSizeMB  int    `yaml:"cache_max_size_mb"`
	CacheTTLMinutes int    `yaml:"cache_ttl_minutes"`
	// 压缩相关配置
	EnableCompression bool `yaml:"enable_compression"`
	MinSizeToCompress int  `yaml:"min_size_to_compress"` // 最小压缩大小（字节）
	// GC相关配置
	GCPercent      int  `yaml:"gc_percent"`      // GC触发阈值百分比
	OptimizeMemory bool `yaml:"optimize_memory"` // 是否启用内存优化
	// 插件相关配置
	PluginTimeoutSeconds int           `yaml:"plugin_timeout_seconds"` // 插件超时时间（秒）
	PluginTimeout        time.Duration `yaml:"plugin_timeout"`         // 插件超时时间（Duration）
	// 异步插件相关配置
	AsyncPluginEnabled        bool          `yaml:"async_plugin_enabled"`         // 是否启用异步插件
	EnabledPlugins            []string      `yaml:"enabled_plugins"`              // 启用的具体插件列表（空表示启用所有）
	AsyncResponseTimeout      int           `yaml:"async_response_timeout"`       // 响应超时时间（秒）
	AsyncResponseTimeoutDur   time.Duration `yaml:"async_response_timeout_dur"`   // 响应超时时间（Duration）
	AsyncMaxBackgroundWorkers int           `yaml:"async_max_background_workers"` // 最大后台工作者数量
	AsyncMaxBackgroundTasks   int           `yaml:"async_max_background_tasks"`   // 最大后台任务数量
	AsyncCacheTTLHours        int           `yaml:"async_cache_ttl_hours"`        // 异步缓存有效期（小时）
	AsyncLogEnabled           bool          `yaml:"async_log_enabled"`            // 是否启用异步插件详细日志
	// HTTP服务器配置
	HTTPReadTimeout  time.Duration `yaml:"http_read_timeout"`  // 读取超时
	HTTPWriteTimeout time.Duration `yaml:"http_write_timeout"` // 写入超时
	HTTPIdleTimeout  time.Duration `yaml:"http_idle_timeout"`  // 空闲超时
	HTTPMaxConns     int           `yaml:"http_max_conns"`     // 最大连接数
	// 认证相关配置
	AuthEnabled     bool              `yaml:"auth_enabled"`      // 是否启用认证
	AuthUsers       map[string]string `yaml:"auth_users"`        // 用户名:密码映射
	AuthTokenExpiry time.Duration     `yaml:"auth_token_expiry"` // Token有效期
	AuthJWTSecret   string            `yaml:"auth_jwt_secret"`   // JWT签名密钥
	DB              struct {
		Host      string `yaml:"host"`
		Port      int    `yaml:"port"`
		User      string `yaml:"user"`
		Password  string `yaml:"password"`
		Database  string `yaml:"database"`
		Charset   string `yaml:"charset"`
		ParseTime bool   `yaml:"parseTime"`
		Loc       string `yaml:"loc"`
	} `yaml:"db"`
}

// 全局配置实例
var (
	AppConfig *Config
	DB        *gorm.DB
)

func InitConfigFromFile(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	c := &Config{}
	if err := yaml.Unmarshal(raw, c); err != nil {
		return nil, err
	}

	// 计算字段
	c.UseProxy = c.ProxyURL != ""
	c.PluginTimeout = time.Duration(c.PluginTimeoutSeconds) * time.Second
	c.AsyncResponseTimeoutDur = time.Duration(c.AsyncResponseTimeout) * time.Second
	c.CachePath = getCachePath(c.CachePath)
	c.AsyncMaxBackgroundWorkers = getAsyncMaxBackgroundWorkers(c.AsyncMaxBackgroundTasks)
	c.AsyncMaxBackgroundTasks = getAsyncMaxBackgroundTasks(c.AsyncMaxBackgroundWorkers, c.AsyncMaxBackgroundTasks)
	c.HTTPReadTimeout = c.HTTPReadTimeout * time.Second
	c.HTTPWriteTimeout = c.HTTPWriteTimeout * time.Second
	c.HTTPIdleTimeout = c.HTTPIdleTimeout * time.Second
	c.AuthTokenExpiry = c.AuthTokenExpiry * time.Hour
	return c, nil
}

func InitDB(cfg *Config) error {
	// 1. 拼 DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		cfg.DB.User,
		cfg.DB.Password,
		cfg.DB.Host,
		cfg.DB.Port,
		cfg.DB.Database,
		cfg.DB.Charset,
		cfg.DB.ParseTime,
		cfg.DB.Loc,
	)

	// 2. 打开连接
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("gorm open: %w", err)
	}

	// 取出底层 *sql.DB 设置连接池
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB: %w", err)
	}

	// 连接池参数（可按需挪到 YAML）
	sqlDB.SetMaxOpenConns(100)                 // 最大连接数
	sqlDB.SetMaxIdleConns(20)                  // 最大空闲连接
	sqlDB.SetConnMaxLifetime(30 * time.Minute) // 连接最长生命周期
	sqlDB.SetConnMaxIdleTime(10 * time.Minute) // 空闲连接最大存活时间

	return sqlDB.Ping() // 验证连通性

	/*DB, err = sql.Open("mysql", dsn)
	if err != nil {
		return err
	}
	// 3. 验证连通性
	return DB.Ping()*/
}

// Init 初始化配置
func Init() {
	var err error
	AppConfig, err = InitConfigFromFile("config/config.yaml") // 或从 flag 读取路径
	if err != nil {
		log.Fatalf("load config.yaml: %v", err)
	}

	// 数据库链接
	if err := InitDB(AppConfig); err != nil {
		log.Fatalf("init db: %v", err)
	}

	// 应用GC配置
	applyGCSettings()

}

// 更新默认并发数（根据实际插件数或0调用）
// pluginCount: 如果插件被禁用则为0，否则为实际插件数
func UpdateDefaultConcurrency(pluginCount int) {
	if AppConfig == nil {
		return
	}

	// 只有当未通过环境变量指定并发数时才进行调整
	concurrencyEnv := os.Getenv("CONCURRENCY")
	if concurrencyEnv != "" {
		return
	}

	// 计算频道数
	channelCount := len(AppConfig.DefaultChannels)

	// 计算并发数 = 频道数 + 插件数（插件禁用时为0）+ 10
	concurrency := channelCount + pluginCount + 10
	if concurrency < 1 {
		concurrency = 1 // 确保至少为1
	}

	// 更新配置
	AppConfig.DefaultConcurrency = concurrency
}

// 从环境变量获取缓存路径，如果未设置则使用默认路径
func getCachePath(path string) string {
	if path == "" {
		// 默认在当前目录下创建cache文件夹
		defaultPath, err := filepath.Abs("./cache")
		if err != nil {
			return "./cache"
		}
		return defaultPath
	}
	return path
}

// 从环境变量获取最大后台工作者数量，如果未设置则自动计算
func getAsyncMaxBackgroundWorkers(sizeEnv int) int {
	if sizeEnv > 0 {
		return sizeEnv
	}

	// 自动计算：根据CPU核心数计算
	// 每个CPU核心分配5个工作者，最小20个
	cpuCount := runtime.NumCPU()
	workers := cpuCount * 5

	// 确保至少有20个工作者
	if workers < 20 {
		workers = 20
	}

	return workers
}

// 从环境变量获取最大后台任务数量，如果未设置则自动计算
func getAsyncMaxBackgroundTasks(task int, worker int) int {

	if task > 0 {
		return task
	}

	// 自动计算：工作者数量的5倍，最小100个
	workers := getAsyncMaxBackgroundWorkers(worker)
	tasks := workers * 5

	// 确保至少有100个任务
	if tasks < 100 {
		tasks = 100
	}

	return tasks
}

// 应用GC设置
func applyGCSettings() {
	// 设置GC百分比
	debug.SetGCPercent(AppConfig.GCPercent)

	// 如果启用内存优化
	if AppConfig.OptimizeMemory {
		// 释放操作系统内存
		debug.FreeOSMemory()
	}
}
