package dispatch

import (
	"fmt"
	"github.com/qinguoyi/ObjectStorageProxy/app/models"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/event"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/repo"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/utils"
	"github.com/qinguoyi/ObjectStorageProxy/bootstrap/plugins"
	"gorm.io/gorm"
	"sync"
)

type Job struct {
	TaskID   int64  // 任务ID
	TaskType string // 任务类型
}

type Worker struct {
	Wg *sync.WaitGroup
}

func NewWorker() *Worker {
	return &Worker{
		Wg: &sync.WaitGroup{},
	}
}

func (w *Worker) Start() {
	go func() {
		defer w.Wg.Done()
		for {
			select {
			// 这里不能用协程，因为协程内部开协程，两者没有依赖关系，外部协程退出，内部协程仍然在执行
			case job := <-JobQueue:
				handler := event.NewEventsHandler().GetHandler(job.TaskType)
				var lgDB = new(plugins.LangGoDB).Use("default").NewDB()
				// 创建日志数据
				taskLogData := models.TaskLog{
					TaskID: job.TaskID,
					Status: utils.TaskStatusRunning,
				}
				_ = repo.TaskLogRepo.Create(lgDB, &taskLogData)
				_ = repo.NewTaskRepo().UpdateColumn(lgDB, job.TaskID, "task_log_id", taskLogData.ID)

				// 没找到执行的handler
				if handler == nil {
					// 事务更新
					if err := lgDB.Transaction(
						func(tx *gorm.DB) error {
							if repo.NewTaskRepo().ErrorTaskByID(lgDB, job.TaskID, 0) == 1 {
								if err := repo.TaskLogRepo.UpdateColumn(lgDB, taskLogData.ID, map[string]interface{}{
									"status":     utils.TaskStatusError,
									"error_info": fmt.Sprintf("不存在对应消息的handler%v\n", job.TaskType),
								}); err != nil {
								}
							}
							return nil
						}); err != nil {
					}
				} else {
					// 开始执行
					err := handler(job.TaskID)
					tkInfo, _ := repo.NewTaskRepo().GetByID(lgDB, job.TaskID)
					if err == nil {
						// 执行成功
						if txErr := lgDB.Transaction(
							func(tx *gorm.DB) error {
								if repo.NewTaskRepo().FinishTaskByID(lgDB, job.TaskID, tkInfo.ExecuteTime+1) == 1 {
									if updateErr := repo.TaskLogRepo.UpdateColumn(lgDB, taskLogData.ID,
										map[string]interface{}{
											"status":     utils.TaskStatusFinish,
											"error_info": "",
										}); updateErr != nil {
									}
								}
								return nil
							}); txErr != nil {
						}
					} else {
						// 执行失败
						//还未达到执行次数上限
						if tkInfo.ExecuteTime < utils.CompensationTotal {
							if txErr := lgDB.Transaction(
								func(tx *gorm.DB) error {
									// 更新任务信息中的执行次数
									if updateErr := repo.NewTaskRepo().UpdateColumn(lgDB, job.TaskID,
										"execute_time", tkInfo.ExecuteTime+1); updateErr != nil {
										return updateErr
									}

									if updateErr := repo.TaskLogRepo.UpdateColumn(lgDB, taskLogData.ID,
										map[string]interface{}{
											"status":     utils.TaskStatusError,
											"error_info": "",
										}); updateErr != nil {
									}
									if repo.NewTaskRepo().ResetTaskByID(lgDB, job.TaskID, tkInfo.NodeId) == 1 {
									}
									return nil
								}); txErr != nil {
							}
						} else {
							if txErr := lgDB.Transaction(
								func(tx *gorm.DB) error {
									if repo.NewTaskRepo().ErrorTaskByID(lgDB, job.TaskID, tkInfo.ExecuteTime+1) == 1 {
										if updateErr := repo.TaskLogRepo.UpdateColumn(lgDB, taskLogData.ID,
											map[string]interface{}{
												"status":     utils.TaskStatusError,
												"error_info": err.Error(),
											}); updateErr != nil {
										}
									}
									return nil
								}); txErr != nil {
							}
						}
					}
				}
			case <-taskCtx.Done():
				return
			}
		}
	}()
}

func (w *Worker) Stop() {
	w.Wg.Wait()
}
