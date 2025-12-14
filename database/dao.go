package database

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"pansou/model"
)

// SearchDAO 搜索数据访问对象
type SearchDAO struct{}

// NewSearchDAO 创建SearchDAO实例
func NewSearchDAO() *SearchDAO {
	return &SearchDAO{}
}

// SearchByKeyword 根据关键词搜索（优化版，支持多种搜索策略）
func (dao *SearchDAO) SearchByKeyword(keyword string) ([]model.SearchResult, error) {
	if !IsEnabled() {
		return nil, fmt.Errorf("数据库未启用")
	}

	var records []model.SearchRecord

	// 策略1: 精确关键词匹配（最快）
	err := DB.Where("keyword = ?", keyword).
		Order("datetime DESC").
		Limit(100).
		Find(&records).Error

	if err != nil {
		return nil, err
	}

	// 如果精确匹配没有结果，使用全文搜索（MySQL 5.7+ 支持中文分词）
	if len(records) == 0 {
		// 策略2: 全文搜索（标题和内容）
		err = DB.Where("MATCH(title) AGAINST(? IN NATURAL LANGUAGE MODE)", keyword).
			Or("MATCH(content) AGAINST(? IN NATURAL LANGUAGE MODE)", keyword).
			Order("datetime DESC").
			Limit(100).
			Find(&records).Error

		if err != nil {
			// 如果全文搜索失败（可能是MySQL版本不支持），回退到模糊搜索
			records = []model.SearchRecord{}
		}
	}

	// 策略3: 如果全文搜索也没有结果，使用模糊搜索
	if len(records) == 0 {
		err = DB.Where("keyword LIKE ?", "%"+keyword+"%").
			Or("title LIKE ?", "%"+keyword+"%").
			Or("content LIKE ?", "%"+keyword+"%").
			Order("datetime DESC").
			Limit(100).
			Find(&records).Error

		if err != nil {
			return nil, err
		}
	}

	// 如果没有找到记录，返回空切片
	if len(records) == 0 {
		return []model.SearchResult{}, nil
	}

	// 使用 Preload 优化关联查询，避免 N+1 问题
	recordIDs := make([]uint, len(records))
	for i, record := range records {
		recordIDs[i] = record.ID
	}

	// 一次性获取所有关联的链接
	var allLinkRecords []model.LinkRecord
	DB.Where("search_id IN ?", recordIDs).Find(&allLinkRecords)

	// 将链接按 search_id 分组
	linkMap := make(map[uint][]model.LinkRecord)
	for _, lr := range allLinkRecords {
		linkMap[lr.SearchID] = append(linkMap[lr.SearchID], lr)
	}

	// 转换为SearchResult格式
	results := make([]model.SearchResult, 0, len(records))
	for _, record := range records {
		// 获取该记录的所有链接
		linkRecords := linkMap[record.ID]

		// 转换链接
		links := make([]model.Link, 0, len(linkRecords))
		for _, lr := range linkRecords {
			links = append(links, model.Link{
				Type:      lr.Type,
				URL:       lr.URL,
				Password:  lr.Password,
				Datetime:  lr.Datetime,
				WorkTitle: lr.WorkTitle,
			})
		}

		// 解析Tags和Images
		var tags []string
		var images []string
		if record.Tags != "" {
			json.Unmarshal([]byte(record.Tags), &tags)
		}
		if record.Images != "" {
			json.Unmarshal([]byte(record.Images), &images)
		}

		results = append(results, model.SearchResult{
			MessageID: record.MessageID,
			UniqueID:  record.UniqueID,
			Channel:   record.Channel,
			Datetime:  record.Datetime,
			Title:     record.Title,
			Content:   record.Content,
			Links:     links,
			Tags:      tags,
			Images:    images,
		})
	}

	return results, nil
}

// SaveSearchResults 保存搜索结果到数据库
func (dao *SearchDAO) SaveSearchResults(keyword string, results []model.SearchResult) error {
	if !IsEnabled() {
		return fmt.Errorf("数据库未启用")
	}

	// 批量保存，使用事务确保数据一致性
	tx := DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	for _, result := range results {
		// 检查UniqueID是否已存在
		var existingRecord model.SearchRecord
		err := tx.Where("unique_id = ?", result.UniqueID).First(&existingRecord).Error

		if err == nil {
			// 记录已存在，更新关键词关联
			// 检查当前关键词是否已关联
			if !strings.Contains(existingRecord.Keyword, keyword) {
				existingRecord.Keyword = existingRecord.Keyword + "," + keyword
				tx.Save(&existingRecord)
			}
			continue
		}

		// 序列化Tags和Images
		tagsJSON, _ := json.Marshal(result.Tags)
		imagesJSON, _ := json.Marshal(result.Images)

		// 创建新记录
		record := model.SearchRecord{
			Keyword:   keyword,
			MessageID: result.MessageID,
			UniqueID:  result.UniqueID,
			Channel:   result.Channel,
			Datetime:  result.Datetime,
			Title:     result.Title,
			Content:   result.Content,
			Tags:      string(tagsJSON),
			Images:    string(imagesJSON),
		}

		if err := tx.Create(&record).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("保存搜索记录失败: %v", err)
		}

		// 保存关联的链接
		for _, link := range result.Links {
			// 检查链接是否已存在
			var existingLink model.LinkRecord
			linkErr := tx.Where("url = ?", link.URL).First(&existingLink).Error

			if linkErr == nil {
				// 链接已存在，更新search_id关联（如果需要）
				continue
			}

			linkRecord := model.LinkRecord{
				SearchID:  record.ID,
				Type:      link.Type,
				URL:       link.URL,
				Password:  link.Password,
				Datetime:  link.Datetime,
				WorkTitle: link.WorkTitle,
			}

			if err := tx.Create(&linkRecord).Error; err != nil {
				tx.Rollback()
				return fmt.Errorf("保存链接记录失败: %v", err)
			}
		}
	}

	return tx.Commit().Error
}

