package repo

import (
	"github.com/qinguoyi/ObjectStorageProxy/app/models"
	"gorm.io/gorm"
)

// TaskLogRepo .
var TaskLogRepo = newTaskLogRepo()

func newTaskLogRepo() *taskLogRepo {
	return &taskLogRepo{}
}

type taskLogRepo struct{}

func (r *taskLogRepo) Create(db *gorm.DB, m *models.TaskLog) error {
	err := db.Create(m).Error
	return err
}

func (r *taskLogRepo) UpdateColumn(db *gorm.DB, logID int64, columns map[string]interface{}) error {
	err := db.Model(&models.TaskLog{}).Where("id = ?", logID).Updates(columns).Error
	return err
}
