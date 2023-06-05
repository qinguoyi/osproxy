package models

import "time"

// MultiPartInfo 分片信息
type MultiPartInfo struct {
	ID           int        `gorm:"column:id;primaryKey;not null;autoIncrement;comment:自增ID"`
	StorageUid   int64      `gorm:"column:storage_uid;not null;comment:存储UID"`
	ChunkNum     int        `gorm:"column:chunk_num;not null;comment:分片序号"`
	Bucket       string     `gorm:"column:bucket;not null;comment:桶"`
	StorageName  string     `gorm:"column:storage_name;not null;comment:存储名称"`
	StorageSize  int64      `gorm:"column:storage_size;comment:文件大小"`
	PartFileName string     `gorm:"column:part_file_name;not null;comment:分片文件名称"`
	PartMd5      string     `gorm:"column:part_md5;not null;comment:分片md5"`
	Status       int        `gorm:"column:status;not null;comment:状态信息"`
	CreatedAt    *time.Time `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt    *time.Time `gorm:"column:updated_at;not null;comment:更新时间"`
}

// PartInfo .
type PartInfo struct {
	StorageUid int64 `json:"storage_uid"`
	MaxChunk   int   `json:"max_chunk"`
}

// MergeInfo .
type MergeInfo struct {
	StorageUid int64 `json:"storageUid"`
	ChunkSum   int64 `json:"chunkSum"`
}
