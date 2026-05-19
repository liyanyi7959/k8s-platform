package model

import "time"

// KbNode 为知识库“节点”模型：同时表示目录（folder）与文档（doc）。
//
// 设计说明：
// - 用一张表表示树结构，靠 parent_id 组织层级
// - Type 区分节点类型：folder/doc
// - Sort 用于同一父目录下的排序（越小越靠前）
type KbNode struct {
	ID        uint64     `gorm:"column:id;primaryKey;autoIncrement"`
	Type      string     `gorm:"column:type;type:varchar(16);not null"`
	ParentID  uint64     `gorm:"column:parent_id;not null;default:0;index:idx_kb_nodes_parent,priority:1"`
	Name      string     `gorm:"column:name;type:varchar(80);not null"`
	IconURL   *string    `gorm:"column:icon_url;type:mediumtext"`
	Sort      int        `gorm:"column:sort;not null;default:0;index:idx_kb_nodes_parent,priority:2"`
	CreatedAt time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt *time.Time `gorm:"column:deleted_at;index"`
}

// TableName 指定表名。
func (KbNode) TableName() string { return "kb_nodes" }

// KbDoc 为知识库“文档内容”模型，与 KbNode（doc 类型）一对一关联。
//
// - NodeID 为主键（同时也是关联的 kb_nodes.id）
// - ContentMD 存 Markdown 原文
// - CurrentVersion 用于乐观锁（保存时提交 base_version 校验）
// - Views 为浏览次数（可用于热门排序）
type KbDoc struct {
	NodeID         uint64     `gorm:"column:node_id;primaryKey"`
	ContentMD      string     `gorm:"column:content_md;type:longtext;not null"`
	CurrentVersion int        `gorm:"column:current_version;not null;default:0"`
	Views          uint64     `gorm:"column:views;not null;default:0"`
	CreatedAt      time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt      *time.Time `gorm:"column:deleted_at;index"`
}

// TableName 指定表名。
func (KbDoc) TableName() string { return "kb_docs" }

type KbDocTag struct {
	DocNodeID uint64    `gorm:"column:doc_node_id;primaryKey"`
	Tag       string    `gorm:"column:tag;type:varchar(64);primaryKey"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (KbDocTag) TableName() string { return "kb_doc_tags" }
