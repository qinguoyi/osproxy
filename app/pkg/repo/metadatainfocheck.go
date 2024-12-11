package repo

import (
	"github.com/qinguoyi/osproxy/app/models"
	"gorm.io/gorm"
)

type metadatainfocheckRepo struct{}

func NewMetaDataInfoCheckRepo() *metadatainfocheckRepo {
	return &metadatainfocheckRepo{}
}

func (r *metadatainfocheckRepo) GetUIDIsMerged(db *gorm.DB, uid int64) (bool, error) {
	ret := &models.MetaDataInfoCheck{}
	if err := db.Where("uid = ?", uid).First(ret).Error; err != nil {
		return false, err
	}
	return ret.IsMerged, nil
}

func (r *metadatainfocheckRepo) UpdateMerge(db *gorm.DB, uid int64, merge bool) error {
	return db.Model(&models.MetaDataInfoCheck{}).Where("uid = ?", uid).Update("merge", merge).Error
}
