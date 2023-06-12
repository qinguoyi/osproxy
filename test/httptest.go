package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/qinguoyi/ObjectStorageProxy/app/models"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/base"
	"io"
	"math"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func minH(a, b int64) int64 {
	if a <= b {
		return a
	} else {
		return b
	}
}

func main() {
	// 基础信息
	baseUrl := "http://127.0.0.1:8888"
	uploadFilePath := "./xxx.jpg"
	uploadFile := filepath.Base(uploadFilePath)

	// ##################### 获取上传连接 ###################
	fmt.Println("获取上传连接")
	urlStr := "/api/storage/v0/link/upload"
	body := map[string]interface{}{
		"filePath": []string{fmt.Sprintf("%s", uploadFile)},
		"expire":   86400,
	}
	jsonBytes, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	req := base.Request{
		Url:    fmt.Sprintf("%s%s", baseUrl, urlStr),
		Body:   io.NopCloser(strings.NewReader(string(jsonBytes))),
		Method: "POST",
		Params: map[string]string{},
	}
	_, data, _, err := base.Ask(req)
	if err != nil {
		panic(err)
	}
	var uploadLink []*models.GenUploadResp
	if err := json.Unmarshal(data.Data, &uploadLink); err != nil {
		panic(err)
	}

	// ##################### 上传文件 ######################
	// +++++++ 单文件 ++++++

	filePath := uploadFilePath
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Failed to open file:", err)
		return
	}
	defer file.Close()
	md5Str, _ := base.CalculateFileMd5(filePath)

	fileInfo, _ := os.Stat(filePath)
	fileSize := fileInfo.Size()
	fmt.Println(fileSize)
	if fileSize <= 1024*1024*1 {
		fmt.Println("单文件上传")
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)

		// 打开文件
		defer func(srcFile multipart.File) {
			err := srcFile.Close()
			if err != nil {

			}
		}(file)

		// 创建表单数据项
		dst, err := writer.CreateFormFile("file", uploadFile)
		if err != nil {
			panic(err)
		}

		// 将文件内容写入表单数据项
		if _, err = io.Copy(dst, file); err != nil {
			panic(err)
		}
		err = writer.Close()
		if err != nil {
			panic(err)
		}
		u, err := url.Parse(uploadLink[0].Url.Single)
		if err != nil {
			panic(err)
		}
		query := u.Query()
		uidStr := base.Query(query, "uid")
		date := base.Query(query, "date")
		expireStr := base.Query(query, "expire")
		signature := base.Query(query, "signature")
		req := base.Request{
			Url:  fmt.Sprintf("%s%s", baseUrl, uploadLink[0].Url.Single),
			Body: io.NopCloser(body),
			HeaderSet: map[string]string{
				"Content-Type": writer.FormDataContentType(),
			},
			Method: "PUT",
			Params: map[string]string{"md5": md5Str, "uid": uidStr,
				"date": date, "expire": expireStr, "signature": signature},
		}
		_, _, _, err = base.Ask(req)
		if err != nil {
			panic(err)
		}
	} else {
		// +++++++ 多文件 ++++++
		// 分片上传
		fmt.Println("多文件上传")
		chunkSize := 1024.0 * 1024
		currentChunk := int64(1)
		totalChunk := int64(math.Ceil(float64(fileSize) / chunkSize))
		var wg sync.WaitGroup
		ch := make(chan struct{}, 5)
		for currentChunk <= totalChunk {

			start := (currentChunk - 1) * int64(chunkSize)
			end := minH(fileSize, start+int64(chunkSize))
			buffer := make([]byte, end-start)
			// 循环读取，会自动偏移
			_, err := file.Read(buffer)
			if err != nil && err != io.EOF {
				fmt.Println("读取文件长度失败", err)
				break
			}
			//fmt.Println("当前read长度", n)
			md5Part, _ := base.CalculateByteMd5(buffer)

			// 多协程上传
			ch <- struct{}{}
			wg.Add(1)
			go func(data []byte, md5V string, chunkNum int64, wg *sync.WaitGroup) {
				defer wg.Done()
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				// 创建表单数据项
				dst, err := writer.CreateFormFile("file",
					fmt.Sprintf("%s%d", uploadLink[0].Uid, chunkNum))
				if err != nil {
					panic(err)
				}

				// 将文件内容写入表单数据项
				if _, err = io.Copy(dst, bytes.NewReader(data)); err != nil {
					panic(err)
				}
				err = writer.Close()
				if err != nil {
					panic(err)
				}

				u, err := url.Parse(uploadLink[0].Url.Multi.Upload)
				if err != nil {
					panic(err)
				}
				query := u.Query()
				uidStr := base.Query(query, "uid")
				date := base.Query(query, "date")
				expireStr := base.Query(query, "expire")
				signature := base.Query(query, "signature")
				req := base.Request{
					Url:  fmt.Sprintf("%s%s", baseUrl, uploadLink[0].Url.Multi.Upload),
					Body: io.NopCloser(body),
					HeaderSet: map[string]string{
						"Content-Type": writer.FormDataContentType(),
					},
					Method: "PUT",
					Params: map[string]string{"uid": uidStr, "date": date, "expire": expireStr, "signature": signature,
						"md5": md5V, "chunkNum": fmt.Sprintf("%d", chunkNum)},
				}
				code, _, _, err := base.Ask(req)
				if err != nil {
					fmt.Println(code)
					fmt.Println(err)
				}

				<-ch
			}(buffer, md5Part, currentChunk, &wg)
			currentChunk += 1
		}
		wg.Wait()

		// 合并
		u, err := url.Parse(uploadLink[0].Url.Multi.Merge)
		if err != nil {
			panic(err)
		}
		query := u.Query()
		uidStr := base.Query(query, "uid")
		date := base.Query(query, "date")
		expireStr := base.Query(query, "expire")
		signature := base.Query(query, "signature")
		req := base.Request{
			Url:       fmt.Sprintf("%s%s", baseUrl, uploadLink[0].Url.Multi.Merge),
			Body:      io.NopCloser(strings.NewReader("")),
			HeaderSet: map[string]string{},
			Method:    "PUT",
			Params: map[string]string{"uid": uidStr, "date": date, "expire": expireStr, "signature": signature,
				"md5": md5Str, "num": fmt.Sprintf("%d", totalChunk), "size": fmt.Sprintf("%d", fileSize)},
		}
		_, _, _, err = base.Ask(req)
		if err != nil {
			panic(err)
		}
	}

	// ##################### 获取下载链接 ###################
	fmt.Println("获取下载链接")

	urlStr = "/api/storage/v0/link/download"
	body = map[string]interface{}{
		"uid":    []string{uploadLink[0].Uid},
		"expire": 86400,
	}
	jsonBytes, err = json.Marshal(body)
	if err != nil {
		panic(err)
	}
	req = base.Request{
		Url:    fmt.Sprintf("%s%s", baseUrl, urlStr),
		Body:   io.NopCloser(strings.NewReader(string(jsonBytes))),
		Method: "POST",
		Params: map[string]string{},
	}
	_, data, _, err = base.Ask(req)
	if err != nil {
		panic(err)
	}
	var downloadLink []*models.GenDownloadResp
	if err := json.Unmarshal(data.Data, &downloadLink); err != nil {
		panic(err)
	}

	// ##################### 下载文件 ######################
	fmt.Println("下载文件")
	downloadUrl := downloadLink[0].Url
	u, err := url.Parse(downloadUrl)
	if err != nil {
		panic(err)
	}
	query := u.Query()
	uidStr := base.Query(query, "uid")
	name := base.Query(query, "name")
	date := base.Query(query, "date")
	expireStr := base.Query(query, "expire")
	signature := base.Query(query, "signature")
	bucketName := base.Query(query, "bucket")
	objectName := base.Query(query, "object")

	fileName := fmt.Sprintf("%d", time.Now().Unix())
	dourl := fmt.Sprintf("%s%s", baseUrl, downloadUrl)
	fmt.Printf("下载链接为：%s", dourl)
	// Create the file
	out, err := os.Create(fileName)
	if err != nil {
		panic(err)
	}

	req = base.Request{
		Url:       dourl,
		Body:      io.NopCloser(strings.NewReader("")),
		HeaderSet: map[string]string{},
		Method:    "GET",
		Params: map[string]string{"uid": uidStr, "name": name, "date": date, "expire": expireStr, "signature": signature,
			"md5": md5Str, "bucket": bucketName, "object": objectName},
	}
	_, bodyData, _, err := base.AskFile(req)
	if err != nil {
		panic(err)
	}
	defer bodyData.Close()
	_, err = io.Copy(out, bodyData)
	if err != nil {
		panic(err)
	}

	// 计算md5
	md5New, _ := base.CalculateFileMd5(fileName)
	if md5New == md5Str {
		fmt.Println("测试成功.")
	} else {
		fmt.Println("测试失败", md5New, md5Str)
	}
	_ = out.Close()
	_ = os.Remove(fileName)
}
