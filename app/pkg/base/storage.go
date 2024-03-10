package base

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/qinguoyi/osproxy/app/models"
	"github.com/qinguoyi/osproxy/app/pkg/utils"
	"github.com/qinguoyi/osproxy/bootstrap"
	"github.com/qinguoyi/osproxy/bootstrap/plugins"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var lgLogger *bootstrap.LangGoLogger

func GetExtension(filename string) string {
	ext := filepath.Ext(filename)
	if ext == "" {
		return ""
	}
	return strings.ToLower(ext[1:])
}

// selectBucketBySuffix .
func selectBucketBySuffix(filename string) string {
	suffix := GetExtension(filename)
	if suffix == "" {
		return ""
	}
	switch suffix {
	case "jpg", "jpeg", "png", "gif", "bmp":
		return "image"
	case "mp4", "avi", "wmv", "mpeg":
		return "video"
	case "mp3", "wav", "flac":
		return "audio"
	case "pdf", "doc", "docx", "ppt", "pptx", "xls", "xlsx":
		return "doc"
	case "zip", "rar", "tar", "gz", "7z":
		return "archive"
	default:
		return "unknown"
	}
}

func CheckValid(uidStr, date, expireStr string) (int64, error, string) {
	// check
	uid, err := strconv.ParseInt(uidStr, 10, 64)
	if err != nil {
		return 0, err, fmt.Sprintf("uid参数有误，详情:%s", err)
	}

	loc, _ := time.LoadLocation("Local")
	t, err := time.ParseInLocation("2006-01-02T15:04:05Z", date, loc)
	if err != nil {
		return uid, err, fmt.Sprintf("时间参数转换失败，详情:%s", err)
	}

	expire, err := strconv.ParseInt(expireStr, 10, 64)
	if err != nil {
		return uid, err, fmt.Sprintf("expire参数有误，详情:%s", err)
	}
	now := time.Now().In(loc)
	duration := now.Sub(t)
	if int64(duration.Seconds()) > expire {
		return uid, errors.New("链接时间已过期"), "链接时间已过期"
	}
	return uid, nil, ""
}

// GenUploadSingle .
func GenUploadSingle(filename string, expire int, respChan chan models.GenUploadResp,
	metaDataInfoChan chan models.MetaDataInfo, wg *sync.WaitGroup) {
	defer wg.Done()
	bucket := selectBucketBySuffix(filename)
	uid, err := NewSnowFlake().NextId()
	if err != nil {
		//lgLogger.WithContext(c).Error("雪花算法生成ID失败，详情：", zap.Any("err", err.Error()))
		return
	}
	uidStr := strconv.FormatInt(uid, 10)
	name := filepath.Base(filename)
	name = url.PathEscape(name)
	storageName := fmt.Sprintf("%s.%s", uidStr, GetExtension(filename))
	objectName := fmt.Sprintf("%s/%s", bucket, storageName)

	// 在本地创建uid的目录
	if err := os.MkdirAll(path.Join(utils.LocalStore, uidStr), 0755); err != nil {
		//lgLogger.WithContext(c).Error("创建本地目录失败，详情：", zap.Any("err", err.Error()))
		return
	}

	// 生成加密query
	date := time.Now().Format("2006-01-02T15:04:05Z")
	signature := decode(fmt.Sprintf("%s-%d", date, expire))
	queryString := GenUploadSignature(uidStr, date, expire, signature)
	single := fmt.Sprintf("/api/storage/v0/upload?%s", queryString)
	multi := fmt.Sprintf("/api/storage/v0/upload/multi?%s", queryString)
	merge := fmt.Sprintf("/api/storage/v0/upload/merge?%s", queryString)
	respChan <- models.GenUploadResp{
		Uid: uidStr,
		Url: &models.UrlResult{
			Single: single,
			Multi: &models.MultiUrlResult{
				Merge:  merge,
				Upload: multi,
			},
		},
		Path: filename,
	}
	// 生成DB信息
	now := time.Now()
	metaDataInfoChan <- models.MetaDataInfo{
		UID:         uid,
		Bucket:      bucket,
		Name:        name,
		StorageName: storageName,
		Address:     objectName,
		MultiPart:   false,
		Status:      -1,
		ContentType: "application/octet-stream", //先按照文件后缀占位，后面文件上传会覆盖
		CreatedAt:   &now,
		UpdatedAt:   &now,
	}
	return
}

func GenDownloadSingle(meta models.MetaDataInfo, expire string, respChan chan models.GenDownloadResp,
	wg *sync.WaitGroup) {
	defer wg.Done()
	uid := meta.UID
	bucketName := meta.Bucket
	srcName, err := url.PathUnescape(meta.Name)
	if err != nil {
		fmt.Println("文件名解码失败:", err)
	}
	objectName := meta.StorageName

	// 生成加密query
	date := time.Now().Format("2006-01-02T15:04:05Z")
	signature := decode(fmt.Sprintf("%s-%s-%s-%s", date, expire,
		bucketName, objectName))
	queryString := GenDownloadSignature(uid, srcName, bucketName, objectName, expire, date, signature)
	url := fmt.Sprintf("/api/storage/v0/download?%s", queryString)
	info := models.GenDownloadResp{
		Uid: fmt.Sprintf("%d", uid),
		Url: url,
		Meta: models.MetaInfo{
			SrcName: srcName,
			DstName: objectName,
			Height:  meta.Height,
			Width:   meta.Width,
			Md5:     meta.Md5,
			Size:    fmt.Sprintf("%d", meta.StorageSize),
		},
	}
	respChan <- info
	// 写入redis
	key := fmt.Sprintf("%d-%s", uid, expire)
	lgRedis := new(plugins.LangGoRedis).NewRedis()
	b, err := json.Marshal(info)
	if err != nil {
	}
	lgRedis.SetNX(context.Background(), key, b, 5*60*time.Second)
}

func GetRange(rangeHeader string, size int64) (int64, int64) {
	var start, end int64
	if rangeHeader == "" {
		end = size - 1
	} else {
		split := strings.Split(rangeHeader, "=")
		ranges := strings.Split(split[1], "-")
		start, _ = strconv.ParseInt(ranges[0], 10, 64)
		if ranges[1] != "" {
			end, _ = strconv.ParseInt(ranges[1], 10, 64)
		}
		if end >= size || end == 0 {
			end = size - 1
		}
	}
	return start, end
}
