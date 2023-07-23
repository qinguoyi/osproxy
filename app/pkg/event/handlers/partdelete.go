package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qinguoyi/osproxy/app/models"
	"github.com/qinguoyi/osproxy/app/pkg/event"
	"github.com/qinguoyi/osproxy/app/pkg/repo"
	"github.com/qinguoyi/osproxy/app/pkg/storage"
	"github.com/qinguoyi/osproxy/app/pkg/utils"
	"github.com/qinguoyi/osproxy/bootstrap/plugins"
	"os"
	"path"
)

func init() {
	event.NewEventsHandler().RegPreProcess(utils.TaskPartDelete, preProcessPartDelete)
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

	dirName := path.Join(utils.LocalStore, fmt.Sprintf("%d", msg.StorageUid))
	if err := os.RemoveAll(dirName); err != nil {
		return errors.New("删除本地脏数据失败")
	}

	if err := repo.NewMultiPartInfoRepo().Updates(lgDB, msg.StorageUid, map[string]interface{}{
		"status": -1,
	}); err != nil {
		return errors.New("删除数据库分片信息错误")
	}

	sto := storage.NewStorage().Storage
	for _, v := range multiPartInfoList {
		if err := sto.DeleteObject(v.Bucket, v.StorageName); err != nil {
			return errors.New("删除对象存储的脏数据失败")
		}
	}
	return nil
}
