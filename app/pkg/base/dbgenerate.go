package base

/*
数据库生成uid
*/

import (
	"errors"
	"github.com/qinguoyi/osproxy/app/pkg/repo"
	"github.com/qinguoyi/osproxy/bootstrap/plugins"
	"gorm.io/gorm"
	"time"
)

// Uid 每个业务独立的Uid发号器
type Uid struct {
	businessId string
	ch         chan int64
	min, max   int64
}

// NewUid 生成新的发号器
func NewUid(bizId string) (*Uid, error) {
	percent := 0.5
	// 获取对应业务的step
	lgDB := new(plugins.LangGoDB).Use("default").NewDB()
	uidInfo, err := repo.NewUidRepo().GetByBusinessID(lgDB, bizId)
	if err != nil {
		return nil, err
	}
	reserveLen := int64(float64(uidInfo.Step) * percent)
	lid := Uid{
		businessId: bizId,
		ch:         make(chan int64, reserveLen),
	}
	// 启动协程
	go lid.producer(bizId)
	return &lid, nil
}

// NextId 获取
func (u *Uid) NextId() (int64, error) {
	select {
	case <-time.After(time.Second):
		return 0, errors.New("get uid timeout")
	case uid := <-u.ch:
		return uid, nil
	}
}

// producer 生产者
func (u *Uid) producer(bizId string) {
	err := u.reload(bizId)
	if err != nil {
		return
	}

	// 一直往ch中增加数据，如果ch满了就会阻塞，如果没满就会继续加，如果min>max了，就去db获取数据；相当于ch缓存了len的数据，不会等到号段耗尽采取拿
	for {
		if u.min >= u.max {
			err := u.reload(bizId)
			if err != nil {
				return
			}
		}
		u.min++
		u.ch <- u.min
	}
}

// reload 从db中获取号段
func (u *Uid) reload(bizId string) error {
	for {
		err := u.getData(bizId)
		if err != nil {
			return err
		}
		time.Sleep(time.Second)
	}
}

// getData 获取数据，这里事务用于分布式部署生成，每个生成器使用独立的号段来生成，服务重启直接取新号段，不管之前的号码消费完没
func (u *Uid) getData(bizId string) error {
	var maxId int64
	var step int64
	lgDB := new(plugins.LangGoDB).Use("default").NewDB()
	err := lgDB.Transaction(
		func(tx *gorm.DB) error {
			// 获取业务ID对应的步进数据
			uidInfo, err := repo.NewUidRepo().GetByBusinessID(lgDB, bizId)
			if err != nil {
				return err
			}
			maxId, step = uidInfo.MaxId, uidInfo.Step
			// 更新号段
			if err = repo.NewUidRepo().Updates(tx, bizId, map[string]interface{}{
				"max_id": maxId + step,
			}); err != nil {
				return err
			}
			return nil
		})
	if err != nil {
		return err
	}
	u.min = maxId
	u.max = maxId + step
	return nil
}
