package models

import "time"

// TaskInfo .
type TaskInfo struct {
	ID          int64      `gorm:"column:id;primaryKey;not null;autoIncrement;comment:自增ID"`
	Status      int        `json:"Status" gorm:"column:status;not null;index:idx_task_status"`       // 任务状态 0 未执行 1 执行中 2 执行完成 99 执行失败
	TaskType    string     `json:"taskType" gorm:"column:task_type;not null;type:varchar(255)"`      // 任务类型
	UserID      string     `json:"userId" gorm:"column:user_id;type:varchar(255)"`                   // 任务触发者
	ExtraData   string     `json:"extraData" gorm:"column:extra_data;type:text"`                     // 任务补充信息
	NodeId      string     `json:"nodeId" gorm:"column:node_id;type:varchar(255);index:idx_node_id"` // 任务运行节点ID
	TaskLogID   int        `json:"taskLogId" gorm:"column:task_log_id"`                              // 任务日志ID，只展示最新的任务日志ID
	ExecuteTime int        `json:"executeTime" gorm:"column:execute_time;comment:任务执行次数;default:1"`
	CreatedAt   *time.Time `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt   *time.Time `gorm:"column:updated_at;not null;comment:更新时间"`
}

// TaskLog .
type TaskLog struct {
	ID        int64      `gorm:"column:id;primaryKey;not null;autoIncrement;comment:自增ID"`
	TaskID    int64      `json:"TaskID" gorm:"column:task_id;index:idx_task_id"`        // 任务ID，可以看到历史执行任务
	Status    int        `json:"Status" gorm:"column:status;index:idx_task_log_status"` // 任务状态 0 未执行 1 执行中 2 执行完成 99 执行失败
	ErrorInfo string     `json:"ErrorInfo" gorm:"column:error_info;type:text"`          // 错误信息
	CreatedAt *time.Time `gorm:"column:created_at;not null;comment:创建时间"`
	UpdatedAt *time.Time `gorm:"column:updated_at;not null;comment:更新时间"`
}
