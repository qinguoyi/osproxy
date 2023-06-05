package base

/*
雪花算法
*/

import (
	"context"
	"errors"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/utils"
	"github.com/qinguoyi/ObjectStorageProxy/bootstrap/plugins"
	"strconv"
	"sync"
	"time"
)

var (
	snowFlake *Snowflake
	once      sync.Once
)

const (
	twepoch            = int64(1417937700000) // Unix纪元时间戳
	workerIdBits       = uint(5)              // 机器ID所占位数
	datacenterBits     = uint(5)              // 数据中心ID所占位数
	maxWorkerId        = int64(-1) ^ (int64(-1) << workerIdBits)
	maxDatacenterId    = int64(-1) ^ (int64(-1) << datacenterBits)
	sequenceBits       = uint(12) // 序列号所占位数
	workerIdShift      = sequenceBits
	datacenterIdShift  = sequenceBits + workerIdBits
	timestampLeftShift = sequenceBits + workerIdBits + datacenterBits
	sequenceMask       = int64(-1) ^ (int64(-1) << sequenceBits)
)

type Snowflake struct {
	mu            sync.Mutex
	lastTimestamp int64
	workerId      int64
	datacenterId  int64
	sequence      int64
}

// InitSnowFlake .
func InitSnowFlake() {
	// get local ip
	ip, err := GetOutBoundIP()
	if err != nil {
		panic(err)
	}

	// get workId from redis
	var workId int64
	ctx := context.Background()
	lgRedis := new(plugins.LangGoRedis).NewRedis()

	ipExist := lgRedis.Exists(ctx, ip).Val()
	if ipExist == 1 {
		curWorkId, err := lgRedis.Get(ctx, ip).Result()
		if err != nil {
			panic(err)
		}
		workId, err = strconv.ParseInt(curWorkId, 10, 64)
		if err != nil {
			panic(err)
		}
	} else {
		newWorkId, err := lgRedis.Incr(ctx, utils.WorkID).Result()
		if err != nil {
			panic(err)
		}
		lgRedis.Set(ctx, ip, newWorkId, -1)
		workId = newWorkId
	}
	once.Do(func() {
		res, err := newSnowFlake(workId, 0)
		if err != nil {
			panic(err)
		}
		snowFlake = res
	})
}

func newSnowFlake(workerId, datacenterId int64) (*Snowflake, error) {
	if workerId < 0 || workerId > maxWorkerId {
		return nil, errors.New("worker id out of range")
	}
	if datacenterId < 0 || datacenterId > maxDatacenterId {
		return nil, errors.New("datacenter id out of range")
	}
	return &Snowflake{
		lastTimestamp: 0,
		workerId:      workerId,
		datacenterId:  datacenterId,
		sequence:      0,
	}, nil
}

// NewSnowFlake .
func NewSnowFlake() *Snowflake {
	if snowFlake == nil {
		once.Do(func() {
			res, err := newSnowFlake(10, 10)
			if err != nil {
				panic(err)
			}
			snowFlake = res
		})
	}
	return snowFlake
}

// NextId .
func (sf *Snowflake) NextId() (int64, error) {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	timestamp := time.Now().UnixNano() / 1000000

	if timestamp < sf.lastTimestamp {
		return 0, errors.New("clock moved backwards")
	}

	if timestamp == sf.lastTimestamp {
		sf.sequence = (sf.sequence + 1) & sequenceMask
		if sf.sequence == 0 {
			// 时钟回拨
			for timestamp <= sf.lastTimestamp {
				timestamp = time.Now().UnixNano() / 1000000
			}
		}
	} else {
		sf.sequence = 0
	}

	sf.lastTimestamp = timestamp
	// 相当于
	id := ((timestamp - twepoch) << timestampLeftShift) | (sf.datacenterId << datacenterIdShift) | (sf.workerId << workerIdShift) | sf.sequence

	return id, nil
}
