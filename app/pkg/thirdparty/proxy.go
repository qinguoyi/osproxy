package thirdparty

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/base"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

type storageService struct{}

// NewStorageService .
func NewStorageService() *storageService { return &storageService{} }

// Locate .
func (s *storageService) Locate(scheme, ip, port, uid string) (string, error) {
	urlStr := "/api/storage/v0/proxy"
	req := base.Request{
		Url:    fmt.Sprintf("%s://%s:%s%s", scheme, ip, port, urlStr),
		Body:   io.NopCloser(strings.NewReader("")),
		Method: "GET",
		Params: map[string]string{"uid": uid},
	}
	_, data, _, err := base.Ask(req)
	if err != nil {
		return "", err
	}

	return strings.Trim(string(data.Data), "\""), nil
}

// UploadForward .
func (s *storageService) UploadForward(c *gin.Context, scheme, ip, port, uid string, single bool) (int, *base.Response, http.Header, error) {
	var urlStr string
	if single {
		urlStr = fmt.Sprintf("/api/storage/v0/upload/%s", uid)
	} else {
		urlStr = fmt.Sprintf("/api/storage/v0/upload/%s/multi", uid)
	}
	// 获取查询参数
	queryParam := map[string]string{}
	query := c.Request.URL.Query()
	for k, v := range query {
		queryParam[k] = v[0]
	}

	form, err := c.MultipartForm()
	if err != nil {
		return 500, nil, nil, err
	}
	// 创建表单数据
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for key, value := range form.Value {
		for _, v := range value {
			_ = writer.WriteField(key, v)
		}
	}

	// 处理文件数据
	files := form.File["file"]
	for _, file := range files {
		// 打开文件
		src, err := file.Open()
		if err != nil {
			return 500, nil, nil, err
		}
		defer func(src multipart.File) {
			err := src.Close()
			if err != nil {

			}
		}(src)

		// 创建表单数据项
		dst, err := writer.CreateFormFile("file", file.Filename)
		if err != nil {
			return 500, nil, nil, err
		}

		// 将文件内容写入表单数据项
		if _, err = io.Copy(dst, src); err != nil {
			return 500, nil, nil, err
		}
	}
	err = writer.Close()
	if err != nil {
		return 500, nil, nil, err
	}
	req := base.Request{
		Url:  fmt.Sprintf("%s://%s:%s%s", scheme, ip, port, urlStr),
		Body: io.NopCloser(body),
		HeaderSet: map[string]string{
			"Content-Type": writer.FormDataContentType(),
		},
		Method: "PUT",
		Params: queryParam,
	}
	return base.Ask(req)
}

// MergeForward .
func (s *storageService) MergeForward(c *gin.Context, scheme, ip, port, uid string) (int, *base.Response, http.Header, error) {
	urlStr := fmt.Sprintf("/api/storage/v0/upload/%s/merge", uid)
	// 获取查询参数
	queryParam := map[string]string{}
	query := c.Request.URL.Query()
	for k, v := range query {
		queryParam[k] = v[0]
	}

	req := base.Request{
		Url:       fmt.Sprintf("%s://%s:%s%s", scheme, ip, port, urlStr),
		Body:      nil,
		HeaderSet: map[string]string{},
		Method:    "PUT",
		Params:    queryParam,
	}
	return base.Ask(req)
}
