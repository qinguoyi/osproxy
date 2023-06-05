package middleware

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/qinguoyi/ObjectStorageProxy/bootstrap"
	"go.uber.org/zap"
	"time"
)

/*
打印请求和响应数据
*/

type RequestLog struct {
	Logger *bootstrap.LangGoLogger
}

func NewRequestLog(logger *bootstrap.LangGoLogger) *RequestLog {
	return &RequestLog{
		Logger: logger,
	}
}

// CustomResponseWriter _
type CustomResponseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write _
func (w CustomResponseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// WriteString _
func (w CustomResponseWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

// Handler request response日志打印 接管gin的默认日志
func (r *RequestLog) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 打印请求 日志打印推荐用zap的方法替换fmt.Sprintf TODO：当有文件传输时，不打印文件内容，过滤或者打印文件名
		//s, _ := ioutil.ReadAll(c.Request.Body) // 这样读取一次后，body就清空了
		//requestDump, err := httputil.DumpRequest(c.Request, true)
		//if err != nil {
		//	r.Logger.WithContext(c).Error("打印请求日志失败，详情", zap.String("err: ", err.Error()))
		//}
		r.Logger.WithContext(c).Info("RequestInfo",
			zap.String("content-type", c.ContentType()),
			zap.String("Ip", c.ClientIP()),
			zap.String("Method", c.Request.Method),
			zap.String("URL", c.Request.URL.Path),
			zap.String("Query", c.Request.URL.RawQuery),
			zap.String("Header", fmt.Sprintf("user-id: %s, organization-id: %s, request-id: %s, write: %s",
				c.Request.Header.Get("user-id"), c.Request.Header.Get("organization-id"),
				c.Request.Header.Get("request-id"), c.Request.Header.Get("write"))),
			//zap.String("body", string(requestDump)),
		)

		// 处理业务
		blw := &CustomResponseWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw
		c.Next()
		cost := time.Since(start)

		// 打印响应
		r.Logger.WithContext(c).Info("ResponseInfo",
			zap.String("Path", c.Request.URL.Path),
			zap.Int("Status", c.Writer.Status()),
			zap.Duration("Cost", cost),
			//zap.String("Data", blw.body.String()),
		)
	}
}
