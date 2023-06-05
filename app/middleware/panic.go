package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/qinguoyi/ObjectStorageProxy/app/pkg/web"
	"github.com/qinguoyi/ObjectStorageProxy/bootstrap"
	"go.uber.org/zap"
	"net/http"
	"runtime/debug"
)

type PanicRecover struct {
	Logger *bootstrap.LangGoLogger
}

// NewPanicRecover _
func NewPanicRecover(logger *bootstrap.LangGoLogger) *PanicRecover {
	return &PanicRecover{
		Logger: logger,
	}
}

// Handler _
func (p *PanicRecover) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				p.Logger.NewContext(c, zap.String("recovered from panic%s", fmt.Sprintf("%v", err)))
				debug.PrintStack()
				c.AbortWithStatusJSON(http.StatusInternalServerError, web.Response{
					Message: fmt.Sprintf("%v", err),
					Data:    "",
				})
			}
		}()

		c.Next()
	}
}
