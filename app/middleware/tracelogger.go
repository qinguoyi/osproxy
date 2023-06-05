package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/qinguoyi/ObjectStorageProxy/bootstrap"
	"go.uber.org/zap"
)

// TraceLog .
type TraceLog struct {
	Logger *bootstrap.LangGoLogger
}

// NewTrace .
func NewTrace(logger *bootstrap.LangGoLogger) *TraceLog {
	return &TraceLog{
		Logger: logger,
	}
}

// Handler .
func (t *TraceLog) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 每个请求生成的请求traceId具有全局唯一性
		traceId := c.GetHeader("request-id")
		if traceId == "" {
			traceId = uuid.New().String()
		}
		t.Logger.NewContext(c, zap.String("traceId", traceId))

		c.Next()
	}
}
