package global

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// JobLogger 任务日志管理器
type JobLogger struct {
	jobID   uint
	jobName string
}

// 全局文件句柄缓存和互斥锁
var (
	fileHandles = make(map[string]*os.File)
	fileMutex   sync.RWMutex
	writeMutex  sync.Mutex
)

// NewJobLogger 创建任务日志管理器
func NewJobLogger(jobID uint, jobName string) *JobLogger {
	return &JobLogger{
		jobID:   jobID,
		jobName: jobName,
	}
}

// getLogPath 获取日志文件路径
func (jl *JobLogger) getLogPath() string {
	now := time.Now()
	year := fmt.Sprintf("%d", now.Year())
	month := fmt.Sprintf("%02d", now.Month())
	day := fmt.Sprintf("%02d", now.Day())

	// 创建目录结构: runtime/jobs/任务ID/年/月/日.log
	logDir := filepath.Join("runtime", "jobs", fmt.Sprintf("%d", jl.jobID), year, month)

	// 确保目录存在
	if err := os.MkdirAll(logDir, 0755); err != nil {
		if ZapLog != nil {
			ZapLog.Error("创建日志目录失败", LogError(err))
		}
		return ""
	}

	return filepath.Join(logDir, fmt.Sprintf("%s.log", day))
}

// getFileHandle 获取文件句柄（带缓存）
func (jl *JobLogger) getFileHandle(logPath string) (*os.File, error) {
	// 先尝试从缓存获取
	fileMutex.RLock()
	if file, exists := fileHandles[logPath]; exists {
		fileMutex.RUnlock()
		return file, nil
	}
	fileMutex.RUnlock()

	// 缓存中没有，创建新文件句柄
	fileMutex.Lock()
	defer fileMutex.Unlock()

	// 双重检查，防止并发创建
	if file, exists := fileHandles[logPath]; exists {
		return file, nil
	}

	// 打开文件（追加模式）
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	// 缓存文件句柄
	fileHandles[logPath] = file
	return file, nil
}

// writeLog 写入日志（单行结构化JSON格式）
func (jl *JobLogger) writeLog(level, message string) {
	logPath := jl.getLogPath()
	if logPath == "" {
		return
	}

	now := time.Now()
	timestamp := now.Format("2006-01-02 15:04:05.000")

	// 日志级别中文映射
	var levelCN string
	switch level {
	case "START":
		levelCN = "开始"
	case "END":
		levelCN = "结束"
	case "COMMAND":
		levelCN = "命令"
	case "HTTP":
		levelCN = "HTTP"
	case "FUNCTION":
		levelCN = "函数"
	case "ERROR":
		levelCN = "错误"
	case "SUCCESS":
		levelCN = "成功"
	case "WARNING":
		levelCN = "警告"
	default:
		levelCN = level
	}

	logEntry := map[string]interface{}{
		"time":     timestamp,
		"level":    levelCN,
		"job_id":   jl.jobID,
		"job_name": jl.jobName,
		"message":  message,
	}
	line, err := json.Marshal(logEntry)
	if err != nil {
		return
	}
	logLine := string(line) + "\n"

	// 获取文件句柄
	file, err := jl.getFileHandle(logPath)
	if err != nil {
		if ZapLog != nil {
			ZapLog.Error("获取日志文件句柄失败", LogError(err))
		}
		return
	}

	// 使用互斥锁确保写入原子性
	writeMutex.Lock()
	defer writeMutex.Unlock()

	if _, err := file.WriteString(logLine); err != nil {
		if ZapLog != nil {
			ZapLog.Error("写入日志文件失败", LogError(err))
		}
		file.Sync()
	}
}

// indentForDetail 判断是否需要缩进详细内容
func indentForDetail(line string) string {
	// 直接返回缩进
	return "    "
}

// Error 记录错误日志
func (jl *JobLogger) Error(message string) {
	jl.writeLog("ERROR", message)
}

// Success 记录成功日志
func (jl *JobLogger) Success(message string) {
	jl.writeLog("SUCCESS", message)
}

// Warning 记录警告日志
func (jl *JobLogger) Warning(message string) {
	jl.writeLog("WARNING", message)
}

// Start 记录任务开始日志（参数化模式）
func (jl *JobLogger) Start(mode string) {
	message := fmt.Sprintf("任务开始执行 - 模式: %s", mode)
	jl.writeLog("START", message)
}

