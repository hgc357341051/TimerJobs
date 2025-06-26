package middlewares

import (
	"runtime"
	funcs "xiaohuAdmin/function"

	"github.com/gin-gonic/gin"
)

func MemoryGuard(max_MB int) gin.HandlerFunc {
	return func(c *gin.Context) {

		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		usedMB := m.Alloc / 1024 / 1024

		if int(usedMB) >= max_MB {

			funcs.JsonRes(c, 503, "Service Unavailable - Memory limit exceeded (%dMB used / %dMB allowed)", nil)

			c.Abort()
			return
		}

		c.Next()
	}
}
