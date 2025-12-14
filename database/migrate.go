package database

import (
	"fmt"
)

// MigrateFulltextIndex 迁移全文索引（用于已有数据库）
func MigrateFulltextIndex() error {
	if !IsEnabled() {
		return fmt.Errorf("数据库未启用")
	}

	// 检查是否已存在全文索引
	var indexCount int64
	DB.Raw("SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = DATABASE() AND table_name = 'search_records' AND index_name = 'idx_fulltext_title'").Scan(&indexCount)

	if indexCount > 0 {
		fmt.Println("✅ 全文索引已存在，跳过创建")
		return nil
	}

	fmt.Println("正在创建全文索引...")

	// 创建标题全文索引
	if err := DB.Exec("ALTER TABLE search_records ADD FULLTEXT INDEX idx_fulltext_title (title) WITH PARSER ngram").Error; err != nil {
		return fmt.Errorf("创建标题全文索引失败: %v", err)
	}
	fmt.Println("✅ 标题全文索引创建成功")

	// 创建内容全文索引
	if err := DB.Exec("ALTER TABLE search_records ADD FULLTEXT INDEX idx_fulltext_content (content) WITH PARSER ngram").Error; err != nil {
		return fmt.Errorf("创建内容全文索引失败: %v", err)
	}
	fmt.Println("✅ 内容全文索引创建成功")

	return nil
}

// OptimizeDatabase 优化数据库（清理碎片、更新统计信息）
func OptimizeDatabase() error {
	if !IsEnabled() {
		return fmt.Errorf("数据库未启用")
	}

	fmt.Println("正在优化数据库...")

	// 优化表
	tables := []string{"search_records", "link_records"}
	for _, table := range tables {
		if err := DB.Exec(fmt.Sprintf("OPTIMIZE TABLE %s", table)).Error; err != nil {
			fmt.Printf("⚠️ 优化表 %s 失败: %v\n", table, err)
		} else {
			fmt.Printf("✅ 表 %s 优化成功\n", table)
		}
	}

	// 分析表（更新统计信息）
	for _, table := range tables {
		if err := DB.Exec(fmt.Sprintf("ANALYZE TABLE %s", table)).Error; err != nil {
			fmt.Printf("⚠️ 分析表 %s 失败: %v\n", table, err)
		} else {
			fmt.Printf("✅ 表 %s 分析成功\n", table)
		}
	}

	return nil
}