// End 记录任务结束日志
func (jl *JobLogger) End(success bool, duration time.Duration, output string) {
	status := "成功"
	if !success {
		status = "失败"
	}
	// 只记录状态和时长
	message := fmt.Sprintf("执行状态: %s | 执行时长: %v", status, duration)
	jl.writeLog("END", message)
}

// CommandOutput 记录命令执行输出
func (jl *JobLogger) CommandOutput(command string, exitCode int, stdout, stderr string, duration time.Duration) {
	var details []string
	details = append(details, fmt.Sprintf("命令: %s", command))
	details = append(details, fmt.Sprintf("退出码: %d", exitCode))
	details = append(details, fmt.Sprintf("执行时长: %v", duration))

	if stdout != "" {
		details = append(details, fmt.Sprintf("标准输出:\n%s", stdout))
	}

	if stderr != "" {
		details = append(details, fmt.Sprintf("错误输出:\n%s", stderr))
	}

	message := strings.Join(details, " | ")
	jl.writeLog("COMMAND", message)
}

// HTTPOutput 记录HTTP请求输出
func (jl *JobLogger) HTTPOutput(url, method string, statusCode int, responseBody string, duration time.Duration) {
	var details []string
	details = append(details, fmt.Sprintf("URL: %s", url))
	details = append(details, fmt.Sprintf("方法: %s", method))
	details = append(details, fmt.Sprintf("状态码: %d", statusCode))
	details = append(details, fmt.Sprintf("请求时长: %v", duration))

	if responseBody != "" {
		// 限制响应体长度
		if len(responseBody) > 1000 {
			responseBody = responseBody[:1000] + "\n... (响应体已截断)"
		}
		details = append(details, fmt.Sprintf("响应内容:\n%s", responseBody))
	}

	message := strings.Join(details, " | ")
	jl.writeLog("HTTP", message)
}

// FunctionOutput 记录函数执行输出
func (jl *JobLogger) FunctionOutput(functionName string, args []string, result string, duration time.Duration) {
	var details []string
	details = append(details, fmt.Sprintf("函数名: %s", functionName))
	if len(args) > 0 {
		details = append(details, fmt.Sprintf("参数: %v", args))
	}
	details = append(details, fmt.Sprintf("执行时长: %v", duration))

	if result != "" {
		// 限制结果长度
		if len(result) > 1000 {
			result = result[:1000] + "\n... (结果已截断)"
		}
		details = append(details, fmt.Sprintf("执行结果:\n%s", result))
	}

	message := strings.Join(details, " | ")
	jl.writeLog("FUNCTION", message)
}

// CloseAllFileHandles 关闭所有文件句柄（程序退出时调用）
func CloseAllFileHandles() {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	for path, file := range fileHandles {
		file.Close()
		delete(fileHandles, path)
	}
}

// 聚合任务执行日志结构体
// 每次任务执行完毕后只写一条
type JobExecLog struct {
	Time       string   `json:"time"`     // 任务开始时间
	EndTime    string   `json:"end_time"` // 任务结束时间
	JobID      uint     `json:"job_id"`
	JobName    string   `json:"job_name"`
	Status     string   `json:"status"` // 成功/失败
	DurationMs int64    `json:"duration_ms"`
	Mode       string   `json:"mode"`
	Command    string   `json:"command,omitempty"`
	ExitCode   int      `json:"exit_code,omitempty"`
	Stdout     string   `json:"stdout,omitempty"`
	Stderr     string   `json:"stderr,omitempty"`
	HttpUrl    string   `json:"http_url,omitempty"`
	HttpMethod string   `json:"http_method,omitempty"`
	HttpStatus int      `json:"http_status,omitempty"`
	HttpResp   string   `json:"http_resp,omitempty"`
	FuncName   string   `json:"func_name,omitempty"`
	FuncArgs   []string `json:"func_args,omitempty"`
	FuncResult string   `json:"func_result,omitempty"`
	ErrorMsg   string   `json:"error_msg,omitempty"`
}

// 写入聚合日志
func (jl *JobLogger) WriteSummaryLog(log *JobExecLog) {
	logPath := jl.getLogPath()
	line, _ := json.Marshal(log)
	file, _ := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer file.Close()
	file.WriteString(string(line) + "\n")
}
