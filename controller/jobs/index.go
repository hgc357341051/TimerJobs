package index

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"
	funcs "xiaohuAdmin/function"
	"xiaohuAdmin/global"
	"xiaohuAdmin/middlewares"
	"xiaohuAdmin/models/jobs"

	"github.com/gin-gonic/gin"
)

type Index struct{} //分组结构体

type LogEntry struct {
	Level     string `json:"level"`
	Time      string `json:"time"`
	Caller    string `json:"caller"`
	Message   string `json:"msg"`
	JobID     uint   `json:"id,omitempty"`
	Entry_id  uint   `json:"entry_id,omitempty"`
	Method    string `json:"method,omitempty"`
	Path      string `json:"path,omitempty"`
	Ip        string `json:"ip,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
	Status    int    `json:"status,omitempty"`
}

// JobRequest 任务请求结构体
// 用于新增、删除、停止、查询任务等接口
// 字段含义见各接口注释
// 示例：{"name":"测试任务","desc":"描述","cron_expr":"* * * * * *","mode":"command","command":"echo hello","state":0,"allow_mode":0,"max_run_count":0}
type JobRequest struct {
	ID          uint   `form:"id" json:"id"`
	Name        string `form:"name" json:"name"`
	Desc        string `form:"desc,omitempty" json:"desc,omitempty"`
	CronExpr    string `form:"cron_expr" json:"cron_expr"`
	Mode        string `form:"mode" json:"mode"`
	Command     string `form:"command" json:"command"`
	State       int    `form:"state,omitempty" json:"state,omitempty"`
	AllowMode   int    `form:"allow_mode,omitempty" json:"allow_mode,omitempty"`
	MaxRunCount int    `form:"max_run_count,omitempty" json:"max_run_count,omitempty"`
	Page        int    `form:"page" json:"page"`
	Size        int    `form:"size" json:"size"`
}

// JobEditRequest 任务编辑结构体
// 用于编辑任务接口
// 示例：{"id":1,"name":"新名称","desc":"新描述"}
type JobEditRequest struct {
	ID          uint    `form:"id" json:"id" binding:"required"`
	Name        *string `form:"name" json:"name"`
	Desc        *string `form:"desc" json:"desc"`
	CronExpr    *string `form:"cron_expr" json:"cron_expr"`
	Mode        *string `form:"mode" json:"mode"`
	Command     *string `form:"command" json:"command"`
	State       *int    `form:"state" json:"state"`
	AllowMode   *int    `form:"allow_mode" json:"allow_mode"`
	MaxRunCount *uint   `form:"max_run_count" json:"max_run_count"`
}

// JobRunRequest 任务运行结构体
// 用于手动运行任务等接口
// 示例：{"id":1}
type JobRunRequest struct {
	ID uint `form:"id" json:"id" binding:"required"`
}

// JobLogsRequest 日志查询结构体
// 用于任务日志查询接口
// 示例：{"id":1,"date":"2025-06-25","limit":3}
type JobLogsRequest struct {
	ID    uint   `json:"id" binding:"required"`
	Date  string `json:"date"`
	Limit int    `json:"limit"`
	Page  int    `form:"page" json:"page"`
	Size  int    `form:"size" json:"size"`
}

// IPRequest IP白/黑名单结构体
// 用于IP白名单、黑名单相关接口
// 示例：{"ip":"127.0.0.1"}
type IPRequest struct {
	IP string `form:"ip" json:"ip" binding:"required"`
}

// 通用参数绑定和校验
func bindAndValidate(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBind(obj); err != nil {
		funcs.No(c, "参数错误："+err.Error(), nil)
		return false
	}
	return true
}

// @Summary 系统健康检查
// @Description 检查系统运行状态
// @Tags 系统状态
// @Accept json
// @Produce json
// @Success 200 {object} function.JsonData "成功响应"
// @Router /jobs/health [get]
func (*Index) Health(c *gin.Context) {
	config := global.GetGlobalConfig()

	funcs.Ok(c, "系统运行正常", gin.H{
		"app_name":    config.App.Name,
		"app_version": config.App.Version,
		"server_port": config.Server.Port,
		"timestamp":   time.Now().Unix(),
		"uptime":      getUptime(),
		"memory":      getMemoryStats(),
		"goroutines":  runtime.NumGoroutine(),
	})
}

// DatabaseInfo 数据库信息
func DatabaseInfo(c *gin.Context) {
	dbInfo := global.GetDatabaseInfo()
	dbStats := global.GetDBStats()

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"information": dbInfo,
			"statistics":  dbStats,
		},
	})
}

// @Summary 新增任务
// @Description 新增一个定时任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param data body index.JobRequest true "任务参数" 例：{"name":"测试任务","desc":"描述","cron_expr":"* * * * * *","mode":"command","command":"echo hello","state":0,"allow_mode":0,"max_run_count":0}
// @Success 200 {object} function.JsonData "操作成功"
// @Failure 400 {object} function.JsonData "参数错误"
// @Router /jobs/add [post]
func (*Index) AddJob(c *gin.Context) {
	var jobReq JobRequest
	if !bindAndValidate(c, &jobReq) {
		return
	}
	job := jobs.Jobs{
		Name:        jobReq.Name,
		Desc:        jobReq.Desc,
		CronExpr:    jobReq.CronExpr,
		Mode:        jobReq.Mode,
		Command:     jobReq.Command,
		State:       jobReq.State,
		MaxRunCount: uint(jobReq.MaxRunCount),
		AllowMode:   jobReq.AllowMode,
	}
	if err := global.CreateJob(&job); err != nil {
		global.ZapLog.Error("任务添加失败1",
			global.LogField("name", job.Name),
			global.LogError(err))
		funcs.No(c, "任务添加失败1："+err.Error(), nil)
		return
	}
	global.ZapLog.Info("任务添加成功",
		global.LogField("name", job.Name),
		global.LogField("id", job.ID))
	funcs.Ok(c, "任务添加成功", nil)
}

// @Summary 删除任务
// @Description 根据ID删除指定任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param data body index.JobRequest true "任务ID" 例：{"id":1}
// @Success 200 {object} function.JsonData "操作成功"
// @Failure 400 {object} function.JsonData "参数错误"
// @Router /jobs/del [post]
func (*Index) DeleteJob(c *gin.Context) {
	var jobReq JobRequest
	if !bindAndValidate(c, &jobReq) {
		return
	}
	job := jobs.Jobs{ID: jobReq.ID}
	if err := global.DB.First(&job).Error; err != nil {
		funcs.No(c, "任务未找到："+err.Error(), nil)
		return
	}
	if err := global.DB.Delete(&job).Error; err != nil {
		funcs.No(c, "任务删除失败："+err.Error(), nil)
		return
	}
	if job.State == 1 || job.State == 0 {
		if err := global.RemoveJob(job.ID); err != nil {
			// 记录错误但不影响删除操作的成功
			if global.ZapLog != nil {
				global.ZapLog.Error("从调度器移除任务失败", global.LogError(err))
			}
		}
	}
	funcs.Ok(c, "任务删除成功", nil)
}

// @Summary 停止任务
// @Description 停止正在运行的任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param data body index.JobRequest true "任务ID" 例：{"id":1}
// @Success 200 {object} function.JsonData "操作成功"
// @Failure 400 {object} function.JsonData "参数错误"
// @Failure 404 {object} function.JsonData "任务未找到"
// @Router /jobs/stop [post]
func (*Index) StopJob(c *gin.Context) {
	var jobReq JobRequest
	if !bindAndValidate(c, &jobReq) {
		return
	}
	job := jobs.Jobs{ID: jobReq.ID}
	if err := global.DB.First(&job).Error; err != nil {
		funcs.No(c, "任务未找到："+err.Error(), nil)
		return
	}
	if job.State == 1 || job.State == 0 {
		job.State = 2
		if err := global.DB.Save(&job).Error; err != nil {
			funcs.No(c, "任务停止失败："+err.Error(), nil)
			return
		}
		if err := global.RemoveJob(job.ID); err != nil {
			// 记录错误但不影响停止操作的成功
			if global.ZapLog != nil {
				global.ZapLog.Error("从调度器移除任务失败", global.LogError(err))
			}
		}
		funcs.Ok(c, "任务停止成功", nil)
	} else {
		funcs.Ok(c, "任务已停止", nil)
	}
}

// @Summary 查询任务详情
// @Description 根据ID查询任务详情
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param id query int true "任务ID"
// @Success 200 {object} jobs.Jobs "任务详情"
// @Failure 400 {object} function.JsonData "参数错误"
// @Router /jobs/read [get]
func (*Index) JobInfo(c *gin.Context) {
	var jobReq JobRequest
	if !bindAndValidate(c, &jobReq) {
		return
	}
	job := jobs.Jobs{ID: jobReq.ID}
	if err := global.DB.First(&job).Error; err != nil {
		funcs.No(c, "任务未找到："+err.Error(), nil)
		return
	}
	funcs.Ok(c, "任务信息", job)
}

// @Summary 获取任务列表
// @Description 分页查询任务列表
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} function.PageData "分页数据"
// @Failure 400 {object} function.JsonData "参数错误"
// @Router /jobs/list [get]
func (*Index) JobList(c *gin.Context) {
	var jobReq JobRequest
	if !bindAndValidate(c, &jobReq) {
		return
	}
	if jobReq.Page == 0 {
		jobReq.Page = 1
	}
	if jobReq.Size == 0 {
		jobReq.Size = 10
	}
	var jobList []jobs.Jobs
	offset := (jobReq.Page - 1) * jobReq.Size
	limit := jobReq.Size
	var total int64
	if err := global.DB.Model(&jobs.Jobs{}).Count(&total).Error; err != nil {
		funcs.No(c, "查询任务总数失败："+err.Error(), nil)
		return
	}
	if err := global.DB.Offset(int(offset)).Limit(int(limit)).Find(&jobList).Error; err != nil {
		funcs.No(c, "查询任务列表失败："+err.Error(), nil)
		return
	}
	totalPages := (total + int64(jobReq.Size) - 1) / int64(jobReq.Size)
	funcs.JsonPage(c, "查询任务列表成功", jobList, total, totalPages, jobReq.Page, jobReq.Size)
}

// @Summary 编辑任务
// @Description 编辑指定任务的信息
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param data body index.JobEditRequest true "任务编辑参数" 例：{"id":1,"name":"新名称"}
// @Success 200 {object} function.JsonData "操作成功"
// @Failure 400 {object} function.JsonData "参数错误"
// @Router /jobs/edit [post]
func (*Index) EditJob(c *gin.Context) {
	var jobReq JobEditRequest
	if !bindAndValidate(c, &jobReq) {
		return
	}
	var oldJob jobs.Jobs
	if err := global.DB.First(&oldJob, jobReq.ID).Error; err != nil {
		funcs.No(c, "任务未找到："+err.Error(), nil)
		return
	}
	needRestart := jobReq.CronExpr != nil || jobReq.Mode != nil || jobReq.Command != nil || jobReq.State != nil || jobReq.AllowMode != nil || jobReq.MaxRunCount != nil
	if jobReq.Name != nil {
		oldJob.Name = *jobReq.Name
	}
	if jobReq.Desc != nil {
		oldJob.Desc = *jobReq.Desc
	}
	if jobReq.CronExpr != nil {
		oldJob.CronExpr = *jobReq.CronExpr
	}
	if jobReq.Mode != nil {
		oldJob.Mode = *jobReq.Mode
	}
	if jobReq.Command != nil {
		oldJob.Command = *jobReq.Command
	}
	if jobReq.State != nil {
		oldJob.State = *jobReq.State
	}
	if jobReq.AllowMode != nil {
		oldJob.AllowMode = *jobReq.AllowMode
	}
	if jobReq.MaxRunCount != nil {
		oldJob.MaxRunCount = *jobReq.MaxRunCount
	}
	if err := global.DB.Save(&oldJob).Error; err != nil {
		funcs.No(c, "任务更新失败："+err.Error(), nil)
		return
	}
	if needRestart {
		if err := global.UpdateJob(&oldJob); err != nil {
			funcs.No(c, "任务调度器更新失败："+err.Error(), nil)
			return
		}
	}
	funcs.Ok(c, "任务更新成功", nil)
}

// @Summary 手动运行任务
// @Description 手动运行指定任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param data body index.JobRunRequest true "任务ID" 例：{"id":1}
// @Success 200 {object} function.JsonData "操作成功"
// @Failure 400 {object} function.JsonData "参数错误"
// @Router /jobs/run [post]
func (*Index) JobRun(c *gin.Context) {
	var jobRunReq JobRunRequest
	if !bindAndValidate(c, &jobRunReq) {
		return
	}
	var job global.Jobs
	if err := global.DB.First(&job, jobRunReq.ID).Error; err != nil {
		funcs.No(c, "任务未找到", nil)
		return
	}
	// 根据配置决定是否允许手动并发
	if global.GetJobsConfigBool("jobs.manual_allow_concurrent", true) {
		execID := global.RunJobManually(&job)
		funcs.Ok(c, "任务已手动执行", gin.H{"exec_id": execID, "skipped": false})
		return
	}
	// 不允许并发时，按 AllowMode 执行策略
	execID, skipped, reason := global.RunJobManuallyWithPolicy(&job)
	if skipped {
		funcs.Ok(c, "任务已按策略跳过", gin.H{"skipped": true, "reason": reason})
		return
	}
	funcs.Ok(c, "任务已手动执行", gin.H{"exec_id": execID, "skipped": false})
}

// @Summary 启动所有任务
// @Description 启动任务调度器运行所有任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Success 200 {object} function.JsonData "成功响应"
// @Router /jobs/runAll [post]
func (*Index) JobRunAll(c *gin.Context) {

	global.Timer.Start()
	global.TimerRunning = true
	funcs.Ok(c, "任务调度器启动成功", nil)
}

// @Summary 停止所有任务
// @Description 停止任务调度器所有任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Success 200 {object} function.JsonData "成功响应"
// @Router /jobs/stopAll [post]
func (*Index) JobStopAll(c *gin.Context) {

	global.StopTimer()
	funcs.Ok(c, "任务调度器停止成功", nil)
}

// @Summary 查询任务状态
// @Description 查询任务调度器运行状态
// @Tags 任务管理
// @Accept json
// @Produce json
// @Success 200 {object} function.JsonData "成功响应"
// @Router /jobs/jobState [get]
func (*Index) JobState(c *gin.Context) {
	entries := global.Timer.Entries()
	if global.IsTimerRunning() {

		funcs.Ok(c, "任务运行中", gin.H{"num": len(entries), "state": true})
	} else {
		funcs.Ok(c, "任务停止", gin.H{"num": len(entries), "state": false})
	}
}

// @Summary 重启任务
// @Description 根据任务ID重启任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param data body object true "请求体，格式为{id:1}"
// @Success 200 {object} function.JsonData "任务重启成功"
// @Failure 404 {object} function.JsonData "参数错误"
// @Failure 404 {object} function.JsonData "任务未找到"
// @Router /jobs/restart [post]
func (*Index) JobRestart(c *gin.Context) {
	var jobReq JobRequest
	if !bindAndValidate(c, &jobReq) {
		return
	}
	var job jobs.Jobs
	if err := global.DB.First(&job, jobReq.ID).Error; err != nil {
		funcs.No(c, "任务未找到："+err.Error(), nil)
		return
	}

	// 先停止任务（从调度器中移除）
	if err := global.RemoveJob(job.ID); err != nil {
		// 记录错误但不影响重启操作
		global.ZapLog.Warn("移除旧任务失败", global.LogError(err), global.LogField("job_id", job.ID))
	}

	// 无论任务之前是什么状态，都将状态设置为等待（0）并重新添加到调度器
	job.State = 0 // 将状态改为等待
	if err := global.DB.Save(&job).Error; err != nil {
		funcs.No(c, "更新任务状态失败："+err.Error(), nil)
		return
	}

	// 重新添加到调度器
	if err := global.AddJob(&job); err != nil {
		funcs.No(c, "任务重启失败："+err.Error(), nil)
		return
	}

	funcs.Ok(c, "任务重启成功", nil)
}

// @Summary 查看系统日志
// @Description 分页查询系统运行日志
// @Tags 日志管理
// @Accept json
// @Produce json
// @Param date query string false "查询日期(格式:YYYY-MM-DD)"
// @Param page query integer false "页码" default(1)
// @Param size query integer false "每页数量" default(10)
// @Success 200 {object} function.PageData "成功响应"
// @Failure 404 {object} function.JsonData "参数错误"
// @Router /jobs/zapLogs [get]
func (*Index) ZapLogs(c *gin.Context) {
	// 日志条目结构体

	type JobLogsRequest struct {
		Date string `form:"date" json:"date"`
		Page int    `form:"page" json:"page"`
		Size int    `form:"size" json:"size"`
	}

	var req JobLogsRequest
	if err := c.ShouldBind(&req); err != nil {
		funcs.No(c, "参数错误："+err.Error(), nil)
		return
	}

	// 默认值处理
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.Size <= 0 {
		req.Size = 10
	}

	// 异步处理日志文件读取和解析
	resultChan := make(chan struct {
		entries []LogEntry
		err     error
	}, 1)

	go func() {
		// 解析日期并构建日志文件路径
		dateFormatted := ""
		if req.Date != "" {
			t, err := time.Parse("2006-01-02", req.Date)
			if err != nil {
				resultChan <- struct {
					entries []LogEntry
					err     error
				}{nil, fmt.Errorf("日期格式错误，应为 YYYY-MM-DD")}
				return
			}
			dateFormatted = t.Format("20060102")
		} else {
			dateFormatted = time.Now().Format("20060102")
		}

		// 构建系统日志文件路径（注意目录为runtime）
		logDir := "runtime"
		logFile := fmt.Sprintf("%s/logs_%s.log", logDir, dateFormatted)

		// 检查文件是否存在
		if _, err := os.Stat(logFile); os.IsNotExist(err) {
			resultChan <- struct {
				entries []LogEntry
				err     error
			}{[]LogEntry{}, nil}
			return
		}

		// 读取日志文件
		file, err := os.Open(logFile)
		if err != nil {
			resultChan <- struct {
				entries []LogEntry
				err     error
			}{nil, fmt.Errorf("打开日志文件失败：%v", err)}
			return
		}
		defer file.Close()

		// 解析日志条目
		var entries []LogEntry
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Bytes()
			var entry LogEntry
			if err := json.Unmarshal(line, &entry); err == nil {
				entries = append(entries, entry)
			}
		}

		// 按时间倒序排序（最新在前）
		sort.Slice(entries, func(i, j int) bool {
			t1, _ := time.Parse("2006-01-02 15:04:05", entries[i].Time)
			t2, _ := time.Parse("2006-01-02 15:04:05", entries[j].Time)
			return t1.After(t2)
		})

		resultChan <- struct {
			entries []LogEntry
			err     error
		}{entries, nil}
	}()

	// 等待异步结果
	result := <-resultChan
	if result.err != nil {
		funcs.No(c, result.err.Error(), nil)
		return
	}

	entries := result.entries

	// 分页处理
	total := len(entries)
	totalPages := (total + req.Size - 1) / req.Size
	start := (req.Page - 1) * req.Size
	end := req.Page * req.Size

	if start > total {
		start = total
	}
	if end > total {
		end = total
	}

	pageEntries := entries[start:end]

	funcs.JsonPage(c, "系统日志查询成功", pageEntries, int64(total), int64(totalPages), req.Page, req.Size)
}

// @Summary 获取日志开关状态
// @Description 获取系统日志开关状态
// @Tags 日志管理
// @Accept json
// @Produce json
// @Success 200 {object} function.JsonData "成功响应"
// @Router /jobs/switchState [get]
func (*Index) LogSwitchState(c *gin.Context) {
	config := global.GetGlobalConfig()

	funcs.Ok(c, "获取日志开关状态成功", gin.H{
		"zapLogSwitch": config.Logs.ZapLogSwitch,
		"jobLogSwitch": config.Logs.ZapLogSwitch, // 使用相同的开关
	})
}

// @Summary 校准任务列表
// @Description 校准任务列表
// @Tags 任务管理
// @Accept json
// @Produce json
// @Success 200 {object} function.JsonData "成功响应"
// @Router /jobs/checkJob [post]
func (i *Index) CalibrateJobList(c *gin.Context) {

	// 查询所有状态为0或1的任务
	var dbJobs []jobs.Jobs
	if err := global.DB.Where("state IN (?)", []int{0, 1}).Find(&dbJobs).Error; err != nil {
		funcs.No(c, "查询任务失败: "+err.Error(), nil)
		return
	}

	// 创建数据库中有效任务ID的集合
	validJobIDs := make(map[uint]bool)
	for _, job := range dbJobs {
		validJobIDs[job.ID] = true
	}

	// 检查当前TaskList中的任务是否在有效任务中
	for taskID := range global.TaskList {
		if _, exists := validJobIDs[taskID]; !exists {
			// 任务不在有效列表中，需要移除
			if err := global.RemoveJob(taskID); err != nil {
				global.ZapLog.Error("移除无效任务失败",
					global.LogField("taskID", taskID),
					global.LogError(err))
				// 记录错误但继续处理其他任务
			} else {
				global.ZapLog.Info("已移除无效任务",
					global.LogField("taskID", taskID))
			}
		}
	}

	// 添加缺失的任务
	for _, job := range dbJobs {
		if _, exists := global.TaskList[job.ID]; !exists {
			if err := global.AddJob(&job); err != nil {
				global.ZapLog.Error("添加任务失败",
					global.LogField("jobID", job.ID),
					global.LogField("jobName", job.Name),
					global.LogError(err))
				// 记录错误但继续处理其他任务
			} else {
				global.ZapLog.Info("已添加缺失任务",
					global.LogField("jobID", job.ID),
					global.LogField("jobName", job.Name))
			}
		}
	}

	funcs.Ok(c, "任务列表已校准", nil)

}

// @Summary 获取调度器任务列表
// @Description 获取当前调度器中正在运行的任务列表
// @Tags 任务管理
// @Accept json
// @Produce json
// @Success 200 {object} function.JsonData "成功响应"
// @Failure 404 {object} function.JsonData "参数错误"
// @Router /jobs/scheduler [get]
func (*Index) GetSchedulerTasks(c *gin.Context) {
	// 获取调度器中的任务条目
	entries := global.Timer.Entries()

	// 构建任务列表
	var taskList []map[string]interface{}

	for _, entry := range entries {
		// 从TaskList中查找对应的任务ID
		var jobID uint
		for id, entryID := range global.GetTaskListSnapshot() {
			if entryID == entry.ID {
				jobID = id
				break
			}
		}

		// 获取任务详细信息
		var job global.Jobs
		if err := global.DB.First(&job, jobID).Error; err == nil {
			taskInfo := map[string]interface{}{
				"id":         jobID,
				"name":       job.Name,
				"desc":       job.Desc,
				"cron_expr":  job.CronExpr,
				"mode":       job.Mode,
				"command":    job.Command,
				"state":      job.State,
				"next_run":   entry.Next.Format("2006-01-02 15:04:05"),
				"prev_run":   entry.Prev.Format("2006-01-02 15:04:05"),
				"run_count":  job.RunCount,
				"created_at": job.CreatedAt,
				"updated_at": job.UpdatedAt,
			}
			taskList = append(taskList, taskInfo)
		}
	}

	// 返回调度器状态和任务列表
	funcs.Ok(c, "获取调度器任务列表成功", gin.H{
		"scheduler_running": global.IsTimerRunning(),
		"total_tasks":       len(entries),
		"tasks":             taskList,
	})
}

// @Summary 获取可用函数列表
// @Description 获取系统中所有可用的函数任务列表
// @Tags 任务管理
// @Accept json
// @Produce json
// @Success 200 {object} function.JsonData "成功响应"
// @Router /jobs/functions [get]
func (*Index) GetFunctions(c *gin.Context) {
	functions := global.ListFunctions()

	// 构建函数详细信息
	var functionDetails []map[string]interface{}
	for _, name := range functions {
		detail := map[string]interface{}{
			"name":        name,
			"description": getFunctionDescription(name),
			"parameters":  getFunctionParameters(name),
		}
		functionDetails = append(functionDetails, detail)
	}

	funcs.Ok(c, "获取函数列表成功", functionDetails)
}

// getFunctionDescription 获取函数描述
func getFunctionDescription(name string) string {
	descriptions := map[string]string{
		"Dayin":    "示例函数 - 打印任务信息",
		"Test":     "测试函数 - 用于测试任务执行",
		"Hello":    "问候函数 - 发送问候消息",
		"Time":     "时间函数 - 获取当前时间",
		"Echo":     "回显函数 - 回显输入参数",
		"Math":     "数学计算函数 - 执行数学运算",
		"File":     "文件操作函数 - 文件读写删除",
		"Database": "数据库操作函数 - 执行SQL语句",
		"Email":    "邮件发送函数 - 发送邮件",
		"SMS":      "短信发送函数 - 发送短信",
		"Webhook":  "Webhook调用函数 - 调用外部接口",
		"Backup":   "备份函数 - 数据备份",
		"Cleanup":  "清理函数 - 清理临时文件",
		"Monitor":  "监控函数 - 系统监控",
		"Report":   "报告函数 - 生成报告",
	}

	if desc, exists := descriptions[name]; exists {
		return desc
	}
	return "未知函数"
}

// getFunctionParameters 获取函数参数说明
func getFunctionParameters(name string) string {
	parameters := map[string]string{
		"Dayin":    "参数1,参数2,参数3 - 任意参数；times(可选)，interval(秒，可选)",
		"Test":     "任意参数 - 用于测试；times(可选)，interval(秒，可选)",
		"Hello":    "name - 问候对象名称；times(可选)，interval(秒，可选)",
		"Time":     "format - 时间格式(可选)；times(可选)，interval(秒，可选)",
		"Echo":     "任意参数 - 回显内容；times(可选)，interval(秒，可选)",
		"Math":     "操作符 数字1 数字2 - 数学运算；times(可选)，interval(秒，可选)",
		"File":     "操作 文件路径 - 文件操作；times(可选)，interval(秒，可选)",
		"Database": "操作 SQL语句 - 数据库操作；times(可选)，interval(秒，可选)",
		"Email":    "收件人 主题 内容 - 邮件发送；times(可选)，interval(秒，可选)",
		"SMS":      "手机号 内容 - 短信发送；times(可选)，interval(秒，可选)",
		"Webhook":  "URL 数据(可选) - Webhook调用；times(可选)，interval(秒，可选)",
		"Backup":   "源路径(可选) - 数据备份；times(可选)，interval(秒，可选)",
		"Cleanup":  "路径(可选) - 清理操作；times(可选)，interval(秒，可选)",
		"Monitor":  "目标(可选) - 监控目标；times(可选)，interval(秒，可选)",
		"Report":   "报告类型(可选) - 报告类型；times(可选)，interval(秒，可选)",
	}

	if param, exists := parameters[name]; exists {
		return param
	}
	return "无参数（支持 times/interval 可选）"
}

// @Summary 查询任务日志
// @Description 按任务ID和日期查询任务日志，默认返回最新3条
// @Tags 日志管理
// @Accept json
// @Produce json
// @Param data body index.JobLogsRequest true "日志查询参数" 例：{"id":1,"date":"2025-06-25","limit":3}
// @Success 200 {object} function.JsonData "查询成功"
// @Failure 400 {object} function.JsonData "参数错误"
// @Router /jobs/logs [post]
func (*Index) JobLogs(c *gin.Context) {
	var req JobLogsRequest
	if !bindAndValidate(c, &req) {
		return
	}

	// 如果date为空，默认当天
	dateStr := req.Date
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	logFile := fmt.Sprintf("runtime/jobs/%d/%s/%s/%s.log", req.ID, dateStr[:4], dateStr[5:7], dateStr[8:10])
	file, err := os.Open(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			c.JSON(200, gin.H{
				"code": 200,
				"msg":  "查询成功",
				"data": []interface{}{},
			})
		} else {
			funcs.No(c, "打开日志文件失败："+err.Error(), nil)
		}
		return
	}
	defer file.Close()

	// 读取所有行
	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		funcs.No(c, "读取日志文件失败："+err.Error(), nil)
		return
	}

	// 解析为JobExecLog对象数组
	var logs []map[string]interface{}
	keepCount := getJobLogKeepCount()
	limit := req.Limit
	if limit <= 0 {
		limit = keepCount
	}
	if limit <= 0 {
		limit = len(lines) // 0为不限制
	}
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			continue
		}
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err == nil {
			logs = append(logs, obj)
		}
		if len(logs) >= limit {
			break
		}
	}
	// 反转为最新在前
	for l, r := 0, len(logs)-1; l < r; l, r = l+1, r-1 {
		logs[l], logs[r] = logs[r], logs[l]
	}

	c.JSON(200, gin.H{
		"code": 200,
		"msg":  "查询成功",
		"data": logs,
	})
}

// @Summary 按执行ID查询任务执行结果
// @Description 通过任务ID与exec_id查询某次执行的汇总日志（可选date，默认当天）
// @Tags 日志管理
// @Accept json
// @Produce json
// @Param id query int true "任务ID"
// @Param exec_id query string true "执行ID"
// @Param date query string false "查询日期(YYYY-MM-DD)"
// @Success 200 {object} function.JsonData "查询成功"
// @Failure 400 {object} function.JsonData "参数错误"
// @Router /jobs/execs [get]
func (*Index) GetExecByID(c *gin.Context) {
	jobID := funcs.GetQueryInt(c, "id", 0)
	execID := funcs.GetQueryString(c, "exec_id", "")
	dateStr := funcs.GetQueryString(c, "date", "")
	if jobID <= 0 || strings.TrimSpace(execID) == "" {
		funcs.No(c, "参数错误：id 和 exec_id 必填", nil)
		return
	}
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}
	logFile := fmt.Sprintf("runtime/jobs/%d/%s/%s/%s.log", jobID, dateStr[:4], dateStr[5:7], dateStr[8:10])
	file, err := os.Open(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			funcs.Ok(c, "未找到执行记录", nil)
		} else {
			funcs.No(c, "打开日志文件失败："+err.Error(), nil)
		}
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}
		var entry map[string]interface{}
		if err := json.Unmarshal(line, &entry); err == nil {
			if v, ok := entry["exec_id"].(string); ok && v == execID {
				funcs.Ok(c, "查询成功", entry)
				return
			}
		}
	}
	if err := scanner.Err(); err != nil {
		funcs.No(c, "读取日志文件失败："+err.Error(), nil)
		return
	}
	funcs.Ok(c, "未找到执行记录", nil)
}

// 获取日志保留数量
func getJobLogKeepCount() int {
	config := global.GetGlobalConfig()
	if config != nil {
		return config.JobLogKeepCount
	}
	return 3
}

// @Summary 获取IP控制状态
// @Description 获取IP白名单和黑名单配置状态
// @Tags IP控制
// @Accept json
// @Produce json
// @Success 200 {object} function.JsonData "成功响应"
// @Router /jobs/ip-control/status [get]
func (*Index) GetIPControlStatus(c *gin.Context) {
	status := middlewares.GetIPControlStatus()
	funcs.Ok(c, "获取IP控制状态成功", status)
}

// @Summary 添加IP到白名单
// @Description 添加IP到白名单
// @Tags IP管理
// @Accept json
// @Produce json
// @Param data body index.IPRequest true "IP参数" 例：{"ip":"127.0.0.1"}
// @Success 200 {object} function.JsonData "操作成功"
// @Failure 400 {object} function.JsonData "参数错误"
// @Router /jobs/ip-control/whitelist/add [post]
func (*Index) AddToWhitelist(c *gin.Context) {
	var req IPRequest
	if !bindAndValidate(c, &req) {
		return
	}

	if err := middlewares.AddToWhitelist(req.IP); err != nil {
		funcs.No(c, "添加白名单失败："+err.Error(), nil)
		return
	}

	funcs.Ok(c, "IP已添加到白名单", nil)
}

// @Summary 从白名单移除IP
// @Description 从白名单移除指定IP
// @Tags IP管理
// @Accept json
// @Produce json
// @Param data body index.IPRequest true "IP参数" 例：{"ip":"127.0.0.1"}
// @Success 200 {object} function.JsonData "操作成功"
// @Failure 400 {object} function.JsonData "参数错误"
// @Router /jobs/ip-control/whitelist/remove [post]
func (*Index) RemoveFromWhitelist(c *gin.Context) {
	var req IPRequest
	if !bindAndValidate(c, &req) {
		return
	}

	if err := middlewares.RemoveFromWhitelist(req.IP); err != nil {
		funcs.No(c, "移除白名单失败："+err.Error(), nil)
		return
	}

	funcs.Ok(c, "IP已从白名单中移除", nil)
}

// @Summary 添加IP到黑名单
// @Description 添加IP到黑名单
// @Tags IP管理
// @Accept json
// @Produce json
// @Param data body index.IPRequest true "IP参数" 例：{"ip":"127.0.0.1"}
// @Success 200 {object} function.JsonData "操作成功"
// @Failure 400 {object} function.JsonData "参数错误"
// @Router /jobs/ip-control/blacklist/add [post]
func (*Index) AddToBlacklist(c *gin.Context) {
	var req IPRequest
	if !bindAndValidate(c, &req) {
		return
	}

	if err := middlewares.AddToBlacklist(req.IP); err != nil {
		funcs.No(c, "添加黑名单失败："+err.Error(), nil)
		return
	}

	funcs.Ok(c, "IP已添加到黑名单", nil)
}

// @Summary 从黑名单移除IP
// @Description 从黑名单移除指定IP
// @Tags IP管理
// @Accept json
// @Produce json
// @Param data body index.IPRequest true "IP参数" 例：{"ip":"127.0.0.1"}
// @Success 200 {object} function.JsonData "操作成功"
// @Failure 400 {object} function.JsonData "参数错误"
// @Router /jobs/ip-control/blacklist/remove [post]
func (*Index) RemoveFromBlacklist(c *gin.Context) {
	var req IPRequest
	if !bindAndValidate(c, &req) {
		return
	}

	if err := middlewares.RemoveFromBlacklist(req.IP); err != nil {
		funcs.No(c, "移除黑名单失败："+err.Error(), nil)
		return
	}

	funcs.Ok(c, "IP已从黑名单中移除", nil)
}

// @Summary 重载配置
// @Description 重新加载配置文件并更新全局配置
// @Tags 系统管理
// @Accept json
// @Produce json
// @Success 200 {object} function.JsonData "成功响应"
// @Failure 500 {object} function.JsonData "重载失败"
// @Router /jobs/reload-config [post]
func (*Index) ReloadConfig(c *gin.Context) {
	if err := global.ReloadConfig(); err != nil {
		funcs.No(c, "重载配置失败："+err.Error(), nil)
		return
	}

	funcs.Ok(c, "配置重载成功", nil)
}

// @Summary 获取任务系统配置
// @Description 返回 jobs.* 相关配置快照
// @Tags 任务管理
// @Accept json
// @Produce json
// @Success 200 {object} function.JsonData "成功响应"
// @Router /jobs/config [get]
func (*Index) GetJobsConfig(c *gin.Context) {
	cfg := global.GetGlobalConfig()
	if cfg == nil {
		funcs.No(c, "配置未初始化", nil)
		return
	}
	data := gin.H{
		"default_allow_mode":      cfg.Jobs.DefaultAllowMode,
		"manual_allow_concurrent": cfg.Jobs.ManualAllowConcurrent,
		"default_timeout_seconds": cfg.Jobs.DefaultTimeoutSeconds,
		"http_response_max_bytes": cfg.Jobs.HTTPResponseMaxBytes,
		"log_summary_enabled":     cfg.Jobs.LogSummaryEnabled,
		"log_line_truncate":       cfg.Jobs.LogLineTruncate,
	}
	funcs.Ok(c, "获取配置成功", data)
}

// getUptime 获取系统运行时间（秒）
func getUptime() int64 {
	// 计算从程序启动到现在的秒数
	return int64(time.Since(global.StartTime).Seconds())
}

// getMemoryStats 获取内存统计
func getMemoryStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"alloc":       m.Alloc,
		"total_alloc": m.TotalAlloc,
		"sys":         m.Sys,
		"num_gc":      m.NumGC,
	}
}

// ClearLogsRequest 清空日志请求结构体
// 用于清空任务日志或系统日志
// 示例：{"id":1,"date":"2025-06-25","type":"job"}
type ClearLogsRequest struct {
	ID   uint   `json:"id"`                      // 任务ID，type为job时必填
	Date string `json:"date"`                    // 日期，格式：YYYY-MM-DD
	Type string `json:"type" binding:"required"` // 日志类型：job 或 zap
}

// @Summary 清空日志
// @Description 清空指定任务或系统指定日期的日志文件
// @Tags 日志管理
// @Accept json
// @Produce json
// @Param data body index.ClearLogsRequest true "清空日志参数" 例：{"id":1,"date":"2025-06-25","type":"job"}
// @Success 200 {object} function.JsonData "操作成功"
// @Failure 400 {object} function.JsonData "参数错误"
// @Router /jobs/logs/clear [post]
func (*Index) ClearLogs(c *gin.Context) {
	var req ClearLogsRequest
	if !bindAndValidate(c, &req) {
		return
	}

	// 如果date为空，默认当天
	dateStr := req.Date
	if dateStr == "" {
		dateStr = time.Now().Format("2006-01-02")
	}

	var logFile string
	if req.Type == "job" {
		if req.ID <= 0 {
			funcs.No(c, "参数错误：任务ID不能为空", nil)
			return
		}
		logFile = fmt.Sprintf("runtime/jobs/%d/%s/%s/%s.log", req.ID, dateStr[:4], dateStr[5:7], dateStr[8:10])
	} else if req.Type == "zap" {
		logFile = fmt.Sprintf("runtime/logs_%s.log", strings.ReplaceAll(dateStr, "-", ""))
	} else {
		funcs.No(c, "参数错误：type必须为job或zap", nil)
		return
	}

	// 检查文件是否存在
	if _, err := os.Stat(logFile); os.IsNotExist(err) {
		funcs.Ok(c, "日志文件不存在，无需清空", gin.H{
			"file":   logFile,
			"exists": false,
		})
		return
	}

	// 清空日志文件
	if err := os.Truncate(logFile, 0); err != nil {
		funcs.No(c, "清空日志文件失败："+err.Error(), nil)
		return
	}

	funcs.Ok(c, "日志已清空", gin.H{
		"file":    logFile,
		"cleared": true,
	})
}
