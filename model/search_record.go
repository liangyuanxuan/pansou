package model

import "time"

// SearchRecord 搜索记录数据库模型
type SearchRecord struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	Keyword   string    `gorm:"index:idx_keyword;type:varchar(255);not null" json:"keyword"` // 搜索关键词
	MessageID string    `gorm:"type:varchar(100)" json:"message_id"`
	UniqueID  string    `gorm:"uniqueIndex;type:varchar(255);not null" json:"unique_id"` // 全局唯一ID
	Channel   string    `gorm:"index;type:varchar(100)" json:"channel"`
	Datetime  time.Time `gorm:"index" json:"datetime"`
	Title     string    `gorm:"type:text;index:idx_fulltext_title,class:FULLTEXT,option:WITH PARSER ngram" json:"title"`         // 全文索引
	Content   string    `gorm:"type:longtext;index:idx_fulltext_content,class:FULLTEXT,option:WITH PARSER ngram" json:"content"` // 全文索引
	Tags      string    `gorm:"type:text" json:"tags"`                                                                           // JSON存储
	Images    string    `gorm:"type:text" json:"images"`                                                                         // JSON存储
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (SearchRecord) TableName() string {
	return "search_records"
}

// LinkRecord 链接记录数据库模型
type LinkRecord struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	SearchID  uint      `gorm:"index:idx_search_id;not null" json:"search_id"` // 关联SearchRecord的ID
	Type      string    `gorm:"index;type:varchar(50)" json:"type"`            // 网盘类型
	URL       string    `gorm:"uniqueIndex;type:varchar(500);not null" json:"url"`
	Password  string    `gorm:"type:varchar(100)" json:"password"`
	Datetime  time.Time `json:"datetime"`
	WorkTitle string    `gorm:"type:text" json:"work_title"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 指定表名
func (LinkRecord) TableName() string {
	return "link_records"
}
