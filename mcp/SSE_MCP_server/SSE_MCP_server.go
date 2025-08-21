package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// API配置
var (
	APIBaseURL = getEnvWithDefault("API_BASE_URL", "http://127.0.0.1:36363")
	APITimeout = 30 * time.Second
	ServerPort = getEnvWithDefault("SERVER_PORT", ":8080") // SSE服务器端口
)

// getEnvWithDefault 获取环境变量，如果不存在则返回默认值
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

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

// 全局MCP服务器实例
var mcpServer *server.MCPServer

func main() {
	// 加载.env文件
	if err := godotenv.Load(); err != nil {
		// .env文件不存在时使用默认值，不报错
		fmt.Println("No .env file found, using default values")
	}

	// 重新加载环境变量（确保.env文件中的值生效）
	APIBaseURL = getEnvWithDefault("API_BASE_URL", "http://127.0.0.1:36363")
	ServerPort = getEnvWithDefault("SERVER_PORT", ":8080")

	// Create a new MCP server
	mcpServer = server.NewMCPServer(
		"Xiaohu Jobs MCP SSE",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithPromptCapabilities(true),
	)

	// Add tools
	addTools(mcpServer)

	// Add resources
	addResources(mcpServer)

	// 创建HTTP多路复用器
	mux := http.NewServeMux()

	// 使用官方的StreamableHTTPServer，注册/mcp端点
	httpServer := server.NewStreamableHTTPServer(mcpServer)
	mux.Handle("/mcp", httpServer)

	// 注册健康检查端点
	mux.HandleFunc("/health", handleHealth)

	// 启动HTTP服务器
	fmt.Printf("Starting SSE MCP server on port %s...\n", ServerPort)
	fmt.Printf("SSE endpoint: http://127.0.0.1%s/mcp\n", ServerPort)
	fmt.Printf("Health check: http://127.0.0.1%s/health\n", ServerPort)

	// 启动自定义HTTP服务器
	server := &http.Server{
		Addr:    ServerPort,
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

// 添加工具函数
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

	// Calibrate job list tool
	s.AddTool(mcp.NewTool("calibrate_job_list",
		mcp.WithDescription("Calibrate and synchronize the job list"),
	), calibrateJobListTool)

	// Get IP control status tool
	s.AddTool(mcp.NewTool("get_ip_control_status",
		mcp.WithDescription("Get IP control status"),
	), getIPControlStatusTool)

	// Add IP whitelist tool
	s.AddTool(mcp.NewTool("add_ip_whitelist",
		mcp.WithDescription("Add IP to whitelist"),
		mcp.WithString("ip",
			mcp.Description("IP address to add to whitelist"),
			mcp.Required(),
		),
	), addIPWhitelistTool)

	// Remove IP whitelist tool
	s.AddTool(mcp.NewTool("remove_ip_whitelist",
		mcp.WithDescription("Remove IP from whitelist"),
		mcp.WithString("ip",
			mcp.Description("IP address to remove from whitelist"),
			mcp.Required(),
		),
	), removeIPWhitelistTool)

	// Add IP blacklist tool
	s.AddTool(mcp.NewTool("add_ip_blacklist",
		mcp.WithDescription("Add IP to blacklist"),
		mcp.WithString("ip",
			mcp.Description("IP address to add to blacklist"),
			mcp.Required(),
		),
	), addIPBlacklistTool)

	// Remove IP blacklist tool
	s.AddTool(mcp.NewTool("remove_ip_blacklist",
		mcp.WithDescription("Remove IP from blacklist"),
		mcp.WithString("ip",
			mcp.Description("IP address to remove from blacklist"),
			mcp.Required(),
		),
	), removeIPBlacklistTool)

	// Get job executions tool
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

	// Get log switch status tool
	s.AddTool(mcp.NewTool("get_log_switch_status",
		mcp.WithDescription("Get system log switch status"),
	), getLogSwitchStatusTool)
}

// 添加资源函数
func addResources(s *server.MCPServer) {
	// 添加示例资源
	s.AddResource(mcp.NewResource("jobs://list", "List of all jobs",
		mcp.WithResourceDescription("Get a list of all configured jobs"),
	), getJobsResource)

	s.AddResource(mcp.NewResource("jobs://status", "System status",
		mcp.WithResourceDescription("Get the current system status"),
	), getSystemStatusResource)
}

// 资源处理函数
func getJobsResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// 调用list_jobs工具获取数据
	result, err := listJobsTool(ctx, mcp.CallToolRequest{})
	if err != nil {
		return nil, err
	}

	// 将结果转换为JSON字符串
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %v", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:  "jobs://list",
			Text: string(resultJSON),
		},
	}, nil
}

func getSystemStatusResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// 调用get_scheduler_status工具获取数据
	result, err := getSchedulerStatusTool(ctx, mcp.CallToolRequest{})
	if err != nil {
		return nil, err
	}

	// 将结果转换为JSON字符串
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %v", err)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:  "jobs://status",
			Text: string(resultJSON),
		},
	}, nil
}

// 以下是所有工具函数的实现
func listJobsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	page := int(request.GetFloat("page", 1))
	size := int(request.GetFloat("size", 10))
	state := request.GetString("state", "")
	name := request.GetString("name", "")

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

	var apiResp PageResponse
	if err = json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	jobsData, err := json.MarshalIndent(apiResp.Data, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal jobs data: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(jobsData)), nil
}

// mapStateToStatus 将状态码转换为可读状态
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

func getJobTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobID := request.GetString("job_id", "")
	if jobID == "" {
		return mcp.NewToolResultError("job_id is required"), nil
	}

	// 验证job_id是否为数字
	if _, err := strconv.Atoi(jobID); err != nil {
		return mcp.NewToolResultError("Invalid job_id format: must be a number"), nil
	}

	endpoint := fmt.Sprintf("/jobs/read?id=%s", jobID)
	resp, err := makeAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err = json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	// 转换数据格式
	jobData, err := json.Marshal(apiResp.Data)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal job data: " + err.Error()), nil
	}
	var job Job
	if err = json.Unmarshal(jobData, &job); err != nil {
		return mcp.NewToolResultError("Failed to unmarshal job data: " + err.Error()), nil
	}

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

	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal result: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(jsonData)), nil
}

func createJobTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name := request.GetString("name", "")
	if name == "" {
		return mcp.NewToolResultError("name is required"), nil
	}

	jobData := map[string]interface{}{
		"name":          name,
		"command":       request.GetString("command", ""),
		"cron_expr":     request.GetString("cron_expr", ""),
		"mode":          request.GetString("mode", "command"),
		"max_run_count": request.GetFloat("max_run_count", 0),
		"allow_mode":    request.GetFloat("allow_mode", 0),
		"desc":          request.GetString("desc", ""),
		"url":           request.GetString("url", ""),
		"http_mode":     request.GetString("http_mode", "GET"),
		"proxy":         request.GetString("proxy", ""),
		"headers":       request.GetString("headers", ""),
		"data":          request.GetString("data", ""),
		"cookies":       request.GetString("cookies", ""),
		"result":        request.GetString("result", ""),
		"timeout":       request.GetFloat("timeout", 60),
		"cmd":           request.GetString("cmd", ""),
		"workdir":       request.GetString("workdir", ""),
		"env":           request.GetString("env", ""),
		"func_name":     request.GetString("func_name", ""),
		"arg":           request.GetString("arg", ""),
		"times":         request.GetFloat("times", 0),
		"interval":      request.GetFloat("interval", 0),
	}

	// 移除空值
	for k, v := range jobData {
		if v == "" || v == 0.0 {
			delete(jobData, k)
		}
	}

	jsonData, err := json.Marshal(jobData)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal job data: " + err.Error()), nil
	}
	resp, err := makeAPIRequest("POST", "/jobs/create", bytes.NewBuffer(jsonData))
	if err != nil {
		return mcp.NewToolResultError("Failed to connect to API: " + err.Error()), nil
	}
	defer resp.Body.Close()

	var apiResp APIResponse
	if err = json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	resultData, err := json.MarshalIndent(apiResp.Data, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal result data: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(resultData)), nil
}

func updateJobTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	jobID := request.GetString("job_id", "")
	if jobID == "" {
		return mcp.NewToolResultError("job_id is required"), nil
	}

	jobData := map[string]interface{}{
		"name":          request.GetString("name", ""),
		"command":       request.GetString("command", ""),
		"cron_expr":     request.GetString("cron_expr", ""),
		"mode":          request.GetString("mode", ""),
		"max_run_count": request.GetFloat("max_run_count", 0),
		"allow_mode":    request.GetFloat("allow_mode", 0),
		"desc":          request.GetString("desc", ""),
		"url":           request.GetString("url", ""),
		"http_mode":     request.GetString("http_mode", ""),
		"proxy":         request.GetString("proxy", ""),
		"headers":       request.GetString("headers", ""),
		"data":          request.GetString("data", ""),
		"cookies":       request.GetString("cookies", ""),
		"result":        request.GetString("result", ""),
		"timeout":       request.GetFloat("timeout", 0),
		"cmd":           request.GetString("cmd", ""),
		"workdir":       request.GetString("workdir", ""),
		"env":           request.GetString("env", ""),
		"func_name":     request.GetString("func_name", ""),
		"arg":           request.GetString("arg", ""),
		"times":         request.GetFloat("times", 0),
		"interval":      request.GetFloat("interval", 0),
	}

	// 移除空值
	for k, v := range jobData {
		if v == "" || v == 0.0 {
			delete(jobData, k)
		}
	}

	// 将字符串job_id转换为uint
	jobIDUint, err := strconv.ParseUint(jobID, 10, 32)
	if err != nil {
		return mcp.NewToolResultError("Invalid job_id format: " + err.Error()), nil
	}

	// 添加id到jobData
	jobData["id"] = uint(jobIDUint)

	jsonData, _ := json.Marshal(jobData)
	resp, err := makeAPIRequest("POST", "/jobs/update", bytes.NewBuffer(jsonData))
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

	resultData, _ := json.MarshalIndent(apiResp.Data, "", "  ")
	return mcp.NewToolResultText(string(resultData)), nil
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

	data := map[string]interface{}{
		"id": uint(jobIDUint),
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal data: " + err.Error()), nil
	}

	resp, err := makeAPIRequest("POST", "/jobs/delete", bytes.NewBuffer(jsonData))
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

	resultData, _ := json.MarshalIndent(apiResp.Data, "", "  ")
	return mcp.NewToolResultText(string(resultData)), nil
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
	resp, err := makeAPIRequest("POST", "/jobs/start", bytes.NewBuffer(jsonData))
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
	if jobID == "" {
		return mcp.NewToolResultError("job_id is required"), nil
	}

	// 将字符串job_id转换为uint
	jobIDUint, err := strconv.ParseUint(jobID, 10, 32)
	if err != nil {
		return mcp.NewToolResultError("Invalid job_id format: " + err.Error()), nil
	}

	limit := int(request.GetFloat("limit", 10))

	// 使用JSON格式传递参数
	data := map[string]interface{}{
		"id":    uint(jobIDUint),
		"limit": limit,
	}

	jsonData, _ := json.Marshal(data)
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
	if err = json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return mcp.NewToolResultError("Failed to parse API response: " + err.Error()), nil
	}

	if apiResp.Code != 200 {
		return mcp.NewToolResultError("API error: " + apiResp.Msg), nil
	}

	// 确保 data 不为 nil
	if apiResp.Data == nil {
		return mcp.NewToolResultText("{}"), nil
	}

	switchData, err := json.MarshalIndent(apiResp.Data, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("Failed to marshal switch data: " + err.Error()), nil
	}

	return mcp.NewToolResultText(string(switchData)), nil
}

// makeAPIRequest 辅助函数
func makeAPIRequest(method, endpoint string, body io.Reader) (*http.Response, error) {
	url := APIBaseURL + endpoint
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	return httpClient.Do(req)
}

// 健康检查端点
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	response := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "xiaohu-jobs-mcp",
		"version":   "1.0.0",
		"endpoint":  "/mcp",
		"transport": "sse",
	}

	json.NewEncoder(w).Encode(response)
}
