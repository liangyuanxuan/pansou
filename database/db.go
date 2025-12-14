package database

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"pansou/config"
	"pansou/model"
)

var DB *gorm.DB

// Init 初始化数据库连接
func Init() error {
	// 检查数据库是否启用
	if !config.AppConfig.DBEnabled {
		log.Println("数据库功能未启用")
		return nil
	}

	// 构建DSN连接字符串
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		config.AppConfig.DBUser,
		config.AppConfig.DBPassword,
		config.AppConfig.DBHost,
		config.AppConfig.DBPort,
		config.AppConfig.DBName,
	)

	// 配置GORM
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // 生产环境使用Silent，开发环境可使用Info
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	}

	// 连接数据库
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("连接数据库失败: %v", err)
	}

	// 获取底层的SQL DB对象
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("获取数据库实例失败: %v", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(config.AppConfig.DBMaxIdle)
	sqlDB.SetMaxOpenConns(config.AppConfig.DBMaxOpen)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 自动迁移数据库表
	err = DB.AutoMigrate(&model.SearchRecord{}, &model.LinkRecord{})
	if err != nil {
		return fmt.Errorf("数据库迁移失败: %v", err)
	}

	// 迁移全文索引（如果不存在）
	if err := MigrateFulltextIndex(); err != nil {
		log.Printf("⚠️ 全文索引迁移失败: %v (将继续使用，但全文搜索可能不可用)", err)
	}

	log.Printf("数据库连接成功: %s:%s/%s", config.AppConfig.DBHost, config.AppConfig.DBPort, config.AppConfig.DBName)
	return nil
}

// Close 关闭数据库连接
func Close() error {
	if DB == nil {
		return nil
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

// IsEnabled 检查数据库是否启用
func IsEnabled() bool {
	return config.AppConfig.DBEnabled && DB != nil
}
