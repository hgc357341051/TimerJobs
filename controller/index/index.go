package index

import (
	funcs "xiaohuAdmin/function"

	"github.com/gin-gonic/gin"
)

// Index 首页接口
// @Summary 首页
// @Description 系统首页
// @Tags 系统
// @Accept json
// @Produce json
// @Success 200 {object} function.JsonData "欢迎信息"
// @Router / [get]
func Index(c *gin.Context) {
	funcs.Ok(c, "小胡专用定时任务系统", nil)
}