// CleanOldRecords 清理旧记录（超过指定天数）
func (dao *SearchDAO) CleanOldRecords(days int) error {
	if !IsEnabled() {
		return fmt.Errorf("数据库未启用")
	}

	cutoffTime := time.Now().AddDate(0, 0, -days)

	// 删除旧的搜索记录
	result := DB.Where("created_at < ?", cutoffTime).Delete(&model.SearchRecord{})
	if result.Error != nil {
		return result.Error
	}

	// 删除孤立的链接记录（没有关联的搜索记录）
	DB.Exec("DELETE FROM link_records WHERE search_id NOT IN (SELECT id FROM search_records)")

	return nil
}

// SearchOptions 高级搜索选项
type SearchOptions struct {
	Keyword     string    // 关键词
	Channel     string    // 指定频道
	StartTime   time.Time // 开始时间
	EndTime     time.Time // 结束时间
	Limit       int       // 返回结果数量限制
	UseFulltext bool      // 是否使用全文搜索
}

// AdvancedSearch 高级搜索（支持多种过滤条件）
func (dao *SearchDAO) AdvancedSearch(opts SearchOptions) ([]model.SearchResult, error) {
	if !IsEnabled() {
		return nil, fmt.Errorf("数据库未启用")
	}

	if opts.Limit <= 0 {
		opts.Limit = 100
	}

	query := DB.Model(&model.SearchRecord{})

	// 应用关键词搜索
	if opts.Keyword != "" {
		if opts.UseFulltext {
			// 使用全文搜索
			query = query.Where(
				DB.Where("MATCH(title) AGAINST(? IN NATURAL LANGUAGE MODE)", opts.Keyword).
					Or("MATCH(content) AGAINST(? IN NATURAL LANGUAGE MODE)", opts.Keyword),
			)
		} else {
			// 使用模糊搜索
			keyword := "%" + opts.Keyword + "%"
			query = query.Where(
				DB.Where("keyword LIKE ?", keyword).
					Or("title LIKE ?", keyword).
					Or("content LIKE ?", keyword),
			)
		}
	}

	// 应用频道过滤
	if opts.Channel != "" {
		query = query.Where("channel = ?", opts.Channel)
	}

	// 应用时间范围过滤
	if !opts.StartTime.IsZero() {
		query = query.Where("datetime >= ?", opts.StartTime)
	}
	if !opts.EndTime.IsZero() {
		query = query.Where("datetime <= ?", opts.EndTime)
	}

	// 执行查询
	var records []model.SearchRecord
	err := query.Order("datetime DESC").
		Limit(opts.Limit).
		Find(&records).Error

	if err != nil {
		return nil, err
	}

	if len(records) == 0 {
		return []model.SearchResult{}, nil
	}

	// 优化关联查询
	recordIDs := make([]uint, len(records))
	for i, record := range records {
		recordIDs[i] = record.ID
	}

	var allLinkRecords []model.LinkRecord
	DB.Where("search_id IN ?", recordIDs).Find(&allLinkRecords)

	linkMap := make(map[uint][]model.LinkRecord)
	for _, lr := range allLinkRecords {
		linkMap[lr.SearchID] = append(linkMap[lr.SearchID], lr)
	}

	// 转换结果
	results := make([]model.SearchResult, 0, len(records))
	for _, record := range records {
		linkRecords := linkMap[record.ID]
		links := make([]model.Link, 0, len(linkRecords))
		for _, lr := range linkRecords {
			links = append(links, model.Link{
				Type:      lr.Type,
				URL:       lr.URL,
				Password:  lr.Password,
				Datetime:  lr.Datetime,
				WorkTitle: lr.WorkTitle,
			})
		}

		var tags []string
		var images []string
		if record.Tags != "" {
			json.Unmarshal([]byte(record.Tags), &tags)
		}
		if record.Images != "" {
			json.Unmarshal([]byte(record.Images), &images)
		}

		results = append(results, model.SearchResult{
			MessageID: record.MessageID,
			UniqueID:  record.UniqueID,
			Channel:   record.Channel,
			Datetime:  record.Datetime,
			Title:     record.Title,
			Content:   record.Content,
			Links:     links,
			Tags:      tags,
			Images:    images,
		})
	}

	return results, nil
}

// GetStatistics 获取数据库统计信息
func (dao *SearchDAO) GetStatistics() (map[string]interface{}, error) {
	if !IsEnabled() {
		return nil, fmt.Errorf("数据库未启用")
	}

	stats := make(map[string]interface{})

	// 统计搜索记录数
	var searchCount int64
	DB.Model(&model.SearchRecord{}).Count(&searchCount)
	stats["search_records"] = searchCount

	// 统计链接记录数
	var linkCount int64
	DB.Model(&model.LinkRecord{}).Count(&linkCount)
	stats["link_records"] = linkCount

	// 统计最新记录时间
	var latestRecord model.SearchRecord
	DB.Order("datetime DESC").First(&latestRecord)
	stats["latest_datetime"] = latestRecord.Datetime

	// 统计各频道的记录数
	var channelStats []struct {
		Channel string
		Count   int64
	}
	DB.Model(&model.SearchRecord{}).
		Select("channel, COUNT(*) as count").
		Group("channel").
		Order("count DESC").
		Limit(10).
		Scan(&channelStats)
	stats["top_channels"] = channelStats

	return stats, nil
}
