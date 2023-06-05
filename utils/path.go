package utils

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// RootPath 获取项目根目录绝对路径
func RootPath() string {
	var rootDir string

	exePath, err := os.Executable()
	if err != nil {
		panic(err)
	}

	rootDir = filepath.Dir(filepath.Dir(exePath))

	tmpDir := os.TempDir()
	if strings.Contains(exePath, tmpDir) {
		_, filename, _, ok := runtime.Caller(0)
		if ok {
			rootDir = filepath.Dir(filepath.Dir(filepath.Dir(filename)))
		}
	}

	return rootDir
}

// Exists 路径是否存在
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
