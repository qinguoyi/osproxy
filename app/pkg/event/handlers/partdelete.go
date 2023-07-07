package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qinguoyi/ObjectStorageProxy/app/models"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/event"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/repo"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/storage"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/utils"
	"github.com/qinguoyi/ObjectStorageProxy/bootstrap/plugins"
	"os"
	"path"
)

func init() {
	event.NewEventsHandler().RegPreProcess(utils.TaskPartDelete, preProcessPartMerge)
	event.NewEventsHandler().RegHandler(utils.TaskPartDelete, handlePartDelete)
}

func preProcessPartDelete(i interface{}) bool {
	lgDB := new(plugins.LangGoDB).Use("default").NewDB()

	taskID := i.(int64)
	taskInfo, err := repo.NewTaskRepo().GetByID(lgDB, taskID)
	if err != nil {
		fmt.Printf("任务不存在%v", err)
		return false
	}
	// 反序列extraData
	var msg models.MergeInfo
	if err := json.Unmarshal([]byte(taskInfo.ExtraData), &msg); err != nil {
		fmt.Printf("任务不存在%v", err)
		return false
	}
	return true
}

func handlePartDelete(i interface{}) error {
	lgDB := new(plugins.LangGoDB).Use("default").NewDB()
	var err error
	taskID := i.(int64)
	taskInfo, err := repo.NewTaskRepo().GetByID(lgDB, taskID)
	if err != nil {
		fmt.Printf("任务不存在%v", err)
		return err
	}
	// 反序列extraData
	var msg models.MergeInfo
	if err := json.Unmarshal([]byte(taskInfo.ExtraData), &msg); err != nil {
		return err
	}

	//查询分片信息
	var multiPartInfoList []models.MultiPartInfo
	if err := lgDB.Model(&models.MultiPartInfo{}).Where(
		"storage_uid = ? and status = ?", msg.StorageUid, 1).Order("chunk_num ASC").Find(&multiPartInfoList).Error; err != nil {
		return errors.New("查询分片数据失败")
	}

	//删除本地的脏数据
	sto := storage.NewStorage().Storage
	for _, v := range multiPartInfoList {
		partName := path.Join(utils.LocalStore, fmt.Sprintf("%d", msg.StorageUid), v.PartFileName)
		err = os.RemoveAll(partName)
		if err != nil {
			return errors.New("删除本地脏数据失败")
		}
		err = sto.DeleteObject(v.Bucket, v.StorageName)
		if err != nil {
			return errors.New("删除对象存储的脏数据失败")
		}
	}
	return nil
}
