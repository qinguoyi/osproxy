package dispatch

import (
	"context"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/base"
	_ "github.com/qinguoyi/ObjectStorageProxy/app/pkg/event/handlers" // 为了执行handlers包里的init 自动注册
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/repo"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/utils"
	"github.com/qinguoyi/ObjectStorageProxy/bootstrap/plugins"
)

var (
	taskCtx, taskCancel = context.WithCancel(context.Background())
	JobQueue            = make(chan Job, utils.MaxQueue)
)

// RunTask 启动任务
func RunTask() (*Producer, []Worker) {
	// 启动生产者
	p := NewProduce()
	p.Wg.Add(1)
	go p.Produce()

	// 启动消费者
	var consumers []Worker
	for i := 0; i < utils.MaxWorker; i++ {
		worker := NewWorker()
		consumers = append(consumers, *worker)
		worker.Wg.Add(1)
		worker.Start()
	}
	return p, consumers
}

func StopTask(p *Producer, consumers []Worker) {
	ip, err := base.GetOutBoundIP()
	if err != nil {
		panic(err)
	}

	// 发送任务终止信号
	taskCancel()

	// 等待生产消费者任务终止
	p.Stop()
	for i := 0; i < utils.MaxWorker; i++ {
		consumers[i].Stop()
	}

	// 清理现场，这些是指那些在队列中的任务，但还没有被消费，需要重置
	var lgDB = new(plugins.LangGoDB).Use("default").NewDB()
	runningTask, _ := repo.NewTaskRepo().FindByStatus(lgDB, utils.TaskStatusRunning)
	for _, i := range runningTask {
		_ = repo.NewTaskRepo().ResetTaskByID(lgDB, i.ID, ip)
	}
}
