package elasticsearch

import (
	"time"

	"pansou/model"
)

// SearchCacheDoc ES中的搜索缓存文档
type SearchCacheDoc struct {
	CacheKey    string               `json:"cache_key"`    // 缓存键
	Keyword     string               `json:"keyword"`      // 搜索关键词
	Channels    []string             `json:"channels"`     // 搜索频道
	Plugins     []string             `json:"plugins"`      // 插件列表
	Results     []model.SearchResult `json:"results"`      // 搜索结果
	CreatedAt   time.Time            `json:"created_at"`   // 创建时间
	UpdatedAt   time.Time            `json:"updated_at"`   // 更新时间
	ExpireAt    time.Time            `json:"expire_at"`    // 过期时间
	ResultCount int                  `json:"result_count"` // 结果数量
	SourceType  string               `json:"source_type"`  // 数据来源类型: tg/plugin/all
}

// SearchHistoryDoc ES中的搜索历史文档
type SearchHistoryDoc struct {
	Keyword     string    `json:"keyword"`      // 搜索关键词
	SearchCount int       `json:"search_count"` // 搜索次数
	LastSearch  time.Time `json:"last_search"`  // 最后搜索时间
	CreatedAt   time.Time `json:"created_at"`   // 创建时间
}
