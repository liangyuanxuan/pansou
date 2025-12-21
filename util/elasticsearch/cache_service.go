package elasticsearch

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"pansou/config"
	"pansou/model"
	"pansou/util/cache"
)

// CacheService ES缓存服务
type CacheService struct {
	ttl time.Duration
}

// NewCacheService 创建缓存服务实例
func NewCacheService() *CacheService {
	ttl := time.Duration(config.AppConfig.CacheTTLMinutes) * time.Minute
	return &CacheService{
		ttl: ttl,
	}
}

// GenerateCacheKey 生成缓存键
func GenerateCacheKey(keyword string, channels []string, plugins []string, sourceType string) string {
	// 使用与现有cache包相同的逻辑
	if sourceType == "tg" || (sourceType == "all" && (plugins == nil || len(plugins) == 0)) {
		return cache.GenerateTGCacheKey(keyword, channels)
	}
	return cache.GeneratePluginCacheKey(keyword, plugins)
}

// Get 从ES获取缓存
func (s *CacheService) Get(cacheKey string) ([]model.SearchResult, bool, error) {
	if !IsEnabled() {
		return nil, false, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 构建查询
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"term": map[string]interface{}{
							"cache_key": cacheKey,
						},
					},
					{
						"range": map[string]interface{}{
							"expire_at": map[string]interface{}{
								"gte": time.Now().Format(time.RFC3339),
							},
						},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, false, err
	}

	// 执行搜索
	res, err := ESClient.Search(
		ESClient.Search.WithContext(ctx),
		ESClient.Search.WithIndex(SearchCacheIndex),
		ESClient.Search.WithBody(&buf),
	)
	if err != nil {
		return nil, false, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, false, fmt.Errorf("ES查询失败: %s", res.Status())
	}

	// 解析响应
	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, false, err
	}

	hits := result["hits"].(map[string]interface{})["hits"].([]interface{})
	if len(hits) == 0 {
		return nil, false, nil // 缓存未命中
	}

	// 获取第一个文档
	doc := hits[0].(map[string]interface{})["_source"].(map[string]interface{})

	// 解析results字段
	resultsData, err := json.Marshal(doc["results"])
	if err != nil {
		return nil, false, err
	}

	var results []model.SearchResult
	if err := json.Unmarshal(resultsData, &results); err != nil {
		return nil, false, err
	}

	return results, true, nil
}

