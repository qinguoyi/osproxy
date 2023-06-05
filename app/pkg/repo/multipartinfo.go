package repo

import (
	"github.com/qinguoyi/ObjectStorageProxy/app/models"
	"gorm.io/gorm"
)

type multiPartInfoRepo struct{}

func NewMultiPartInfoRepo() *multiPartInfoRepo { return &multiPartInfoRepo{} }

// GetPartMaxNumByUid .
func (r *multiPartInfoRepo) GetPartMaxNumByUid(db *gorm.DB, uidList []int64) ([]models.PartInfo, error) {
	var ret []models.PartInfo

	if err := db.Model(&models.MultiPartInfo{}).
		Select("storage_uid, max(chunk_num) as max_chunk").
		Where("storage_uid in ?", uidList).
		Group("storage_uid").Find(&ret).Error; err != nil {
		return nil, err
	}

	return ret, nil
}

// GetPartNumByUid .
func (r *multiPartInfoRepo) GetPartNumByUid(db *gorm.DB, uid int64) ([]models.MultiPartInfo, error) {
	var ret []models.MultiPartInfo
	if err := db.Model(&models.MultiPartInfo{}).Where("storage_uid = ?", uid).Find(&ret).Error; err != nil {
		return nil, err
	}

	return ret, nil
}

// GetPartInfo .
func (r *multiPartInfoRepo) GetPartInfo(db *gorm.DB, uid, num int64, md5 string) ([]models.MultiPartInfo, error) {
	var ret []models.MultiPartInfo
	if err := db.Model(&models.MultiPartInfo{}).Where(
		"storage_uid = ? and chunk_num  = ? and part_md5 = ? and status = 1", uid, num, md5,
	).Find(&ret).Error; err != nil {
		return nil, err
	}
	return ret, nil
}

// Updates .
func (r *multiPartInfoRepo) Updates(db *gorm.DB, uid int64, columns map[string]interface{}) error {
	err := db.Model(&models.MultiPartInfo{}).Where("uid = ?", uid).Updates(columns).Error
	return err
}

// Create .
func (r *multiPartInfoRepo) Create(db *gorm.DB, m *models.MultiPartInfo) error {
	err := db.Create(m).Error
	return err
}

// BatchCreate .
func (r *multiPartInfoRepo) BatchCreate(db *gorm.DB, m *[]models.MultiPartInfo) error {
	err := db.Create(m).Error
	return err
}
