package storage

import (
	"context"
	"github.com/minio/minio-go/v7"
	"github.com/qinguoyi/osproxy/app/pkg/utils"
	"github.com/qinguoyi/osproxy/bootstrap/plugins"
	"io"
)

// MinIOStorage minio存储
type MinIOStorage struct {
	client *minio.Client
}

// NewMinIOStorage .
func NewMinIOStorage() *MinIOStorage {
	client := new(plugins.LangGoMinio).NewMinio()
	return &MinIOStorage{
		client: client,
	}
}

// MakeBucket .
func (s *MinIOStorage) MakeBucket(bucketName string) error {
	ctx := context.Background()
	isExist, err := s.client.BucketExists(ctx, bucketName)
	if err != nil {
		panic("")
	}
	if isExist {
		return nil
	}
	return s.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: "cn-north-1"})
}

// BucketExists .
func (s *MinIOStorage) BucketExists(bucketName string) (bool, error) {
	ctx := context.Background()
	return s.client.BucketExists(ctx, bucketName)
}

// GetObject .
func (s *MinIOStorage) GetObject(bucketName, objectName string, offset, length int64) ([]byte, error) {
	ctx := context.Background()
	obj, err := s.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	// 注意，这里需要关闭流，否则会造成资源占用，minio会hang住
	defer func(obj *minio.Object) {
		err := obj.Close()
		if err != nil {
		}
	}(obj)
	_, err = obj.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, err
	}
	data := make([]byte, length)
	n, err := obj.Read(data) // 这里有read，就需要close
	return data[:n], err
}

// PutObject .
func (s *MinIOStorage) PutObject(bucketName, objectName, filePath, contentType string) error {
	ctx := context.Background()
	_, err := s.client.FPutObject(ctx, bucketName, objectName, filePath,
		minio.PutObjectOptions{ContentType: contentType, NumThreads: utils.S3StoragePutThreadNum})
	return err
}

// StatObject .
func (s *MinIOStorage) StatObject(bucketName, objectName string) (int64, error) {
	ctx := context.Background()
	objectInfo, err := s.client.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return 0, err
	}
	return objectInfo.Size, nil
}

func (s *MinIOStorage) DeleteObject(bucketName, objectName string) error {

	ctx := context.Background()
	err := s.client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	return err
}
