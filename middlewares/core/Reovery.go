package middlewares

import (
	"fmt"
	"net/http"
	"runtime/debug"
	funcs "xiaohuAdmin/function"
	"xiaohuAdmin/global"

	"github.com/gin-gonic/gin"
)

// CustomRecovery 自定义恢复中间件
func CustomRecovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 获取堆栈信息
				stack := debug.Stack()

				// 记录错误日志
				global.ZapLog.Error("系统发生panic",
					global.LogField("error", err),
					global.LogField("path", c.Request.URL.Path),
					global.LogField("method", c.Request.Method),
					global.LogField("ip", c.ClientIP()),
					global.LogField("user_agent", c.Request.UserAgent()),
					global.LogField("stack", string(stack)),
				)

				// 检查请求头，如果是API请求返回JSON，否则返回HTML
				if c.GetHeader("Accept") == "application/json" ||
					c.GetHeader("Content-Type") == "application/json" {
					funcs.JsonRes(c, http.StatusInternalServerError,
						"系统内部错误，请联系管理员",
						map[string]interface{}{
							"error": fmt.Sprintf("%v", err),
							"path":  c.Request.URL.Path,
						})
				} else {
					// 返回HTML错误页面
					c.HTML(http.StatusInternalServerError, "error.html", gin.H{
						"error": fmt.Sprintf("%v", err),
						"path":  c.Request.URL.Path,
					})
				}

				c.Abort()
			}
		}()
		c.Next()
	}
}
