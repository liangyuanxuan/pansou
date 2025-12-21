package service

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"pansou/config"
	"pansou/model"
	"pansou/util/elasticsearch"
)

// AdminService 管理端服务
type AdminService struct {
	esCacheService *elasticsearch.CacheService
}

// NewAdminService 创建管理端服务实例
func NewAdminService() *AdminService {
	var esCacheService *elasticsearch.CacheService

	// 尝试获取ES缓存服务
	if elasticsearch.IsEnabled() {
		esCacheService = elasticsearch.NewCacheService()
	}

	return &AdminService{
		esCacheService: esCacheService,
	}
}

// AddSearchData 添加搜索数据到MySQL和ES
func (s *AdminService) AddSearchData(req *model.AddDataRequest) (*model.AddDataResponse, error) {
	response := &model.AddDataResponse{
		Success:    false,
		AddedCount: 0,
		ESIndexed:  false,
	}

	if len(req.Results) == 0 {
		return response, fmt.Errorf("搜索结果列表为空")
	}

	db := config.DB
	if db == nil {
		return response, fmt.Errorf("数据库未初始化")
	}

	// 批量插入到MySQL
	var searchDataList []model.SearchData
	var failedCount int
	var errorMessages string

	for _, result := range req.Results {
		// 转换标签为JSON字符串
		tagsJSON, _ := json.Marshal(result.Tags)

		// 提取链接信息（使用第一个链接作为URL）
		var url string
		var linkType string
		if len(result.Links) > 0 {
			url = result.Links[0].URL
			linkType = result.Links[0].Type
		}

		// 格式化时间
		timeStr := result.Datetime.Format("2006-01-02 15:04:05")

		searchData := model.SearchData{
			Keyword:   req.Keyword,
			Source:    req.Source,
			Title:     result.Title,
			URL:       url,
			Size:      "", // SearchResult中没有Size字段
			Time:      timeStr,
			UpdatedAt: timeStr,
			Type:      linkType,
			Hot:       0, // SearchResult中没有Hot字段
			Tags:      string(tagsJSON),
		}
		searchDataList = append(searchDataList, searchData)
	}

	// 使用事务批量插入
	tx := db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 批量创建记录
	if err := tx.Create(&searchDataList).Error; err != nil {
		tx.Rollback()
		return response, fmt.Errorf("批量插入MySQL失败: %v", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return response, fmt.Errorf("提交事务失败: %v", err)
	}

	response.AddedCount = len(searchDataList) - failedCount
	response.FailedCount = failedCount
	response.FailedErrors = errorMessages

	// 同步到ES（如果ES已启用）
	if s.esCacheService != nil && elasticsearch.IsEnabled() {
		// 生成缓存键
		cacheKey := elasticsearch.GenerateCacheKey(req.Keyword, []string{}, []string{req.Source}, req.Source)

		// 将数据写入ES
		err := s.esCacheService.Set(cacheKey, req.Keyword, []string{}, []string{req.Source}, req.Source, req.Results)
		if err != nil {
			log.Printf("⚠️ 写入ES失败: %v", err)
			response.ESIndexed = false
		} else {
			response.ESIndexed = true
		}
	}

	response.Success = true
	response.Message = fmt.Sprintf("成功添加 %d 条数据到MySQL", response.AddedCount)
	if response.ESIndexed {
		response.Message += "，并已同步到Elasticsearch"
	}

	return response, nil
}

// BatchAddSearchData 批量添加搜索数据
func (s *AdminService) BatchAddSearchData(req *model.BatchAddDataRequest) (*model.BatchAddDataResponse, error) {
	response := &model.BatchAddDataResponse{
		Success:        false,
		TotalAdded:     0,
		TotalFailed:    0,
		ESIndexedAll:   true,
		ItemsProcessed: 0,
	}

	if len(req.Items) == 0 {
		return response, fmt.Errorf("批量添加项为空")
	}

	// 逐个处理每个添加请求
	for _, item := range req.Items {
		result, err := s.AddSearchData(&item)
		response.ItemsProcessed++

		if err != nil {
			log.Printf("批量添加失败[%d]: %v", response.ItemsProcessed, err)
			response.TotalFailed++
			continue
		}

		response.TotalAdded += result.AddedCount
		if !result.ESIndexed {
			response.ESIndexedAll = false
		}
	}

	if response.TotalAdded > 0 {
		response.Success = true
		response.Message = fmt.Sprintf("批量处理完成: 成功 %d 条, 失败 %d 条",
			response.TotalAdded, response.TotalFailed)
	} else {
		response.Message = "批量添加全部失败"
	}

	return response, nil
}

// GetSearchDataByKeyword 根据关键词查询数据（分页）
func (s *AdminService) GetSearchDataByKeyword(keyword string, page, pageSize int) ([]model.SearchData, int64, error) {
	db := config.DB
	if db == nil {
		return nil, 0, fmt.Errorf("数据库未初始化")
	}

	var total int64
	var dataList []model.SearchData

	// 构建查询
	query := db.Model(&model.SearchData{})

	if keyword != "" {
		query = query.Where("keyword LIKE ?", "%"+keyword+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询总数失败: %v", err)
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&dataList).Error; err != nil {
		return nil, 0, fmt.Errorf("查询数据失败: %v", err)
	}

	return dataList, total, nil
}

// DeleteSearchData 删除搜索数据
func (s *AdminService) DeleteSearchData(id uint) error {
	db := config.DB
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	result := db.Delete(&model.SearchData{}, id)
	if result.Error != nil {
		return fmt.Errorf("删除数据失败: %v", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("数据不存在")
	}

	return nil
}

// UpdateSearchData 更新搜索数据
func (s *AdminService) UpdateSearchData(id uint, updates map[string]interface{}) error {
	db := config.DB
	if db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	// 添加更新时间
	updates["updated_db"] = time.Now()

	result := db.Model(&model.SearchData{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("更新数据失败: %v", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("数据不存在")
	}

	return nil
}

// SearchLocalData 从MySQL搜索本地数据
func (s *AdminService) SearchLocalData(keyword string) ([]model.SearchResult, error) {
	db := config.DB
	if db == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}

	var searchDataList []model.SearchData

	// 模糊查询关键词
	err := db.Where("keyword LIKE ? OR title LIKE ?", "%"+keyword+"%", "%"+keyword+"%").
		Order("created_at DESC").
		Limit(100). // 限制返回数量
		Find(&searchDataList).Error

	if err != nil {
		return nil, fmt.Errorf("查询本地数据失败: %v", err)
	}

	// 转换为SearchResult格式
	results := make([]model.SearchResult, 0, len(searchDataList))
	for _, data := range searchDataList {
		// 解析标签
		var tags []string
		if data.Tags != "" {
			json.Unmarshal([]byte(data.Tags), &tags)
		}

		// 解析时间
		datetime, _ := time.Parse("2006-01-02 15:04:05", data.Time)

		// 创建链接
		var links []model.Link
		if data.URL != "" {
			links = append(links, model.Link{
				Type: data.Type,
				URL:  data.URL,
			})
		}

		result := model.SearchResult{
			UniqueID: fmt.Sprintf("local-%d", data.ID),
			Channel:  "local-db", // 标记为本地数据
			Datetime: datetime,
			Title:    data.Title,
			Content:  data.Title,
			Links:    links,
			Tags:     tags,
		}
		results = append(results, result)
	}

	return results, nil
}

// SaveSearchResults 保存搜索结果到MySQL和ES
func (s *AdminService) SaveSearchResults(keyword string, source string, results []model.SearchResult) error {
	if len(results) == 0 {
		return nil
	}

	// 准备添加数据请求
	req := &model.AddDataRequest{
		Keyword: keyword,
		Source:  source,
		Results: results,
	}

	// 调用AddSearchData保存到MySQL和ES
	response, err := s.AddSearchData(req)
	if err != nil {
		return fmt.Errorf("保存搜索结果失败: %v", err)
	}

	if !response.Success {
		return fmt.Errorf("保存搜索结果失败: %s", response.Message)
	}

	log.Printf("✅ 保存搜索结果成功: 关键词=%s, 来源=%s, 数量=%d", keyword, source, response.AddedCount)
	return nil
}
