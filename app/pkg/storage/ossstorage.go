package storage

import (
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/utils"
	"github.com/qinguoyi/ObjectStorageProxy/bootstrap/plugins"
	"io/ioutil"
	"os"
)

// OssStorage oss存储
type OssStorage struct {
	client *oss.Client
}

// NewOssStorage .
func NewOssStorage() *OssStorage {
	client := new(plugins.LangGoOss).NewOss()
	return &OssStorage{
		client: client,
	}
}

// MakeBucket .
func (s *OssStorage) MakeBucket(bucketName string) error {
	isExist, err := s.client.IsBucketExist(bucketName)
	if err != nil {
		panic(err)
	}
	if isExist {
		return nil
	}
	err = s.client.CreateBucket(bucketName)
	if err != nil {
		panic(err)
	}
	return nil
}

// GetObject .
func (s *OssStorage) GetObject(bucketName, objectName string, offset, length int64) ([]byte, error) {
	bucket, err := s.client.Bucket(bucketName)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(-1)
	}
	body, err := bucket.GetObject(objectName, oss.Range(offset, offset+length-1))
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(-1)
	}
	defer body.Close()
	content, err := ioutil.ReadAll(body)
	return content, err
}

// PutObject .
func (s *OssStorage) PutObject(bucketName, objectName, filePath, contentType string) error {
	bucket, err := s.client.Bucket(bucketName)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(-1)
	}
	err = bucket.UploadFile(objectName, filePath,
		1024*1024,
		oss.Routines(utils.S3StoragePutThreadNum),
		oss.ContentType(contentType))
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(-1)
	}
	return nil
}
