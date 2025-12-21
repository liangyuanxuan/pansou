package elasticsearch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	// SearchCacheIndex 搜索缓存索引名
	SearchCacheIndex = "pansou_search_cache"
	// SearchHistoryIndex 搜索历史索引名
	SearchHistoryIndex = "pansou_search_history"
)

// searchCacheMapping 搜索缓存索引的映射定义
var searchCacheMapping = map[string]interface{}{
	"mappings": map[string]interface{}{
		"properties": map[string]interface{}{
			"cache_key": map[string]interface{}{
				"type": "keyword",
			},
			"keyword": map[string]interface{}{
				"type": "text",
				"fields": map[string]interface{}{
					"keyword": map[string]interface{}{
						"type": "keyword",
					},
				},
			},
			"channels": map[string]interface{}{
				"type": "keyword",
			},
			"plugins": map[string]interface{}{
				"type": "keyword",
			},
			"results": map[string]interface{}{
				"type":    "object",
				"enabled": false, // 不索引results字段，仅存储
			},
			"created_at": map[string]interface{}{
				"type": "date",
			},
			"updated_at": map[string]interface{}{
				"type": "date",
			},
			"expire_at": map[string]interface{}{
				"type": "date",
			},
			"result_count": map[string]interface{}{
				"type": "integer",
			},
			"source_type": map[string]interface{}{
				"type": "keyword",
			},
		},
	},
	"settings": map[string]interface{}{
		"number_of_shards":   1,
		"number_of_replicas": 0,
	},
}

// searchHistoryMapping 搜索历史索引的映射定义
var searchHistoryMapping = map[string]interface{}{
	"mappings": map[string]interface{}{
		"properties": map[string]interface{}{
			"keyword": map[string]interface{}{
				"type": "text",
				"fields": map[string]interface{}{
					"keyword": map[string]interface{}{
						"type": "keyword",
					},
				},
			},
			"search_count": map[string]interface{}{
				"type": "integer",
			},
			"last_search": map[string]interface{}{
				"type": "date",
			},
			"created_at": map[string]interface{}{
				"type": "date",
			},
		},
	},
	"settings": map[string]interface{}{
		"number_of_shards":   1,
		"number_of_replicas": 0,
	},
}

// InitIndices 初始化所有索引
func InitIndices() error {
	if !IsEnabled() {
		return nil
	}

	// 创建搜索缓存索引
	if err := createIndexIfNotExists(SearchCacheIndex, searchCacheMapping); err != nil {
		return fmt.Errorf("创建搜索缓存索引失败: %w", err)
	}

	// 创建搜索历史索引
	if err := createIndexIfNotExists(SearchHistoryIndex, searchHistoryMapping); err != nil {
		return fmt.Errorf("创建搜索历史索引失败: %w", err)
	}

	return nil
}

// createIndexIfNotExists 创建索引（如果不存在）
func createIndexIfNotExists(indexName string, mapping map[string]interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 检查索引是否存在
	res, err := ESClient.Indices.Exists([]string{indexName})
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// 索引已存在
	if res.StatusCode == 200 {
		return nil
	}

	// 创建索引
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(mapping); err != nil {
		return err
	}

	res, err = ESClient.Indices.Create(
		indexName,
		ESClient.Indices.Create.WithContext(ctx),
		ESClient.Indices.Create.WithBody(&buf),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("创建索引失败: %s", res.Status())
	}

	return nil
}

// DeleteExpiredCache 删除过期的缓存
func DeleteExpiredCache() error {
	if !IsEnabled() {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 构建删除查询
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"range": map[string]interface{}{
				"expire_at": map[string]interface{}{
					"lt": time.Now().Format(time.RFC3339),
				},
			},
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return err
	}

	// 执行删除
	res, err := ESClient.DeleteByQuery(
		[]string{SearchCacheIndex},
		&buf,
		ESClient.DeleteByQuery.WithContext(ctx),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return fmt.Errorf("解析删除响应失败: %w", err)
		}
		return fmt.Errorf("删除过期缓存失败: %v", e)
	}

	return nil
}

// StartCleanupJob 启动定期清理任务
func StartCleanupJob(interval time.Duration) {
	if !IsEnabled() {
		return
	}

	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			if err := DeleteExpiredCache(); err != nil {
				fmt.Printf("清理过期缓存失败: %v\n", err)
			}
		}
	}()
}

// GetIndexStats 获取索引统计信息
func GetIndexStats(indexName string) (map[string]interface{}, error) {
	if !IsEnabled() {
		return nil, fmt.Errorf("ES未启用")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := ESClient.Indices.Stats(
		ESClient.Indices.Stats.WithContext(ctx),
		ESClient.Indices.Stats.WithIndex(indexName),
	)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("获取索引统计失败: %s", res.Status())
	}

	var stats map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&stats); err != nil {
		return nil, err
	}

	return stats, nil
}

// OptimizeIndex 优化索引
func OptimizeIndex(indexName string) error {
	if !IsEnabled() {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res, err := ESClient.Indices.Forcemerge(
		ESClient.Indices.Forcemerge.WithContext(ctx),
		ESClient.Indices.Forcemerge.WithIndex(indexName),
		ESClient.Indices.Forcemerge.WithMaxNumSegments(1),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("优化索引失败: %s", res.Status())
	}

	return nil
}

// ClearCache 清空指定关键词的缓存
func ClearCache(keyword string) error {
	if !IsEnabled() {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 构建删除查询
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{
				"keyword.keyword": strings.ToLower(keyword),
			},
		},
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return err
	}

	// 执行删除
	res, err := ESClient.DeleteByQuery(
		[]string{SearchCacheIndex},
		&buf,
		ESClient.DeleteByQuery.WithContext(ctx),
	)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("清空缓存失败: %s", res.Status())
	}

	return nil
}
