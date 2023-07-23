package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qinguoyi/osproxy/app/models"
	"github.com/qinguoyi/osproxy/app/pkg/base"
	"github.com/qinguoyi/osproxy/app/pkg/event"
	"github.com/qinguoyi/osproxy/app/pkg/repo"
	"github.com/qinguoyi/osproxy/app/pkg/storage"
	"github.com/qinguoyi/osproxy/app/pkg/utils"
	"github.com/qinguoyi/osproxy/bootstrap/plugins"
	"io"
	"os"
	"path"
	"time"
)

func init() {
	event.NewEventsHandler().RegPreProcess(utils.TaskPartMerge, preProcessPartMerge)
	event.NewEventsHandler().RegHandler(utils.TaskPartMerge, handlePartMerge)
}

func preProcessPartMerge(i interface{}) bool {
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
	dirName := path.Join(utils.LocalStore, fmt.Sprintf("%d", msg.StorageUid))
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		return false
	} else {
		return true
	}
}

func handlePartMerge(i interface{}) error {
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

	// 按顺序合并分片
	var multiPartInfoList []models.MultiPartInfo
	if err := lgDB.Model(&models.MultiPartInfo{}).Where(
		"storage_uid = ? and status = ?", msg.StorageUid, 1).Order("chunk_num ASC").Find(&multiPartInfoList).Error; err != nil {
		return errors.New("查询分片数据失败")
	}
	if msg.ChunkSum != int64(len(multiPartInfoList)) {
		return errors.New("分片数量和整体数量不一致")
	}
	// 在本地
	metaData, err := repo.NewMetaDataInfoRepo().GetByUid(lgDB, msg.StorageUid)
	if err != nil {
		return errors.New("当前上传链接无效，uid不存在")
	}

	fileName := path.Join(utils.LocalStore, fmt.Sprintf("%d", msg.StorageUid), metaData.StorageName)
	out, err := os.Create(fileName)
	if err != nil {
		return errors.New("本地创建文件失败")
	}

	for _, i := range multiPartInfoList {
		partName := path.Join(utils.LocalStore, fmt.Sprintf("%d", msg.StorageUid), i.PartFileName)
		src, err := os.Open(partName)
		if err != nil {
			return errors.New("本地打开分片文件失败")
		}
		if _, err = io.Copy(out, src); err != nil {
			return errors.New("分片文件合并成大文件失败")
		}
		_ = src.Close()
	}

	// 校验md5
	md5Str, err := base.CalculateFileMd5(fileName)
	if err != nil {
		return errors.New(fmt.Sprintf("生成md5失败，详情%s", err.Error()))
	}
	if md5Str != metaData.Md5 {
		return errors.New(fmt.Sprintf("校验md5失败，计算结果:%s, 参数:%s", md5Str, metaData.Md5))
	}
	//判断是否上传过，md5
	resumeInfo, err := repo.NewMetaDataInfoRepo().GetResumeByMd5(lgDB, []string{md5Str})
	if err != nil {
		return err
	}
	if len(resumeInfo) != 0 {
		now := time.Now()
		if err := repo.NewMetaDataInfoRepo().Updates(lgDB, metaData.UID, map[string]interface{}{
			"bucket":       resumeInfo[0].Bucket,
			"storage_name": resumeInfo[0].StorageName,
			"address":      resumeInfo[0].Address,
			"multi_part":   false,
			"updated_at":   &now,
			"content_type": resumeInfo[0].ContentType,
		}); err != nil {
			return errors.New("上传完更新数据失败")
		}
		_ = out.Close()
		fileDir := path.Join(utils.LocalStore, fmt.Sprintf("%d", msg.StorageUid))
		_ = os.RemoveAll(fileDir)
		// 更新数据 删除redis
		lgRedis := new(plugins.LangGoRedis).NewRedis()
		lgRedis.Del(context.Background(), fmt.Sprintf("%d-meta", metaData.UID))
		return nil
	}
	// 上传到minio
	contentType, err := base.DetectContentType(fileName)
	if err != nil {
		return errors.New("判断文件content-type失败")
	}
	if err := storage.NewStorage().Storage.PutObject(
		metaData.Bucket, metaData.StorageName, fileName, contentType); err != nil {
		return errors.New("上传到minio失败")
	}

	// 更新元数据
	now := time.Now()
	if err := repo.NewMetaDataInfoRepo().Updates(lgDB, metaData.UID, map[string]interface{}{
		"multi_part":   false,
		"updated_at":   &now,
		"content_type": contentType,
	}); err != nil {
		return errors.New("上传完更新数据失败")
	}
	// 更新数据 删除redis
	lgRedis := new(plugins.LangGoRedis).NewRedis()
	lgRedis.Del(context.Background(), fmt.Sprintf("%d-meta", metaData.UID))
	_ = out.Close()
	fileDir := path.Join(utils.LocalStore, fmt.Sprintf("%d", msg.StorageUid))
	_ = os.RemoveAll(fileDir)
	return nil
}
