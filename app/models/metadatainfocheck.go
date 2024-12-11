package models

type MetaDataInfoCheck struct {
	ID       int64 `gorm:"column:id;primaryKey;not null;autoIncrement;comment:自增ID"`
	UID      int64 `gorm:"column:uid;not null;comment:唯一ID"`
	IsMerged bool  `gorm:"column:merge;not null;comment:是否合并"`
}
