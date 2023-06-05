package dispatch

import (
	"fmt"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/base"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/event"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/repo"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/utils"
	"github.com/qinguoyi/ObjectStorageProxy/bootstrap/plugins"
	"sync"
	"time"
)

type Producer struct {
	Wg *sync.WaitGroup
}

func NewProduce() *Producer {
	return &Producer{
		Wg: &sync.WaitGroup{},
	}
}

// Produce 生产者
func (p *Producer) Produce() {
	timer := time.NewTimer(1 * time.Nanosecond)
	ip, err := base.GetOutBoundIP()
	if err != nil {
		panic(err)
	}
	defer timer.Stop()
	defer p.Wg.Done()
	for {
		select {
		case <-timer.C:
			var lgDB = new(plugins.LangGoDB).Use("default").NewDB()
			undoTaskList, _ := repo.NewTaskRepo().FindByStatus(lgDB, utils.TaskStatusUndo)
			for _, i := range undoTaskList {
				// 抢占前处理
				preProcess := event.NewEventsHandler().GetPreProcess(i.TaskType)
				if preProcess != nil {
					if f := preProcess(i.ID); !f {
						continue
					}
				}
				// 抢占任务
				affectRow := repo.NewTaskRepo().PreemptiveTaskByID(lgDB, i.ID, ip)
				if affectRow != 0 {
					JobQueue <- Job{
						TaskID:   i.ID,
						TaskType: i.TaskType,
					}
				}
			}

		case <-taskCtx.Done():
			fmt.Println("任务生产者终止...")
			return
		}

		timer.Reset(500 * time.Millisecond)
	}
}

func (p *Producer) Stop() {
	p.Wg.Wait()
}
