package repo

import (
	"github.com/qinguoyi/ObjectStorageProxy/app/models"
	"gorm.io/gorm"
)

type metaDataInfoRepo struct{}

func NewMetaDataInfoRepo() *metaDataInfoRepo { return &metaDataInfoRepo{} }

// GetByUid .
func (r *metaDataInfoRepo) GetByUid(db *gorm.DB, uid int64) (*models.MetaDataInfo, error) {
	ret := &models.MetaDataInfo{}
	if err := db.Where("uid = ?", uid).First(ret).Error; err != nil {
		return ret, err
	}
	return ret, nil
}

// GetResumeByMd5 .
func (r *metaDataInfoRepo) GetResumeByMd5(db *gorm.DB, md5 []string) ([]models.MetaDataInfo, error) {
	var ret []models.MetaDataInfo
	if err := db.Where("md5 in ? and status = 1 and multi_part = ?", md5, false).
		Find(&ret).Error; err != nil {
		return ret, err
	}
	return ret, nil
}

// GetPartByMd5 .
func (r *metaDataInfoRepo) GetPartByMd5(db *gorm.DB, md5 []string) ([]models.MetaDataInfo, error) {
	var ret []models.MetaDataInfo
	if err := db.Where("md5 in ? and status = -1 and multi_part = ? ", md5, true).
		Find(&ret).Error; err != nil {
		return ret, err
	}
	return ret, nil
}

// GetPartByUid .
func (r *metaDataInfoRepo) GetPartByUid(db *gorm.DB, uid int64) ([]models.MetaDataInfo, error) {
	var ret []models.MetaDataInfo
	if err := db.Where("uid = ? and status = -1 and multi_part = ? ", uid, true).
		Find(&ret).Error; err != nil {
		return ret, err
	}
	return ret, nil
}

// GetByUidList .
func (r *metaDataInfoRepo) GetByUidList(db *gorm.DB, uid []int64) ([]models.MetaDataInfo, error) {
	var ret []models.MetaDataInfo
	if err := db.Where("uid in ?", uid).Find(&ret).Error; err != nil {
		return ret, err
	}
	return ret, nil
}

// BatchCreate .
func (r *metaDataInfoRepo) BatchCreate(db *gorm.DB, m *[]models.MetaDataInfo) error {
	err := db.Create(m).Error
	return err
}

// Updates .
func (r *metaDataInfoRepo) Updates(db *gorm.DB, uid int64, columns map[string]interface{}) error {
	err := db.Model(&models.MetaDataInfo{}).Where("uid = ?", uid).Updates(columns).Error
	return err
}
