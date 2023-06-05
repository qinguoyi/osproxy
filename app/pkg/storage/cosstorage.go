package storage

import (
	"context"
	"fmt"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/utils"
	"github.com/qinguoyi/ObjectStorageProxy/bootstrap"
	"github.com/tencentyun/cos-go-sdk-v5"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"
)

// CosStorage cos存储
type CosStorage struct {
	Appid     string
	Region    string
	SecretId  string
	SecretKey string
}

// NewCosStorage .
func NewCosStorage() *CosStorage {
	return &CosStorage{
		Appid:     bootstrap.NewConfig("").Cos.Appid,
		Region:    bootstrap.NewConfig("").Cos.Region,
		SecretId:  bootstrap.NewConfig("").Cos.SecretId,
		SecretKey: bootstrap.NewConfig("").Cos.SecretKey,
	}
}

// MakeBucket .
func (s *CosStorage) MakeBucket(bucketName string) error {
	u, _ := url.Parse(fmt.Sprintf("https://%s-%s.cos.%s.myqcloud.com", bucketName, s.Appid, s.Region))
	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  s.SecretId,
			SecretKey: s.SecretKey,
		},
	})
	ok, err := client.Bucket.IsExist(context.Background())
	if err == nil && ok {
		return nil
	} else if err != nil {
		return err
	} else {
		// 存储桶不存在
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
		defer cancel()
		resp, err := client.Bucket.Put(ctx, nil)
		if err != nil && resp.StatusCode != 409 {
			return err
		}
	}

	return nil
}

// GetObject .
func (s *CosStorage) GetObject(bucketName, objectName string, offset, length int64) ([]byte, error) {
	u, _ := url.Parse(fmt.Sprintf("https://%s-%s.cos.%s.myqcloud.com", bucketName, s.Appid, s.Region))
	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  s.SecretId,
			SecretKey: s.SecretKey,
		},
	})
	resp, err := client.Object.Get(context.Background(), objectName, &cos.ObjectGetOptions{
		Range: fmt.Sprintf("bytes=%d-%d", offset, offset+length-1),
	})
	// 注意，这里需要关闭流，否则会造成资源占用，cos会hang住
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	return content, err
}

// PutObject .
func (s *CosStorage) PutObject(bucketName, objectName, filePath, contentType string) error {
	u, _ := url.Parse(fmt.Sprintf("https://%s-%s.cos.%s.myqcloud.com", bucketName, s.Appid, s.Region))
	b := &cos.BaseURL{BucketURL: u}
	client := cos.NewClient(b, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  s.SecretId,
			SecretKey: s.SecretKey,
		},
	})
	_, _, err := client.Object.Upload(context.Background(), objectName, filePath, &cos.MultiUploadOptions{
		OptIni: &cos.InitiateMultipartUploadOptions{
			ObjectPutHeaderOptions: &cos.ObjectPutHeaderOptions{
				ContentType: contentType,
			},
		},
		ThreadPoolSize: utils.S3StoragePutThreadNum,
	})
	if err != nil {
		return err
	}
	return nil
}
