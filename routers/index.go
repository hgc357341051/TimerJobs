package routers

import (
	indexApp "xiaohuAdmin/controller/index"
	_ "xiaohuAdmin/docs" // 导入docs包以初始化Swagger
	funcs "xiaohuAdmin/function"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// IndexInit 注册首页、Swagger、404/405等通用路由
func IndexInit(r *gin.Engine) {
	indexRouters := r.Group("/")
	{
		// 首页接口
		indexRouters.GET("/", indexApp.Index)
	}

	// Swagger文档路由
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// 405方法不允许处理
	r.NoMethod(func(c *gin.Context) {
		funcs.No(c, "请求方法不存在", nil)
	})
	// 404路径不存在处理
	r.NoRoute(func(c *gin.Context) {
		funcs.No(c, "请求路径不存在", nil)
	})
}
