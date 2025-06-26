package routers

import (
	AdminApp "xiaohuAdmin/controller/admins"

	"github.com/gin-gonic/gin"
)

// AdminsInit 注册管理员相关路由
// 包含登录、注册、资料、状态、列表等接口
func AdminsInit(r *gin.Engine) {
	AdminRouters := r.Group("/admin")
	{
		AdminController := AdminApp.AdminController{}

		// 登录接口
		AdminRouters.POST("/login", AdminController.Login)
		// 注册接口
		AdminRouters.POST("/register", AdminController.Register)
		// 获取当前用户信息
		AdminRouters.GET("/profile", AdminController.GetProfile)
		// 更新当前用户信息
		AdminRouters.POST("/profile", AdminController.UpdateProfile)
		// 管理员列表
		AdminRouters.GET("/list", AdminController.GetAdminList)
		// 修改管理员状态
		AdminRouters.POST("/status", AdminController.UpdateAdminStatus)
		// 删除管理员
		AdminRouters.POST("/delete", AdminController.DeleteAdmin)
	}
}
