package knowledge

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// 文档状态
const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusReady      = "ready"
	StatusFailed     = "failed"
)

// 来源类型
const (
	SourceManual = "manual"
	SourceUpload = "upload"
	SourceURL    = "url"
)

// KnowledgeDoc 是文档元数据实体
type KnowledgeDoc struct {
	ID              string    `gorm:"primaryKey" json:"id"`
	Title           string    `gorm:"not null" json:"title"`
	Country         string    `json:"country"`
	SourceType      string    `gorm:"not null" json:"sourceType"`
	SourceURL       *string   `json:"sourceUrl,omitempty"`
	OriginalContent string    `gorm:"type:text" json:"-"`
	Status          string    `gorm:"not null;default:pending" json:"status"`
	ErrorMessage    *string   `json:"errorMessage,omitempty"`
	ChunkCount      int       `gorm:"not null;default:0" json:"chunkCount"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

func (KnowledgeDoc) TableName() string { return "knowledge_docs" }

// KnowledgeChunk 是分块 + 向量
type KnowledgeChunk struct {
	ID        string          `gorm:"primaryKey" json:"id"`
	DocID     string          `gorm:"not null;uniqueIndex:idx_doc_seq,priority:1" json:"docId"`
	Seq       int             `gorm:"not null;uniqueIndex:idx_doc_seq,priority:2" json:"seq"`
	Content   string          `gorm:"not null;type:text" json:"content"`
	Embedding JSONFloat32     `gorm:"type:jsonb" json:"-"`
	Metadata  json.RawMessage `gorm:"type:jsonb" json:"metadata,omitempty"`
	CreatedAt time.Time       `json:"createdAt"`
}

func (KnowledgeChunk) TableName() string { return "knowledge_chunks" }

// JSONFloat32 自定义类型 — GORM 用 JSON 列存 []float32（兼容 SQLite）
// 生产 PG 路径下 repo_gorm 会改用 pgvector.Vector 类型，跳过此字段
type JSONFloat32 []float32

func (v JSONFloat32) GormDataType() string { return "json" }

// Value 实现 driver.Valuer：编码为 JSON bytes
func (v JSONFloat32) Value() (driver.Value, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}

// Scan 实现 sql.Scanner：从 JSON bytes 解码
func (v *JSONFloat32) Scan(value interface{}) error {
	if value == nil {
		*v = nil
		return nil
	}
	var b []byte
	switch t := value.(type) {
	case []byte:
		b = t
	case string:
		b = []byte(t)
	default:
		return fmt.Errorf("unsupported scan type: %T", value)
	}
	if len(b) == 0 {
		*v = nil
		return nil
	}
	return json.Unmarshal(b, v)
}

// DTOs
type CreateDocRequest struct {
	Title      string  `json:"title" binding:"required,max=200"`
	Country    string  `json:"country" binding:"required,max=100"`
	SourceType string  `json:"sourceType" binding:"required,oneof=manual upload url"`
	SourceURL  *string `json:"sourceUrl,omitempty" binding:"omitempty,url"`
	Content    string  `json:"content" binding:"required_if=SourceType manual,required_if=SourceType upload"`
}

type DocDTO struct {
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Country      string    `json:"country"`
	SourceType   string    `json:"sourceType"`
	SourceURL    *string   `json:"sourceUrl,omitempty"`
	Status       string    `json:"status"`
	ErrorMessage *string   `json:"errorMessage,omitempty"`
	ChunkCount   int       `json:"chunkCount"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (d *KnowledgeDoc) ToDTO() DocDTO {
	return DocDTO{
		ID: d.ID, Title: d.Title, Country: d.Country, SourceType: d.SourceType,
		SourceURL: d.SourceURL, Status: d.Status, ErrorMessage: d.ErrorMessage,
		ChunkCount: d.ChunkCount, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt,
	}
}

type ChunkHit struct {
	ID      string  `json:"id"`
	DocID   string  `json:"docId"`
	Title   string  `json:"title"`
	Snippet string  `json:"snippet"`
	Score   float64 `json:"score"`
}

type SearchRequest struct {
	Query   string `json:"query" binding:"required"`
	Country string `json:"country,omitempty"`
	TopK    int    `json:"topK,omitempty"`
}

type SearchResponse struct {
	Chunks []ChunkHit `json:"chunks"`
}
