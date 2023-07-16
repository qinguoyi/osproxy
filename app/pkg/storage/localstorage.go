package storage

import (
	"fmt"
	"github.com/qinguoyi/osproxy/app/pkg/utils"
	"io"
	"log"
	"os"
	"path"
)

// LocalStorage 本地存储
type LocalStorage struct {
	RootPath string
}

func NewLocalStorage() *LocalStorage {
	return &LocalStorage{
		RootPath: utils.LocalStore,
	}
}

// MakeBucket .
func (s *LocalStorage) MakeBucket(bucketName string) error {
	dirName := path.Join(s.RootPath, bucketName)
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		if err := os.MkdirAll(dirName, 0755); err != nil {
			//lgLogger.WithContext(c).Error("创建本地目录失败，详情：", zap.Any("err", err.Error()))
			return err
		}
	}
	return nil
}

// GetObject .
func (s *LocalStorage) GetObject(bucketName, objectName string, offset, length int64) ([]byte, error) {
	objectPath := path.Join(s.RootPath, bucketName, objectName)
	file, err := os.Open(objectPath)
	if err != nil {
		fmt.Println("Failed to open file:", err)
		return nil, err
	}
	defer file.Close()
	_, err = file.Seek(offset, io.SeekStart)
	if err != nil {
		fmt.Println("Error:", err)
		return nil, err
	}
	buffer := make([]byte, length)
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		fmt.Println("Error:", err)
		return nil, err
	}
	return buffer, nil
}

// PutObject .
func (s *LocalStorage) PutObject(bucketName, objectName, filePath, contentType string) error {
	// copy 数据到 具体的目录
	// 打开源文件
	sourceFile, err := os.Open(filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer sourceFile.Close()

	objectPath := path.Join(s.RootPath, bucketName, objectName)
	file, err := os.Create(objectPath)
	if err != nil {
		fmt.Println("Failed to create file:", err)
		return err
	}
	defer file.Close()

	// 复制文件内容
	_, err = io.Copy(file, sourceFile)
	if err != nil {
		fmt.Println("Failed to copy file:", err)
		return err
	}
	return nil
}

func (s *LocalStorage) DeleteObject(bucketName, objectName string) error {
	objectPath := path.Join(s.RootPath, bucketName, objectName)
	err := os.RemoveAll(objectPath)
	return err
}
