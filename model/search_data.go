package model

import (
	"time"
)

// SearchData 原始搜索数据模型（存储到MySQL）
type SearchData struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Keyword   string    `gorm:"type:varchar(255);not null;index:idx_keyword" json:"keyword"` // 搜索关键词
	Source    string    `gorm:"type:varchar(100);not null;index:idx_source" json:"source"`   // 数据来源（plugin name或channel）
	Title     string    `gorm:"type:varchar(500);not null" json:"title"`                     // 资源标题
	URL       string    `gorm:"type:varchar(1000);not null" json:"url"`                      // 资源链接
	Size      string    `gorm:"type:varchar(50)" json:"size"`                                // 文件大小
	Time      string    `gorm:"type:varchar(100)" json:"time"`                               // 发布时间
	UpdatedAt string    `gorm:"type:varchar(100)" json:"updated_at"`                         // 更新时间
	Type      string    `gorm:"type:varchar(50);index:idx_type" json:"type"`                 // 资源类型
	Hot       int       `gorm:"default:0" json:"hot"`                                        // 热度
	Tags      string    `gorm:"type:text" json:"tags"`                                       // 标签（JSON数组）
	CreatedAt time.Time `gorm:"autoCreateTime;index:idx_created" json:"created_at"`          // 创建时间
	UpdatedDB time.Time `gorm:"autoUpdateTime" json:"updated_db"`                            // 数据库更新时间
}

// TableName 指定表名
func (SearchData) TableName() string {
	return "search_data"
}

// AddDataRequest 添加数据请求结构
type AddDataRequest struct {
	Keyword string         `json:"keyword" binding:"required"` // 搜索关键词
	Source  string         `json:"source" binding:"required"`  // 数据来源
	Results []SearchResult `json:"results" binding:"required"` // 搜索结果列表
}

// AddDataResponse 添加数据响应结构
type AddDataResponse struct {
	Success      bool   `json:"success"`
	AddedCount   int    `json:"added_count"` // 添加到MySQL的数量
	ESIndexed    bool   `json:"es_indexed"`  // 是否已索引到ES
	Message      string `json:"message"`
	FailedCount  int    `json:"failed_count,omitempty"`
	FailedErrors string `json:"failed_errors,omitempty"`
}

// BatchAddDataRequest 批量添加数据请求
type BatchAddDataRequest struct {
	Items []AddDataRequest `json:"items" binding:"required,dive"` // 批量添加项
}

// BatchAddDataResponse 批量添加数据响应
type BatchAddDataResponse struct {
	Success        bool   `json:"success"`
	TotalAdded     int    `json:"total_added"`    // 总共添加的数量
	TotalFailed    int    `json:"total_failed"`   // 失败的数量
	ESIndexedAll   bool   `json:"es_indexed_all"` // 是否全部索引到ES
	Message        string `json:"message"`
	ItemsProcessed int    `json:"items_processed"` // 处理的条目数
}
