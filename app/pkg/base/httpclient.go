package base

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"
)

type Response struct {
	Code  int             `json:"code"`
	Msg   string          `json:"message"`
	Data  json.RawMessage `json:"data"`
	Total int64           `json:"total,omitempty"`
}

var (
	Client *http.Client //HTTPClient
)

type Request struct {
	Url       string
	Body      io.ReadCloser
	HeaderSet map[string]string
	Method    string
	Params    map[string]string
}

func init() {
	Client = &http.Client{
		Timeout: 300 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
			DisableKeepAlives: true,
			Proxy:             http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second, // tcp连接超时时间
				KeepAlive: 60 * time.Second, // 保持长连接的时间
				DualStack: true,
			}).DialContext, // 设置连接的参数
			MaxIdleConns:          100, // 最大空闲连接
			MaxConnsPerHost:       100,
			MaxIdleConnsPerHost:   100,              // 每个host保持的空闲连接数
			ExpectContinueTimeout: 30 * time.Second, // 等待服务第一响应的超时时间
			IdleConnTimeout:       60 * time.Second, // 空闲连接的超时时间
		},
	}
}

// CheckRespStatus 状态检查
func CheckRespStatus(resp *http.Response) (*Response, http.Header, error) {
	bodyBytes, _ := ioutil.ReadAll(resp.Body)
	respRes := Response{}
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		if err := json.Unmarshal(bodyBytes, &respRes); err != nil {
			return nil, nil, err
		}
		return &respRes, resp.Header, nil
	}
	return nil, nil, errors.New(string(bodyBytes))
}

// Ask 建立http请求，返回header信息
func Ask(requester Request) (respStatusCode int, respBytes *Response, respHeader http.Header, err error) {
	request, err := http.NewRequest(requester.Method, requester.Url, requester.Body)
	if err != nil {
		return 401, nil, nil, err
	}
	// header 添加字段,包含token
	if requester.HeaderSet != nil {
		for k, v := range requester.HeaderSet {
			request.Header.Set(k, v)
		}
	}
	// query params
	if requester.Params != nil {
		params := make(url.Values)
		for k, v := range requester.Params {
			params.Add(k, v)
		}
		request.URL.RawQuery = params.Encode()
	}

	resp, err := Client.Do(request)
	if err != nil {
		return 401, nil, nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
		}
	}(resp.Body)

	// 返回的状态码
	respBytes, respHeader, err = CheckRespStatus(resp)
	respStatusCode = resp.StatusCode
	return
}

// CheckFileRespStatus 状态检查
func CheckFileRespStatus(resp *http.Response) (io.ReadCloser, http.Header, error) {
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return resp.Body, resp.Header, nil
	}
	return nil, nil, errors.New("失败")
}

// AskFile 建立http请求，返回header信息
func AskFile(requester Request) (respStatusCode int, respBytes io.ReadCloser, respHeader http.Header, err error) {
	request, err := http.NewRequest(requester.Method, requester.Url, requester.Body)
	if err != nil {
		return 401, nil, nil, err
	}
	// header 添加字段,包含token
	if requester.HeaderSet != nil {
		for k, v := range requester.HeaderSet {
			request.Header.Set(k, v)
		}
	}
	// query params
	if requester.Params != nil {
		params := make(url.Values)
		for k, v := range requester.Params {
			params.Add(k, v)
		}
		request.URL.RawQuery = params.Encode()
	}

	resp, err := Client.Do(request)
	if err != nil {
		return 401, nil, nil, err
	}
	//defer func(Body io.ReadCloser) {
	//	err := Body.Close()
	//	if err != nil {
	//	}
	//}(resp.Body)

	// 返回的状态码
	respBytes, respHeader, err = CheckFileRespStatus(resp)
	respStatusCode = resp.StatusCode
	return
}
