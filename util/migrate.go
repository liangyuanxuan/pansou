package util

import (
	"fmt"
	"log"

	"pansou/config"
	"pansou/model"
)

// MigrateDatabase 执行数据库迁移
func MigrateDatabase() error {
	db := config.DB
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	log.Println("开始执行数据库迁移...")

	// 自动迁移模型
	err := db.AutoMigrate(
		&model.SearchData{},
		// 未来可以在这里添加更多模型
	)

	if err != nil {
		return fmt.Errorf("数据库迁移失败: %v", err)
	}

	log.Println("✅ 数据库迁移完成")
	return nil
}

// CreateIndexes 创建索引以提升查询性能
func CreateIndexes() error {
	db := config.DB
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	log.Println("开始创建数据库索引...")

	// 创建复合索引（keyword + source）
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_keyword_source 
		ON search_data(keyword, source)
	`).Error; err != nil {
		log.Printf("⚠️ 创建复合索引失败: %v", err)
	}

	// 创建全文索引（如果MySQL版本支持）
	if err := db.Exec(`
		CREATE FULLTEXT INDEX IF NOT EXISTS idx_title_fulltext 
		ON search_data(title)
	`).Error; err != nil {
		log.Printf("⚠️ 创建全文索引失败（可能MySQL版本不支持）: %v", err)
	}

	log.Println("✅ 索引创建完成")
	return nil
}
