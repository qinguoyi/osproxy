package repo

import (
	"github.com/qinguoyi/ObjectStorageProxy/app/models"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/utils"
	"gorm.io/gorm"
)

func NewTaskRepo() *taskInfoRepo {
	return &taskInfoRepo{}
}

type taskInfoRepo struct{}

// GetByID .
func (r *taskInfoRepo) GetByID(db *gorm.DB, taskID int64) (*models.TaskInfo, error) {
	ret := &models.TaskInfo{}
	if err := db.Where("id = ?", taskID).First(ret).Error; err != nil {
		return ret, err
	}
	return ret, nil
}

// PreemptiveTaskByID 抢占任务  这里的update需要查看更新的数量，更新数量为0的时候，err也是nil；其他的update先不管
func (r *taskInfoRepo) PreemptiveTaskByID(db *gorm.DB, taskID int64, nodeId string) int64 {
	affected := db.Model(&models.TaskInfo{}).Where("id = ? and status = ?", taskID, utils.TaskStatusUndo).
		UpdateColumns(map[string]interface{}{
			"status":  utils.TaskStatusRunning,
			"node_id": nodeId,
		})
	return affected.RowsAffected
}

// FinishTaskByID 完成任务
func (r *taskInfoRepo) FinishTaskByID(db *gorm.DB, taskID int64) int64 {
	affected := db.Model(&models.TaskInfo{}).Where("id = ? and status = ?", taskID, utils.TaskStatusRunning).
		UpdateColumn("status", utils.TaskStatusFinish)
	return affected.RowsAffected
}

// ErrorTaskByID 任务失败
func (r *taskInfoRepo) ErrorTaskByID(db *gorm.DB, taskID int64) int64 {
	affected := db.Model(&models.TaskInfo{}).Where("id = ? and status = ?", taskID, utils.TaskStatusRunning).
		UpdateColumn("status", utils.TaskStatusError)
	return affected.RowsAffected
}

// ResetTaskByID 重置任务
func (r *taskInfoRepo) ResetTaskByID(db *gorm.DB, taskID int64, nodeId string) int64 {
	affected := db.Model(&models.TaskInfo{}).Where(
		"id = ? and node_id = ? and status =?", taskID, nodeId, utils.TaskStatusRunning).
		UpdateColumns(map[string]interface{}{
			"status":  utils.TaskStatusUndo,
			"node_id": "",
		})
	return affected.RowsAffected
}

// FindByStatus .
func (r *taskInfoRepo) FindByStatus(db *gorm.DB, status int) ([]models.TaskInfo, error) {
	var ret []models.TaskInfo
	if err := db.Where("status = ?", status).Find(&ret).Error; err != nil {
		return ret, err
	}
	return ret, nil
}

// Create .
func (r *taskInfoRepo) Create(db *gorm.DB, m *models.TaskInfo) error {
	err := db.Create(m).Error
	return err
}

// BatchCreate .
func (r *taskInfoRepo) BatchCreate(db *gorm.DB, m []*models.TaskInfo) error {
	err := db.Create(m).Error
	return err
}

// UpdateColumn .
func (r *taskInfoRepo) UpdateColumn(db *gorm.DB, commentID int64, name string, value interface{}) error {
	err := db.Model(&models.TaskInfo{}).Where("id = ?", commentID).UpdateColumn(name, value).Error
	return err
}
