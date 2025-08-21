package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// API配置
const (
	APIBaseURL = "http://127.0.0.1:36363"
	APITimeout = 30 * time.Second
)

// Job结构体 - 映射后端API的数据结构
type Job struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Desc        string    `json:"desc"`
	CronExpr    string    `json:"cron_expr"`
	Mode        string    `json:"mode"`
	Command     string    `json:"command"`
	State       int       `json:"state"` // 0等待 1运行中 2已停止
	RunCount    uint      `json:"run_count"`
	MaxRunCount uint      `json:"max_run_count"`
	AllowMode   int       `json:"allow_mode"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// API响应结构
type APIResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// 分页响应结构
type PageResponse struct {
	Code  int    `json:"code"`
	Msg   string `json:"msg"`
	Data  []Job  `json:"data"`
	Total int64  `json:"total"`
	Pages int64  `json:"pages"`
	Page  int    `json:"page"`
	Size  int    `json:"size"`
}

// HTTP客户端
var httpClient = &http.Client{
	Timeout: APITimeout,
}

func main() {
	// Create a new MCP server
	s := server.NewMCPServer(
		"Xiaohu Jobs MCP",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
	)

	// Add tools
	addTools(s)

	// Add resources
	addResources(s)

	// Add prompts
	//addPrompts(s)

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func addTools(s *server.MCPServer) {
	// List jobs tool
	s.AddTool(mcp.NewTool("list_jobs",
		mcp.WithDescription("List all jobs with optional filtering"),
		mcp.WithNumber("page",
			mcp.Description("Page number for pagination"),
			mcp.DefaultNumber(1),
		),
		mcp.WithNumber("size",
			mcp.Description("Number of jobs per page"),
			mcp.DefaultNumber(10),
		),
		mcp.WithString("state",
			mcp.Description("Filter by job state (0等待 1运行中 2已停止)"),
			mcp.DefaultString(""),
		),
		mcp.WithString("name",
			mcp.Description("Filter by job name"),
			mcp.DefaultString(""),
		),
	), listJobsTool)

	// Get job tool
	s.AddTool(mcp.NewTool("get_job",
		mcp.WithDescription("Get details of a specific job"),
		mcp.WithString("job_id",
			mcp.Description("ID of the job to retrieve"),
			mcp.Required(),
		),
	), getJobTool)

	// Create job tool
	s.AddTool(mcp.NewTool("create_job",
		mcp.WithDescription("Create a new job - supports all 3 task types: HTTP, command, and function"),
		mcp.WithString("name",
			mcp.Description("Name of the job"),
			mcp.Required(),
		),
		mcp.WithString("command",
			mcp.Description("Command to execute - can be overridden by specific mode parameters"),
			mcp.DefaultString(""),
		),
		mcp.WithString("cron_expr",
			mcp.Description("Schedule in cron format"),
			mcp.DefaultString(""),
		),
		mcp.WithString("mode",
			mcp.Description("Execution mode: command, http, func"),
			mcp.DefaultString("command"),
		),
		mcp.WithNumber("max_run_count",
			mcp.Description("Maximum number of times to run"),
			mcp.DefaultNumber(0),
		),
		mcp.WithNumber("allow_mode",
			mcp.Description("Concurrency mode: 0=parallel, 1=skip if running, 2=queue if running"),
			mcp.DefaultNumber(0),
		),
		mcp.WithString("desc",
			mcp.Description("Description of the job"),
			mcp.DefaultString(""),
		),
		// HTTP任务专用参数
		mcp.WithString("url",
			mcp.Description("HTTP URL (for http mode) - overrides command parameter"),
			mcp.DefaultString(""),
		),
		mcp.WithString("http_mode",
			mcp.Description("HTTP method: GET, POST, PUT, DELETE (default: GET)"),
			mcp.DefaultString("GET"),
		),
		mcp.WithString("proxy",
			mcp.Description("HTTP proxy address (e.g., http://proxy.example.com:8080 or socks5://proxy.example.com:1080)"),
			mcp.DefaultString(""),
		),
		mcp.WithString("headers",
			mcp.Description("HTTP headers (format: key1:value1|||key2:value2)"),
			mcp.DefaultString(""),
		),
		mcp.WithString("data",
			mcp.Description("POST data for HTTP requests"),
			mcp.DefaultString(""),
		),
		mcp.WithString("cookies",
			mcp.Description("HTTP cookies string"),
			mcp.DefaultString(""),
		),
		mcp.WithString("result",
			mcp.Description("Custom success result string to match in response"),
			mcp.DefaultString(""),
		),
		mcp.WithNumber("timeout",
			mcp.Description("HTTP timeout in seconds"),
			mcp.DefaultNumber(60),
		),
		// 命令任务专用参数
		mcp.WithString("cmd",
			mcp.Description("Command to execute (for command mode) - overrides command parameter"),
			mcp.DefaultString(""),
		),
		mcp.WithString("workdir",
			mcp.Description("Working directory for command execution"),
			mcp.DefaultString(""),
		),
		mcp.WithString("env",
			mcp.Description("Environment variables (format: key1=value1|||key2=value2)"),
			mcp.DefaultString(""),
		),
		// 函数任务专用参数
		mcp.WithString("func_name",
			mcp.Description("Function name to call (for func mode) - overrides command parameter"),
			mcp.DefaultString(""),
		),
		mcp.WithString("arg",
			mcp.Description("Function argument string"),
			mcp.DefaultString(""),
		),
		// 通用参数
		mcp.WithNumber("times",
			mcp.Description("Number of times to execute (for http/command/func modes)"),
			mcp.DefaultNumber(0),
		),
		mcp.WithNumber("interval",
			mcp.Description("Interval between executions in seconds (for http/command/func modes)"),
			mcp.DefaultNumber(0),
		),
	), createJobTool)

	// Update job tool
	s.AddTool(mcp.NewTool("update_job",
		mcp.WithDescription("Update an existing job - supports all 3 task types: HTTP, command, and function"),
		mcp.WithString("job_id",
			mcp.Description("ID of the job to update"),
			mcp.Required(),
		),
		mcp.WithString("name",
			mcp.Description("New name of the job"),
		),
		mcp.WithString("command",
			mcp.Description("New command to execute - can be overridden by specific mode parameters"),
		),
		mcp.WithString("cron_expr",
			mcp.Description("New schedule in cron format"),
		),
		mcp.WithString("mode",
			mcp.Description("New execution mode: command, http, func"),
		),
		mcp.WithNumber("max_run_count",
			mcp.Description("New maximum run count"),
		),
		mcp.WithNumber("allow_mode",
			mcp.Description("New concurrency mode: 0=parallel, 1=skip if running, 2=queue if running"),
		),
		mcp.WithString("desc",
			mcp.Description("New description of the job"),
		),
		// HTTP任务专用参数
		mcp.WithString("url",
			mcp.Description("HTTP URL (for http mode) - overrides command parameter"),
		),
		mcp.WithString("http_mode",
			mcp.Description("HTTP method: GET, POST, PUT, DELETE (default: GET)"),
		),
		mcp.WithString("proxy",
			mcp.Description("HTTP proxy address (e.g., http://proxy.example.com:8080 or socks5://proxy.example.com:1080)"),
		),
		mcp.WithString("headers",
			mcp.Description("HTTP headers (format: key1:value1|||key2:value2)"),
		),
		mcp.WithString("data",
			mcp.Description("POST data for HTTP requests"),
		),
		mcp.WithString("cookies",
			mcp.Description("HTTP cookies string"),
		),
		mcp.WithString("result",
			mcp.Description("Custom success result string to match in response"),
		),
		mcp.WithNumber("timeout",
			mcp.Description("HTTP timeout in seconds"),
		),
		// 命令任务专用参数
		mcp.WithString("cmd",
			mcp.Description("Command to execute (for command mode) - overrides command parameter"),
		),
		mcp.WithString("workdir",
			mcp.Description("Working directory for command execution"),
		),
		mcp.WithString("env",
			mcp.Description("Environment variables (format: key1=value1|||key2=value2)"),
		),
		// 函数任务专用参数
		mcp.WithString("func_name",
			mcp.Description("Function name to call (for func mode) - overrides command parameter"),
		),
		mcp.WithString("arg",
			mcp.Description("Function argument string"),
		),
		// 通用参数
		mcp.WithNumber("times",
			mcp.Description("Number of times to execute (for http/command/func modes)"),
		),
		mcp.WithNumber("interval",
			mcp.Description("Interval between executions in seconds (for http/command/func modes)"),
		),
	), updateJobTool)

	// Delete job tool
	s.AddTool(mcp.NewTool("delete_job",
		mcp.WithDescription("Delete a job"),
		mcp.WithString("job_id",
			mcp.Description("ID of the job to delete"),
			mcp.Required(),
		),
	), deleteJobTool)

	// Start job tool
	s.AddTool(mcp.NewTool("start_job",
		mcp.WithDescription("Start a job"),
		mcp.WithString("job_id",
			mcp.Description("ID of the job to start"),
			mcp.Required(),
		),
	), startJobTool)

	// Stop job tool
	s.AddTool(mcp.NewTool("stop_job",
		mcp.WithDescription("Stop a running job"),
		mcp.WithString("job_id",
			mcp.Description("ID of the job to stop"),
			mcp.Required(),
		),
	), stopJobTool)

	// Get job logs tool
	s.AddTool(mcp.NewTool("get_job_logs",
		mcp.WithDescription("Get logs for a specific job"),
		mcp.WithString("job_id",
			mcp.Description("ID of the job"),
			mcp.Required(),
		),
		mcp.WithNumber("limit",
			mcp.Description("Number of log entries to return"),
			mcp.DefaultNumber(10),
		),
	), getJobLogsTool)

	// Run job manually tool
	s.AddTool(mcp.NewTool("run_job",
		mcp.WithDescription("Manually run a job immediately"),
		mcp.WithString("job_id",
			mcp.Description("ID of the job to run manually"),
			mcp.Required(),
		),
	), runJobTool)

	// Restart job tool
	s.AddTool(mcp.NewTool("restart_job",
		mcp.WithDescription("Restart a job"),
		mcp.WithString("job_id",
			mcp.Description("ID of the job to restart"),
			mcp.Required(),
		),
	), restartJobTool)

	// Get system logs tool
	s.AddTool(mcp.NewTool("get_system_logs",
		mcp.WithDescription("Get system logs"),
		mcp.WithString("date",
			mcp.Description("Date to query logs (format: YYYY-MM-DD)"),
			mcp.DefaultString(""),
		),
		mcp.WithNumber("page",
			mcp.Description("Page number for pagination"),
			mcp.DefaultNumber(1),
		),
		mcp.WithNumber("size",
			mcp.Description("Number of log entries per page"),
			mcp.DefaultNumber(10),
		),
	), getSystemLogsTool)

	// Get job scheduler status tool
	s.AddTool(mcp.NewTool("get_scheduler_status",
		mcp.WithDescription("Get job scheduler status"),
	), getSchedulerStatusTool)

	// Start all jobs tool
	s.AddTool(mcp.NewTool("start_all_jobs",
		mcp.WithDescription("Start all jobs in the scheduler"),
	), startAllJobsTool)

	// Stop all jobs tool
	s.AddTool(mcp.NewTool("stop_all_jobs",
		mcp.WithDescription("Stop all jobs in the scheduler"),
	), stopAllJobsTool)

	// Get job functions tool
	s.AddTool(mcp.NewTool("get_job_functions",
		mcp.WithDescription("Get available job functions"),
	), getJobFunctionsTool)

	// Get jobs configuration tool
	s.AddTool(mcp.NewTool("get_jobs_config",
		mcp.WithDescription("Get jobs system configuration"),
	), getJobsConfigTool)

	// Reload configuration tool
	s.AddTool(mcp.NewTool("reload_config",
		mcp.WithDescription("Reload system configuration"),
	), reloadConfigTool)

	// Clear job logs tool
	s.AddTool(mcp.NewTool("clear_job_logs",
		mcp.WithDescription("Clear job logs"),
	), clearJobLogsTool)

	// Get IP control status tool
	s.AddTool(mcp.NewTool("get_ip_control_status",
		mcp.WithDescription("Get IP control status"),
	), getIPControlStatusTool)

	// Add IP to whitelist tool
	s.AddTool(mcp.NewTool("add_ip_whitelist",
		mcp.WithDescription("Add IP to whitelist"),
		mcp.WithString("ip",
			mcp.Description("IP address to add to whitelist"),
			mcp.Required(),
		),
	), addIPWhitelistTool)

	// Remove IP from whitelist tool
	s.AddTool(mcp.NewTool("remove_ip_whitelist",
		mcp.WithDescription("Remove IP from whitelist"),
		mcp.WithString("ip",
			mcp.Description("IP address to remove from whitelist"),
			mcp.Required(),
		),
	), removeIPWhitelistTool)

	// Add IP to blacklist tool
	s.AddTool(mcp.NewTool("add_ip_blacklist",
		mcp.WithDescription("Add IP to blacklist"),
		mcp.WithString("ip",
			mcp.Description("IP address to add to blacklist"),
			mcp.Required(),
		),
	), addIPBlacklistTool)

	// Remove IP from blacklist tool
	s.AddTool(mcp.NewTool("remove_ip_blacklist",
		mcp.WithDescription("Remove IP from blacklist"),
		mcp.WithString("ip",
			mcp.Description("IP address to remove from blacklist"),
			mcp.Required(),
		),
	), removeIPBlacklistTool)

	// Get job execution details tool
	s.AddTool(mcp.NewTool("get_job_executions",
		mcp.WithDescription("Get job execution details and history"),
		mcp.WithString("exec_id",
			mcp.Description("Execution ID to query"),
			mcp.Required(),
		),
		mcp.WithString("job_id",
			mcp.Description("Job ID corresponding to the execution"),
			mcp.Required(),
		),
	), getJobExecutionsTool)

	// Get system log switch status tool
	s.AddTool(mcp.NewTool("get_log_switch_status",
		mcp.WithDescription("Get system log switch status"),
	), getLogSwitchStatusTool)

	// Get scheduler tasks tool
	s.AddTool(mcp.NewTool("get_scheduler_tasks",
		mcp.WithDescription("Get scheduler tasks and their status"),
	), getSchedulerTasksTool)

	// Calibrate job list tool
	s.AddTool(mcp.NewTool("calibrate_job_list",
		mcp.WithDescription("Calibrate and synchronize the job list"),
	), calibrateJobListTool)
}

func addResources(s *server.MCPServer) {
	// Health check resource
	s.AddResource(mcp.NewResource("xiaohu://health",
		"System Health",
		mcp.WithMIMEType("application/json"),
	), handleHealthResource)

	// Jobs overview resource
	s.AddResource(mcp.NewResource("xiaohu://jobs/overview",
		"Jobs Overview",
		mcp.WithMIMEType("application/json"),
	), handleJobsOverviewResource)

	// System config resource
	s.AddResource(mcp.NewResource("xiaohu://config",
		"System Configuration",
		mcp.WithMIMEType("application/json"),
	), handleConfigResource)
}

// HTTP请求辅助函数
func makeAPIRequest(method, endpoint string, body io.Reader) (*http.Response, error) {
	url := APIBaseURL + endpoint
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return httpClient.Do(req)
}

// 状态映射函数
func mapStateToStatus(state int) string {
	switch state {
	case 0:
		return "waiting"
	case 1:
		return "running"
	case 2:
		return "stopped"
	default:
		return "unknown"
	}
}

// Tool handlers
func listJobsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page := int(request.GetFloat("page", 1))
	size := int(request.GetFloat("size", 10))
	state := request.GetString("state", "")
	name := request.GetString("name", "")

	// 构建查询参数
	endpoint := fmt.Sprintf("/jobs/list?page=%d&size=%d", page, size)
	if state != "" {
		endpoint += "&state=" + state
	}
	if name != "" {
		endpoint += "&name=" + name
	}

	resp, err := makeAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var pageResp PageResponse
	if err := json.NewDecoder(resp.Body).Decode(&pageResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if pageResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + pageResp.Msg), nil
	}

	// 转换数据格式
	var jobs []map[string]interface{}
	for _, job := range pageResp.Data {
		jobs = append(jobs, map[string]interface{}{
			"id":            strconv.Itoa(int(job.ID)),
			"name":          job.Name,
			"desc":          job.Desc,
			"command":       job.Command,
			"cron_expr":     job.CronExpr,
			"mode":          job.Mode,
			"status":        mapStateToStatus(job.State),
			"state":         job.State,
			"run_count":     job.RunCount,
			"max_run_count": job.MaxRunCount,
			"allow_mode":    job.AllowMode,
			"created_at":    job.CreatedAt.Format(time.RFC3339),
			"updated_at":    job.UpdatedAt.Format(time.RFC3339),
		})
	}

	result := map[string]interface{}{
		"jobs":     jobs,
		"total":    pageResp.Total,
		"page":     pageResp.Page,
		"size":     pageResp.Size,
		"pages":    pageResp.Pages,
		"has_more": int(pageResp.Page) < int(pageResp.Pages),
	}

	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonData)), nil
}

func getJobTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobID := request.GetString("job_id", "")
	if jobID == "" {
		return mcp.NewToolResultError("job_id is required"), nil
	}

	endpoint := fmt.Sprintf("/jobs/read?id=%s", jobID)
	resp, err := makeAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	// 转换数据格式
	jobData, _ := json.Marshal(apiResp.Data)
	var job Job
	json.Unmarshal(jobData, &job)

	result := map[string]interface{}{
		"id":            strconv.Itoa(int(job.ID)),
		"name":          job.Name,
		"desc":          job.Desc,
		"command":       job.Command,
		"cron_expr":     job.CronExpr,
		"mode":          job.Mode,
		"status":        mapStateToStatus(job.State),
		"state":         job.State,
		"run_count":     job.RunCount,
		"max_run_count": job.MaxRunCount,
		"allow_mode":    job.AllowMode,
		"created_at":    job.CreatedAt.Format(time.RFC3339),
		"updated_at":    job.UpdatedAt.Format(time.RFC3339),
	}

	jsonData, _ := json.MarshalIndent(result, "", "  ")
	return mcp.NewToolResultText(string(jsonData)), nil
}

// 更新createJobTool函数，支持所有3种任务类型的详细配置
func createJobTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := request.GetString("name", "")
	command := request.GetString("command", "")
	cronExpr := request.GetString("cron_expr", "")
	mode := request.GetString("mode", "command")
	maxRunCount := int(request.GetFloat("max_run_count", 0))
	allowMode := int(request.GetFloat("allow_mode", 0))
	desc := request.GetString("desc", "")

	// 根据任务类型处理详细配置，与前端Jobs.vue保持一致
	if mode == "http" {
		// HTTP任务配置
		url := request.GetString("url", "")
		httpMode := request.GetString("http_mode", "GET")
		proxy := request.GetString("proxy", "")
		headers := request.GetString("headers", "")
		data := request.GetString("data", "")
		cookies := request.GetString("cookies", "")
		result := request.GetString("result", "")
		times := int(request.GetFloat("times", 0))
		interval := int(request.GetFloat("interval", 0))
		timeout := int(request.GetFloat("timeout", 60))

		// 构建HTTP任务配置字符串，与前端保持一致
		var config strings.Builder

		if url != "" {
			config.WriteString("【url】" + url + "\n")
		}

		if httpMode != "" && strings.ToUpper(httpMode) != "GET" {
			config.WriteString("【mode】" + strings.ToUpper(httpMode) + "\n")
		}

		if headers != "" {
			config.WriteString("【headers】" + headers + "\n")
		}

		if data != "" {
			config.WriteString("【data】" + data + "\n")
		}

		if cookies != "" {
			config.WriteString("【cookies】" + cookies + "\n")
		}

		if proxy != "" {
			config.WriteString("【proxy】" + proxy + "\n")
		}

		if times > 0 {
			config.WriteString("【times】" + strconv.Itoa(times) + "\n")
		}

		if interval > 0 {
			config.WriteString("【interval】" + strconv.Itoa(interval) + "\n")
		}

		if result != "" {
			config.WriteString("【result】" + result + "\n")
		}

		if timeout != 60 {
			config.WriteString("【timeout】" + strconv.Itoa(timeout) + "\n")
		}

		if config.Len() > 0 {
			command = config.String()
		}

	} else if mode == "command" {
		// 命令任务配置
		cmd := request.GetString("cmd", "")
		workdir := request.GetString("workdir", "")
		env := request.GetString("env", "")
		times := int(request.GetFloat("times", 0))
		interval := int(request.GetFloat("interval", 0))
		timeout := int(request.GetFloat("timeout", 30))

		// 构建命令任务配置字符串，与前端保持一致
		var config strings.Builder

		if cmd != "" {
			config.WriteString("【command】" + cmd + "\n")
		}

		if workdir != "" {
			config.WriteString("【workdir】" + workdir + "\n")
		}

		if env != "" {
			config.WriteString("【env】" + env + "\n")
		}

		if times > 0 {
			config.WriteString("【times】" + strconv.Itoa(times) + "\n")
		}

		if interval > 0 {
			config.WriteString("【interval】" + strconv.Itoa(interval) + "\n")
		}

		if timeout != 30 {
			config.WriteString("【timeout】" + strconv.Itoa(timeout) + "\n")
		}

		if config.Len() > 0 {
			command = config.String()
		}

	} else if mode == "func" {
		// 函数任务配置
		funcName := request.GetString("func_name", "")
		arg := request.GetString("arg", "")
		times := int(request.GetFloat("times", 0))
		interval := int(request.GetFloat("interval", 0))

		// 构建函数任务配置字符串，与前端保持一致
		var config strings.Builder

		if funcName != "" {
			config.WriteString("【name】" + funcName + "\n")
		}

		if arg != "" {
			config.WriteString("【arg】" + arg + "\n")
		}

		if times > 0 {
			config.WriteString("【times】" + strconv.Itoa(times) + "\n")
		}

		if interval > 0 {
			config.WriteString("【interval】" + strconv.Itoa(interval) + "\n")
		}

		if config.Len() > 0 {
			command = config.String()
		}
	}

	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	// 根据任务类型验证必需参数
	if mode == "http" {
		url := request.GetString("url", "")
		if url == "" && command == "" {
			return mcp.NewToolResultError("url or command is required for http mode"), nil
		}
	} else if mode == "command" {
		cmd := request.GetString("cmd", "")
		if cmd == "" && command == "" {
			return mcp.NewToolResultError("cmd or command is required for command mode"), nil
		}
	} else if mode == "func" {
		funcName := request.GetString("func_name", "")
		if funcName == "" && command == "" {
			return mcp.NewToolResultError("func_name or command is required for func mode"), nil
		}
	} else {
		if command == "" {
			return mcp.NewToolResultError("command is required"), nil
		}
	}

	jobData := map[string]interface{}{
		"name":          name,
		"command":       command,
		"cron_expr":     cronExpr,
		"mode":          mode,
		"max_run_count": maxRunCount,
		"allow_mode":    allowMode,
		"state":         0,
		"desc":          desc,
	}

	jsonData, _ := json.Marshal(jobData)
	resp, err := makeAPIRequest("POST", "/jobs/add", bytes.NewBuffer(jsonData))
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Job created successfully: %s", apiResp.Msg)), nil
}

// 更新updateJobTool函数，支持所有3种任务类型的详细配置
func updateJobTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobID := request.GetString("job_id", "")
	if jobID == "" {
		return mcp.NewToolResultError("job_id is required"), nil
	}

	// 将字符串job_id转换为uint
	jobIDUint, err := strconv.ParseUint(jobID, 10, 32)
	if err != nil {
		return mcp.NewToolResultError("Invalid job_id format: " + err.Error()), nil
	}

	updateData := map[string]interface{}{
		"id": uint(jobIDUint),
	}

	mode := request.GetString("mode", "")

	// 根据任务类型处理详细配置，与前端Jobs.vue保持一致
	currentMode := mode
	if currentMode == "" {
		// 如果没有指定mode，从现有任务获取
		currentMode = "command" // 默认
	}

	if currentMode == "http" {
		// HTTP任务配置
		url := request.GetString("url", "")
		httpMode := request.GetString("http_mode", "")
		proxy := request.GetString("proxy", "")
		headers := request.GetString("headers", "")
		data := request.GetString("data", "")
		cookies := request.GetString("cookies", "")
		result := request.GetString("result", "")
		times := int(request.GetFloat("times", 0))
		interval := int(request.GetFloat("interval", 0))
		timeout := int(request.GetFloat("timeout", 0))

		// 构建HTTP任务配置字符串，与前端保持一致
		var config strings.Builder

		if url != "" {
			config.WriteString("【url】" + url + "\n")
		}

		if httpMode != "" && strings.ToUpper(httpMode) != "GET" {
			config.WriteString("【mode】" + strings.ToUpper(httpMode) + "\n")
		}

		if headers != "" {
			config.WriteString("【headers】" + headers + "\n")
		}

		if data != "" {
			config.WriteString("【data】" + data + "\n")
		}

		if cookies != "" {
			config.WriteString("【cookies】" + cookies + "\n")
		}

		if proxy != "" {
			config.WriteString("【proxy】" + proxy + "\n")
		}

		if times > 0 {
			config.WriteString("【times】" + strconv.Itoa(times) + "\n")
		}

		if interval > 0 {
			config.WriteString("【interval】" + strconv.Itoa(interval) + "\n")
		}

		if result != "" {
			config.WriteString("【result】" + result + "\n")
		}

		if timeout > 0 {
			config.WriteString("【timeout】" + strconv.Itoa(timeout) + "\n")
		}

		// 使用构建的配置字符串作为command
		if config.Len() > 0 {
			updateData["command"] = config.String()
		}

	} else if currentMode == "command" {
		// 命令任务配置
		cmd := request.GetString("cmd", "")
		workdir := request.GetString("workdir", "")
		env := request.GetString("env", "")
		times := int(request.GetFloat("times", 0))
		interval := int(request.GetFloat("interval", 0))
		timeout := int(request.GetFloat("timeout", 0))

		// 构建命令任务配置字符串，与前端保持一致
		var config strings.Builder

		if cmd != "" {
			config.WriteString("【command】" + cmd + "\n")
		}

		if workdir != "" {
			config.WriteString("【workdir】" + workdir + "\n")
		}

		if env != "" {
			config.WriteString("【env】" + env + "\n")
		}

		if times > 0 {
			config.WriteString("【times】" + strconv.Itoa(times) + "\n")
		}

		if interval > 0 {
			config.WriteString("【interval】" + strconv.Itoa(interval) + "\n")
		}

		if timeout > 0 {
			config.WriteString("【timeout】" + strconv.Itoa(timeout) + "\n")
		}

		// 使用构建的配置字符串作为command
		if config.Len() > 0 {
			updateData["command"] = config.String()
		}

	} else if currentMode == "func" {
		// 函数任务配置
		funcName := request.GetString("func_name", "")
		arg := request.GetString("arg", "")
		times := int(request.GetFloat("times", 0))
		interval := int(request.GetFloat("interval", 0))

		// 构建函数任务配置字符串，与前端保持一致
		var config strings.Builder

		if funcName != "" {
			config.WriteString("【name】" + funcName + "\n")
		}

		if arg != "" {
			config.WriteString("【arg】" + arg + "\n")
		}

		if times > 0 {
			config.WriteString("【times】" + strconv.Itoa(times) + "\n")
		}

		if interval > 0 {
			config.WriteString("【interval】" + strconv.Itoa(interval) + "\n")
		}

		// 使用构建的配置字符串作为command
		if config.Len() > 0 {
			updateData["command"] = config.String()
		}
	}

	if name := request.GetString("name", ""); name != "" {
		updateData["name"] = name
	}
	if command := request.GetString("command", ""); command != "" {
		updateData["command"] = command
	}
	if cronExpr := request.GetString("cron_expr", ""); cronExpr != "" {
		updateData["cron_expr"] = cronExpr
	}
	if mode != "" {
		updateData["mode"] = mode
	}
	if maxRunCount := int(request.GetFloat("max_run_count", -1)); maxRunCount >= 0 {
		updateData["max_run_count"] = maxRunCount
	}
	if allowMode := int(request.GetFloat("allow_mode", -1)); allowMode >= 0 {
		updateData["allow_mode"] = allowMode
	}
	if desc := request.GetString("desc", ""); desc != "" {
		updateData["desc"] = desc
	}

	jsonData, _ := json.Marshal(updateData)
	resp, err := makeAPIRequest("POST", "/jobs/edit", bytes.NewBuffer(jsonData))
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Job %s updated successfully: %s", jobID, apiResp.Msg)), nil
}

func deleteJobTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobID := request.GetString("job_id", "")
	if jobID == "" {
		return mcp.NewToolResultError("job_id is required"), nil
	}

	// 将字符串job_id转换为uint
	jobIDUint, err := strconv.ParseUint(jobID, 10, 32)
	if err != nil {
		return mcp.NewToolResultError("Invalid job_id format: " + err.Error()), nil
	}

	deleteData := map[string]interface{}{
		"id": uint(jobIDUint),
	}

	jsonData, _ := json.Marshal(deleteData)
	resp, err := makeAPIRequest("POST", "/jobs/del", bytes.NewBuffer(jsonData))
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Job %s deleted successfully: %s", jobID, apiResp.Msg)), nil
}

func startJobTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobID := request.GetString("job_id", "")
	if jobID == "" {
		return mcp.NewToolResultError("job_id is required"), nil
	}

	// 将字符串job_id转换为uint
	jobIDUint, err := strconv.ParseUint(jobID, 10, 32)
	if err != nil {
		return mcp.NewToolResultError("Invalid job_id format: " + err.Error()), nil
	}

	startData := map[string]interface{}{
		"id": uint(jobIDUint),
	}

	jsonData, _ := json.Marshal(startData)
	resp, err := makeAPIRequest("POST", "/jobs/restart", bytes.NewBuffer(jsonData))
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Job %s started successfully: %s", jobID, apiResp.Msg)), nil
}

func stopJobTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobID := request.GetString("job_id", "")
	if jobID == "" {
		return mcp.NewToolResultError("job_id is required"), nil
	}

	// 将字符串job_id转换为uint
	jobIDUint, err := strconv.ParseUint(jobID, 10, 32)
	if err != nil {
		return mcp.NewToolResultError("Invalid job_id format: " + err.Error()), nil
	}

	stopData := map[string]interface{}{
		"id": uint(jobIDUint),
	}

	jsonData, _ := json.Marshal(stopData)
	resp, err := makeAPIRequest("POST", "/jobs/stop", bytes.NewBuffer(jsonData))
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Job %s stopped successfully: %s", jobID, apiResp.Msg)), nil
}

func getJobLogsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobID := request.GetString("job_id", "")
	limit := int(request.GetFloat("limit", 10))

	if jobID == "" {
		return mcp.NewToolResultError("job_id is required"), nil
	}

	// 将字符串job_id转换为uint
	jobIDUint, err := strconv.ParseUint(jobID, 10, 32)
	if err != nil {
		return mcp.NewToolResultError("Invalid job_id format: " + err.Error()), nil
	}

	logData := map[string]interface{}{
		"id":    uint(jobIDUint),
		"limit": limit,
	}

	jsonData, _ := json.Marshal(logData)
	resp, err := makeAPIRequest("POST", "/jobs/logs", bytes.NewBuffer(jsonData))
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	logsData, _ := json.MarshalIndent(apiResp.Data, "", "  ")
	return mcp.NewToolResultText(string(logsData)), nil
}

// Resource handlers
func handleHealthResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	resp, err := makeAPIRequest("GET", "/jobs/health", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to API: %v", err)
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %v", err)
	}

	if apiResp.Code != 200 {
		return nil, fmt.Errorf("API error: %s", apiResp.Msg)
	}

	healthData, _ := json.MarshalIndent(apiResp.Data, "", "  ")
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "xiaohu://health",
			MIMEType: "application/json",
			Text:     string(healthData),
		},
	}, nil
}

func handleJobsOverviewResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// 获取任务列表来生成概览
	resp, err := makeAPIRequest("GET", "/jobs/list?page=1&size=1000", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to API: %v", err)
	}
	defer resp.Body.Close()

	var pageResp PageResponse
	if err := json.NewDecoder(resp.Body).Decode(&pageResp); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %v", err)
	}

	if pageResp.Code != 200 {
		return nil, fmt.Errorf("API error: %s", pageResp.Msg)
	}

	// 统计任务状态
	var totalJobs, runningJobs, stoppedJobs, waitingJobs int
	for _, job := range pageResp.Data {
		totalJobs++
		switch job.State {
		case 0:
			waitingJobs++
		case 1:
			runningJobs++
		case 2:
			stoppedJobs++
		}
	}

	overview := map[string]interface{}{
		"total_jobs":   totalJobs,
		"running_jobs": runningJobs,
		"stopped_jobs": stoppedJobs,
		"waiting_jobs": waitingJobs,
		"last_updated": time.Now().Format(time.RFC3339),
	}

	overviewData, _ := json.MarshalIndent(overview, "", "  ")
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "xiaohu://jobs/overview",
			MIMEType: "application/json",
			Text:     string(overviewData),
		},
	}, nil
}

func handleConfigResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	config := map[string]interface{}{
		"server_name": "Xiaohu Jobs System",
		"version":     "1.0.0",
		"api_base":    APIBaseURL,
		"features":    []string{"job_management", "monitoring", "logging", "cron_scheduling"},
		"timezone":    "Local",
		"log_level":   "INFO",
		"mcp_version": "1.0.0",
	}

	configData, _ := json.MarshalIndent(config, "", "  ")
	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      "xiaohu://config",
			MIMEType: "application/json",
			Text:     string(configData),
		},
	}, nil
}

// Tool handlers for new functions

func runJobTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobID := request.GetString("job_id", "")
	if jobID == "" {
		return mcp.NewToolResultError("job_id is required"), nil
	}

	// 将字符串job_id转换为uint
	jobIDUint, err := strconv.ParseUint(jobID, 10, 32)
	if err != nil {
		return mcp.NewToolResultError("Invalid job_id format: " + err.Error()), nil
	}

	runData := map[string]interface{}{
		"id": uint(jobIDUint),
	}

	jsonData, _ := json.Marshal(runData)
	resp, err := makeAPIRequest("POST", "/jobs/run", bytes.NewBuffer(jsonData))
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	// 解析响应数据中的exec_id
	var resultData map[string]interface{}
	if dataBytes, err := json.Marshal(apiResp.Data); err == nil {
		json.Unmarshal(dataBytes, &resultData)
	}

	execID := ""
	if resultData != nil {
		if id, ok := resultData["exec_id"].(string); ok {
			execID = id
		}
	}

	response := map[string]interface{}{
		"job_id":  jobID,
		"status":  "success",
		"message": apiResp.Msg,
		"exec_id": execID,
	}

	jsonResponse, _ := json.MarshalIndent(response, "", "  ")
	return mcp.NewToolResultText(string(jsonResponse)), nil
}

func restartJobTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobID := request.GetString("job_id", "")
	if jobID == "" {
		return mcp.NewToolResultError("job_id is required"), nil
	}

	// 将字符串job_id转换为uint
	jobIDUint, err := strconv.ParseUint(jobID, 10, 32)
	if err != nil {
		return mcp.NewToolResultError("Invalid job_id format: " + err.Error()), nil
	}

	restartData := map[string]interface{}{
		"id": uint(jobIDUint),
	}

	jsonData, _ := json.Marshal(restartData)
	resp, err := makeAPIRequest("POST", "/jobs/restart", bytes.NewBuffer(jsonData))
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Job %s restarted successfully: %s", jobID, apiResp.Msg)), nil
}

func getSystemLogsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	date := request.GetString("date", "")
	page := int(request.GetFloat("page", 1))
	size := int(request.GetFloat("size", 10))

	endpoint := fmt.Sprintf("/jobs/zapLogs?page=%d&size=%d", page, size)
	if date != "" {
		endpoint += "&date=" + date
	}

	resp, err := makeAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	logsData, _ := json.MarshalIndent(apiResp.Data, "", "  ")
	return mcp.NewToolResultText(string(logsData)), nil
}

func getSchedulerStatusTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := makeAPIRequest("GET", "/jobs/jobState", nil)
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	statusData, _ := json.MarshalIndent(apiResp.Data, "", "  ")
	return mcp.NewToolResultText(string(statusData)), nil
}

func getSchedulerTasksTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := makeAPIRequest("GET", "/jobs/scheduler", nil)
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	tasksData, _ := json.MarshalIndent(apiResp.Data, "", "  ")
	return mcp.NewToolResultText(string(tasksData)), nil
}

func calibrateJobListTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := makeAPIRequest("POST", "/jobs/checkJob", nil)
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Job list calibrated successfully: %s", apiResp.Msg)), nil
}

func startAllJobsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := makeAPIRequest("POST", "/jobs/runAll", nil)
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("All jobs started successfully: %s", apiResp.Msg)), nil
}

func stopAllJobsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := makeAPIRequest("POST", "/jobs/stopAll", nil)
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("All jobs stopped successfully: %s", apiResp.Msg)), nil
}

func getJobFunctionsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := makeAPIRequest("GET", "/jobs/functions", nil)
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	functionsData, _ := json.MarshalIndent(apiResp.Data, "", "  ")
	return mcp.NewToolResultText(string(functionsData)), nil
}

func getJobsConfigTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := makeAPIRequest("GET", "/jobs/config", nil)
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	configData, _ := json.MarshalIndent(apiResp.Data, "", "  ")
	return mcp.NewToolResultText(string(configData)), nil
}

func reloadConfigTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := makeAPIRequest("POST", "/jobs/reload-config", nil)
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Configuration reloaded successfully: %s", apiResp.Msg)), nil
}

func clearJobLogsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := makeAPIRequest("POST", "/jobs/logs/clear", nil)
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Job logs cleared successfully: %s", apiResp.Msg)), nil
}

func getIPControlStatusTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := makeAPIRequest("GET", "/jobs/ip-control/status", nil)
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	statusData, _ := json.MarshalIndent(apiResp.Data, "", "  ")
	return mcp.NewToolResultText(string(statusData)), nil
}

func addIPWhitelistTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ip := request.GetString("ip", "")
	if ip == "" {
		return mcp.NewToolResultError("ip is required"), nil
	}

	ipData := map[string]interface{}{
		"ip": ip,
	}

	jsonData, _ := json.Marshal(ipData)
	resp, err := makeAPIRequest("POST", "/jobs/ip-control/whitelist/add", bytes.NewBuffer(jsonData))
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("IP %s added to whitelist successfully: %s", ip, apiResp.Msg)), nil
}

func removeIPWhitelistTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ip := request.GetString("ip", "")
	if ip == "" {
		return mcp.NewToolResultError("ip is required"), nil
	}

	ipData := map[string]interface{}{
		"ip": ip,
	}

	jsonData, _ := json.Marshal(ipData)
	resp, err := makeAPIRequest("POST", "/jobs/ip-control/whitelist/remove", bytes.NewBuffer(jsonData))
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("IP %s removed from whitelist successfully: %s", ip, apiResp.Msg)), nil
}

func addIPBlacklistTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ip := request.GetString("ip", "")
	if ip == "" {
		return mcp.NewToolResultError("ip is required"), nil
	}

	ipData := map[string]interface{}{
		"ip": ip,
	}

	jsonData, _ := json.Marshal(ipData)
	resp, err := makeAPIRequest("POST", "/jobs/ip-control/blacklist/add", bytes.NewBuffer(jsonData))
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("IP %s added to blacklist successfully: %s", ip, apiResp.Msg)), nil
}

func removeIPBlacklistTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	ip := request.GetString("ip", "")
	if ip == "" {
		return mcp.NewToolResultError("ip is required"), nil
	}

	ipData := map[string]interface{}{
		"ip": ip,
	}

	jsonData, _ := json.Marshal(ipData)
	resp, err := makeAPIRequest("POST", "/jobs/ip-control/blacklist/remove", bytes.NewBuffer(jsonData))
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("IP %s removed from blacklist successfully: %s", ip, apiResp.Msg)), nil
}

func getJobExecutionsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	execID := request.GetString("exec_id", "")
	if execID == "" {
		return mcp.NewToolResultError("exec_id is required"), nil
	}

	jobID := request.GetString("job_id", "")
	if jobID == "" {
		return mcp.NewToolResultError("job_id is required"), nil
	}

	// 验证job_id是否为数字
	if _, err := strconv.Atoi(jobID); err != nil {
		return mcp.NewToolResultError("Invalid job_id format: must be a number"), nil
	}

	endpoint := fmt.Sprintf("/jobs/execs?id=%s&exec_id=%s", jobID, execID)
	resp, err := makeAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	// 读取原始响应用于调试
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return mcp.NewToolResultError("Failed to read response body: " + err.Error()), nil
	}

	var apiResp APIResponse
	if err = json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	// 确保 data 不为 nil
	if apiResp.Data == nil {
		return mcp.NewToolResultText("{}"), nil
	}

	execData, err := json.MarshalIndent(apiResp.Data, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal execution data: " + err.Error()), nil
	}

	return mcp.NewToolResultText(string(execData)), nil
}

func getLogSwitchStatusTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resp, err := makeAPIRequest("GET", "/jobs/switchState", nil)
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	switchData, _ := json.MarshalIndent(apiResp.Data, "", "  ")
	return mcp.NewToolResultText(string(switchData)), nil
}
