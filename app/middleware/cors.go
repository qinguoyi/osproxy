package middleware

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

/*
设置跨域
*/

// Cors _
type Cors struct {
}

// NewCors _
func NewCors() *Cors {
	return &Cors{}
}

// Handler _
func (c *Cors) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "*")
		c.Header("Access-Control-Allow-Methods", "*")
		c.Header("Access-Control-Expose-Headers", "*")
		c.Header("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
