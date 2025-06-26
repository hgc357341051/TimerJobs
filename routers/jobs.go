package routers

import (
	JobsApp "xiaohuAdmin/controller/jobs" //jobs模块
	"xiaohuAdmin/global"
	"xiaohuAdmin/middlewares"

	"github.com/gin-gonic/gin"
)

// InitGlobal 注册全局路由，包含管理员、首页、任务等所有模块
func InitGlobal(r *gin.Engine) {
	AdminsInit(r)
	IndexInit(r)
	JobsInit(r)
}

// JobsInit 注册 jobs 相关路由及中间件
// 只在 jobs 路由分组上添加 IP 控制中间件
func JobsInit(r *gin.Engine) {
	JobsRouters := r.Group("/jobs")
	// 只在jobs模块上添加IP控制中间件
	JobsRouters.Use(middlewares.IPControl())
	{
		JobsController := JobsApp.Index{}

		// 任务管理接口
		JobsRouters.POST("/add", JobsController.AddJob)
		JobsRouters.POST("/stop", JobsController.StopJob)
		JobsRouters.POST("/del", JobsController.DeleteJob)
		JobsRouters.GET("/list", JobsController.JobList)
		JobsRouters.POST("/edit", JobsController.EditJob)
		JobsRouters.POST("/run", JobsController.JobRun)
		JobsRouters.POST("/stopAll", JobsController.JobStopAll)
		JobsRouters.POST("/runAll", JobsController.JobRunAll)
		JobsRouters.POST("/restart", JobsController.JobRestart)
		JobsRouters.GET("/read", JobsController.JobInfo)
		JobsRouters.POST("/checkJob", JobsController.CalibrateJobList)
		JobsRouters.GET("/scheduler", JobsController.GetSchedulerTasks)
		JobsRouters.GET("/functions", JobsController.GetFunctions)

		// 日志管理接口
		JobsRouters.GET("/zapLogs", JobsController.ZapLogs)
		JobsRouters.GET("/switchState", JobsController.LogSwitchState)
		JobsRouters.GET("/jobState", JobsController.JobState)
		JobsRouters.POST("/logs", JobsController.JobLogs)

		// IP控制管理接口
		JobsRouters.GET("/ip-control/status", JobsController.GetIPControlStatus)
		JobsRouters.POST("/ip-control/whitelist/add", JobsController.AddToWhitelist)
		JobsRouters.POST("/ip-control/whitelist/remove", JobsController.RemoveFromWhitelist)
		JobsRouters.POST("/ip-control/blacklist/add", JobsController.AddToBlacklist)
		JobsRouters.POST("/ip-control/blacklist/remove", JobsController.RemoveFromBlacklist)

		// 系统管理接口
		JobsRouters.POST("/reload-config", JobsController.ReloadConfig)

		// 系统状态接口
		JobsRouters.GET("/health", JobsController.Health)
		JobsRouters.GET("/jobStatus", func(c *gin.Context) {
			running := global.Timer != nil && global.TimerRunning
			c.JSON(200, gin.H{
				"running":    running,
				"task_count": len(global.TaskList),
			})
		})
	}
}

// RegisterJobRoutes 预留任务相关路由注册（如需拆分可实现）
func RegisterJobRoutes(r *gin.Engine) {
	// TODO: 注册任务相关路由
}