// Set 设置缓存到ES
func (s *CacheService) Set(cacheKey string, keyword string, channels []string, plugins []string, sourceType string, results []model.SearchResult) error {
	if !IsEnabled() {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	now := time.Now()

	// 构建文档
	doc := SearchCacheDoc{
		CacheKey:    cacheKey,
		Keyword:     keyword,
		Channels:    channels,
		Plugins:     plugins,
		Results:     results,
		CreatedAt:   now,
		UpdatedAt:   now,
		ExpireAt:    now.Add(s.ttl),
		ResultCount: len(results),
		SourceType:  sourceType,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(doc); err != nil {
		return err
	}

	// 使用cache_key作为文档ID，实现upsert
	docID := generateDocID(cacheKey)

	res, err := ESClient.Index(
		SearchCacheIndex,
		&buf,
		ESClient.Index.WithContext(ctx),
		ESClient.Index.WithDocumentID(docID),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("索引文档失败: %s", res.Status())
	}

	return nil
}

// Update 更新缓存（用于插件异步更新）
func (s *CacheService) Update(cacheKey string, newResults []model.SearchResult) error {
	if !IsEnabled() {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 先获取现有缓存
	existingResults, hit, err := s.Get(cacheKey)
	if err != nil {
		return err
	}

	// 合并结果
	var mergedResults []model.SearchResult
	if hit {
		mergedResults = mergeSearchResults(existingResults, newResults)
	} else {
		mergedResults = newResults
	}

	// 构建更新文档
	updateDoc := map[string]interface{}{
		"doc": map[string]interface{}{
			"results":      mergedResults,
			"updated_at":   time.Now(),
			"result_count": len(mergedResults),
		},
		"doc_as_upsert": true,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(updateDoc); err != nil {
		return err
	}

	docID := generateDocID(cacheKey)

	res, err := ESClient.Update(
		SearchCacheIndex,
		docID,
		&buf,
		ESClient.Update.WithContext(ctx),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("更新文档失败: %s", res.Status())
	}

	return nil
}

// RecordSearchHistory 记录搜索历史
func (s *CacheService) RecordSearchHistory(keyword string) error {
	if !IsEnabled() {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 使用upsert脚本更新搜索次数
	script := map[string]interface{}{
		"script": map[string]interface{}{
			"source": "ctx._source.search_count += 1; ctx._source.last_search = params.now",
			"params": map[string]interface{}{
				"now": time.Now(),
			},
		},
		"upsert": map[string]interface{}{
			"keyword":      keyword,
			"search_count": 1,
			"last_search":  time.Now(),
			"created_at":   time.Now(),
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(script); err != nil {
		return err
	}

	docID := generateDocID(keyword)

	res, err := ESClient.Update(
		SearchHistoryIndex,
		docID,
		&buf,
		ESClient.Update.WithContext(ctx),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("记录搜索历史失败: %s", res.Status())
	}

	return nil
}

// GetHotKeywords 获取热门搜索关键词
func (s *CacheService) GetHotKeywords(limit int) ([]SearchHistoryDoc, error) {
	if !IsEnabled() {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 构建查询 - 按搜索次数排序
	query := map[string]interface{}{
		"size": limit,
		"sort": []map[string]interface{}{
			{
				"search_count": map[string]interface{}{
					"order": "desc",
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, err
	}

	res, err := ESClient.Search(
		ESClient.Search.WithContext(ctx),
		ESClient.Search.WithIndex(SearchHistoryIndex),
		ESClient.Search.WithBody(&buf),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("查询热门关键词失败: %s", res.Status())
	}

	// 解析响应
	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	hits := result["hits"].(map[string]interface{})["hits"].([]interface{})

	histories := make([]SearchHistoryDoc, 0, len(hits))
	for _, hit := range hits {
		doc := hit.(map[string]interface{})["_source"].(map[string]interface{})

		historyData, err := json.Marshal(doc)
		if err != nil {
			continue
		}

		var history SearchHistoryDoc
		if err := json.Unmarshal(historyData, &history); err != nil {
			continue
		}

		histories = append(histories, history)
	}

	return histories, nil
}

// generateDocID 生成文档ID
func generateDocID(key string) string {
	hash := md5.Sum([]byte(key))
	return fmt.Sprintf("%x", hash)
}

// mergeSearchResults 合并搜索结果（去重）
func mergeSearchResults(existing []model.SearchResult, newResults []model.SearchResult) []model.SearchResult {
	// 使用map进行去重
	resultMap := make(map[string]model.SearchResult)

	// 先添加现有结果
	for _, result := range existing {
		key := generateResultKey(result)
		resultMap[key] = result
	}

	// 合并新结果
	for _, newResult := range newResults {
		key := generateResultKey(newResult)
		if existingResult, exists := resultMap[key]; exists {
			// 选择信息更完整的结果
			resultMap[key] = selectBetterResult(existingResult, newResult)
		} else {
			resultMap[key] = newResult
		}
	}

	// 转换回切片
	merged := make([]model.SearchResult, 0, len(resultMap))
	for _, result := range resultMap {
		merged = append(merged, result)
	}

	// 按时间排序
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].Datetime.After(merged[j].Datetime)
	})

	return merged
}

// generateResultKey 生成结果的唯一标识键
func generateResultKey(result model.SearchResult) string {
	if result.UniqueID != "" {
		return result.UniqueID
	}
	if result.MessageID != "" {
		return result.MessageID
	}
	return fmt.Sprintf("title_%s_%s", result.Title, result.Channel)
}

// selectBetterResult 选择信息更完整的结果
func selectBetterResult(existing, new model.SearchResult) model.SearchResult {
	existingScore := calculateCompletenessScore(existing)
	newScore := calculateCompletenessScore(new)

	if newScore > existingScore {
		return new
	}
	return existing
}

// calculateCompletenessScore 计算结果信息的完整度得分
func calculateCompletenessScore(result model.SearchResult) int {
	score := 0

	if result.UniqueID != "" {
		score += 10
	}

	if len(result.Links) > 0 {
		score += 5 + len(result.Links)
	}

	if result.Content != "" {
		score += 3
	}

	score += len(result.Title) / 10

	if result.Channel != "" {
		score += 2
	}

	score += len(result.Tags)

	return score
}

// SearchByKeyword 根据关键词模糊搜索缓存
func (s *CacheService) SearchByKeyword(keyword string, limit int) ([]SearchCacheDoc, error) {
	if !IsEnabled() {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 构建模糊查询
	query := map[string]interface{}{
		"size": limit,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{
					{
						"match": map[string]interface{}{
							"keyword": keyword,
						},
					},
					{
						"range": map[string]interface{}{
							"expire_at": map[string]interface{}{
								"gte": time.Now().Format(time.RFC3339),
							},
						},
					},
				},
			},
		},
		"sort": []map[string]interface{}{
			{
				"updated_at": map[string]interface{}{
					"order": "desc",
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return nil, err
	}

	res, err := ESClient.Search(
		ESClient.Search.WithContext(ctx),
		ESClient.Search.WithIndex(SearchCacheIndex),
		ESClient.Search.WithBody(&buf),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("搜索缓存失败: %s", res.Status())
	}

	// 解析响应
	var result map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	hits := result["hits"].(map[string]interface{})["hits"].([]interface{})

	caches := make([]SearchCacheDoc, 0, len(hits))
	for _, hit := range hits {
		doc := hit.(map[string]interface{})["_source"].(map[string]interface{})

		cacheData, err := json.Marshal(doc)
		if err != nil {
			continue
		}

		var cacheDoc SearchCacheDoc
		if err := json.Unmarshal(cacheData, &cacheDoc); err != nil {
			continue
		}

		caches = append(caches, cacheDoc)
	}

	return caches, nil
}
