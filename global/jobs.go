// global/jobs.go
package global

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"xiaohuAdmin/models/jobs"

	"github.com/robfig/cron/v3"
	"golang.org/x/net/proxy"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

var (
	Timer        *cron.Cron
	TaskList     map[uint]cron.EntryID
	TimerRunning bool // 新增状态标志
)

// Jobs 定时任务模型 - 使用models/jobs包中的Jobs类型
type Jobs = jobs.Jobs

// 初始化定时任务调度器
func InitJobs() {
	// 检查数据库连接
	if DB == nil {
		if ZapLog != nil {
			ZapLog.Error("数据库连接未初始化，无法加载任务")
		} else {
			fmt.Printf("[任务] 数据库连接未初始化，无法加载任务\n")
		}
		return
	}

	TaskList = make(map[uint]cron.EntryID)
	cronLogger := &CronLogger{}
	Timer = cron.New(cron.WithSeconds(), cron.WithLocation(time.Local), cron.WithLogger(cronLogger))

	var dbJobs []Jobs
	//查询待执行任务列表
	query := DB.
		Where("state IN (?)", []int{0, 1}).
		Find(&dbJobs)

	if query.Error != nil {
		if ZapLog != nil {
			ZapLog.Error("加载定时任务失败", LogError(query.Error))
		} else {
			fmt.Printf("[任务] 加载定时任务失败: %v\n", query.Error)
		}
		return
	}

	for _, job := range dbJobs {
		if job.State != 2 {
			err := AddJob(&job)
			if err != nil {
				if ZapLog != nil {
					ZapLog.Error("添加任务失败",
						LogField("name", job.Name),
						LogField("id", job.ID),
						LogError(err))
				} else {
					fmt.Printf("[任务] 添加任务失败 - name:%s id:%d error:%v\n", job.Name, job.ID, err)
				}
				continue
			}
		}
	}

	Timer.Start()
	TimerRunning = true // 设置初始状态

}

// 修改停止方法
func StopTimer() {
	if Timer != nil {

		ctx := Timer.Stop()

		// 等待所有任务完成并检查状态
		select {
		case <-ctx.Done():
			// 额外检查是否有残留任务
			if len(TaskList) > 0 {
				if ZapLog != nil {
					ZapLog.Warn("检测到残留任务",
						LogField("count", len(TaskList)))
				} else {
					fmt.Printf("[任务] 检测到残留任务: %d个\n", len(TaskList))
				}

				// 强制清除所有任务
				for jobId := range TaskList {
					Timer.Remove(cron.EntryID(TaskList[jobId]))
					delete(TaskList, jobId)
				}
			}
			TimerRunning = false
		case <-time.After(30 * time.Second):
			if ZapLog != nil {
				ZapLog.Error("停止任务超时，强制终止",
					LogField("pending_tasks", len(TaskList)))
			} else {
				fmt.Printf("[任务] 停止任务超时，强制终止，待处理任务: %d个\n", len(TaskList))
			}
			TimerRunning = false
		}
	}
}

func CronExprCheck(spec string) error {
	// 尝试解析 @every 格式的表达式
	_, err := cron.ParseStandard(spec)
	if err == nil {
		return nil
	}

	// 如果 @every 格式解析失败，尝试解析标准的 cron 表达式
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	_, err = parser.Parse(spec)
	if err != nil {
		return fmt.Errorf("校验失败: %s", err)
	}

	return nil
}

// 创建定时任务
func CreateJob(job *Jobs) error {
	// 验证cron表达式
	if err := CronExprCheck(job.CronExpr); err != nil {
		return fmt.Errorf("cron表达式验证失败: %v", err)
	}

	// 新增任务到数据库
	if err := DB.Create(&job).Error; err != nil {
		if ZapLog != nil {
			ZapLog.Error("新增任务失败",
				LogField("name", job.Name),
				LogField("cron", job.CronExpr),
				LogError(err))
		} else {
			fmt.Printf("[任务] 新增任务失败 - name:%s cron:%s error:%v\n", job.Name, job.CronExpr, err)
		}
		return fmt.Errorf("新增任务失败: %v", err)
	}

	// 如果任务不是停止状态就增加到调度器
	if job.State != 2 {
		if err := AddJob(job); err != nil {
			if ZapLog != nil {
				ZapLog.Error("添加任务到调度器失败",
					LogField("name", job.Name),
					LogField("id", job.ID),
					LogError(err))
			} else {
				fmt.Printf("[任务] 添加任务到调度器失败 - name:%s id:%d error:%v\n", job.Name, job.ID, err)
			}
			return err
		}
	}

	return nil
}

// 新增定时任务到调度器
func AddJob(job *Jobs) error {
	eid, err := Timer.AddJob(job.CronExpr, handle_Jobs(job))
	if err != nil {
		if ZapLog != nil {
			ZapLog.Error("添加任务失败",
				LogField("name", job.Name),
				LogField("cron", job.CronExpr),
				LogError(err))
		} else {
			fmt.Printf("[任务] 添加任务失败 - name:%s cron:%s error:%v\n", job.Name, job.CronExpr, err)
		}
		return fmt.Errorf("添加任务失败: %v", err)
	}

	AddTaskId(job.ID, eid)
	return nil
}

// 更新定时任务
func UpdateJob(job *Jobs) error {
	// 验证cron表达式
	if err := CronExprCheck(job.CronExpr); err != nil {
		return fmt.Errorf("cron表达式验证失败: %v", err)
	}

	// 更新数据库
	if err := DB.Save(&job).Error; err != nil {
		if ZapLog != nil {
			ZapLog.Error("更新任务失败",
				LogField("name", job.Name),
				LogField("id", job.ID),
				LogError(err))
		} else {
			fmt.Printf("[任务] 更新任务失败 - name:%s id:%d error:%v\n", job.Name, job.ID, err)
		}
		return fmt.Errorf("更新任务失败: %v", err)
	}

	// 如果任务状态改变，需要重新添加到调度器
	if job.State != 2 {
		// 先移除旧任务
		RemoveJob(job.ID)
		// 再添加新任务
		if err := AddJob(job); err != nil {
			return err
		}
	} else {
		// 如果任务停止，从调度器中移除
		RemoveJob(job.ID)
	}

	return nil
}

// 手动执行任务
func RunJobManually(job *Jobs) {
	executeJob(job)
}

// 执行任务
func executeJob(job *Jobs) bool {
	jobLogger := NewJobLogger(job.ID, job.Name)
	startTime := time.Now()

	log := &JobExecLog{
		Time:    startTime.Format("2006-01-02 15:04:05.000"),
		JobID:   job.ID,
		JobName: job.Name,
		Mode:    job.Mode,
	}
	var success bool
	var err error

	switch job.Mode {
	case "command":
		success, log.Command, log.ExitCode, log.Stdout, log.Stderr, err = executeCommandJobForSummary(job)
	case "http":
		success, log.Stdout, err = executeHTTPJobForSummary(job)
	case "function", "func":
		success, log.Stdout, err = executeFunctionJobForSummary(job)
	default:
		err = fmt.Errorf("不支持的任务模式: %s", job.Mode)
		success = false
	}

	endTime := time.Now()
	log.EndTime = endTime.Format("2006-01-02 15:04:05.000")
	log.Status = map[bool]string{true: "成功", false: "失败"}[success]
	log.DurationMs = endTime.Sub(startTime).Milliseconds()
	if err != nil {
		log.ErrorMsg = err.Error()
	}

	jobLogger.WriteSummaryLog(log)
	return success
}

func handle_Jobs(job *Jobs) cron.Job {
	return cron.FuncJob(func() {

		// 检查任务是否达到最大执行次数
		if job.MaxRunCount > 0 && job.RunCount >= job.MaxRunCount {

			// 停止任务
			job.State = 2
			DB.Save(job)
			RemoveJob(job.ID)
			return
		}

		// 执行任务
		executeJob(job)

		// 只有当MaxRunCount不为0时才更新任务统计信息
		if job.MaxRunCount > 0 {
			DB.Model(job).Updates(map[string]interface{}{
				"run_count": job.RunCount + 1,
			})
		}

	})
}

// executeHTTPJob 执行HTTP任务
// HTTPConfig HTTP任务配置结构
type HTTPConfig struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
	Mode    string            `json:"mode"`
	Times   int               `json:"times"`
	Proxy   string            `json:"proxy"`
	Data    string            `json:"data"`
	Cookies string            `json:"cookies"`
	Result  string            `json:"result"` // 自定义结果判断字符串
}

// parseHTTPConfig 解析HTTP任务配置
func parseHTTPConfig(command string) (*HTTPConfig, error) {
	config := &HTTPConfig{
		Headers: make(map[string]string),
		Mode:    "GET", // 默认GET
		Times:   0,     // 默认0表示不限制
	}

	lines := strings.Split(command, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 解析URL
		if strings.HasPrefix(line, "【url】") {
			config.URL = strings.TrimPrefix(line, "【url】")
			continue
		}

		// 解析请求头
		if strings.HasPrefix(line, "【headers】") {
			headersStr := strings.TrimPrefix(line, "【headers】")
			headersStr = strings.TrimSpace(headersStr)
			if headersStr != "" {
				headers := strings.Split(headersStr, "|||")
				for _, header := range headers {
					header = strings.TrimSpace(header)
					if header == "" {
						continue
					}
					parts := strings.SplitN(header, ":", 2)
					if len(parts) == 2 {
						key := strings.TrimSpace(parts[0])
						value := strings.TrimSpace(parts[1])
						config.Headers[key] = value
					}
				}
			}
			continue
		}

		// 解析请求方式
		if strings.HasPrefix(line, "【mode】") {
			mode := strings.TrimPrefix(line, "【mode】")
			mode = strings.TrimSpace(mode)
			if mode != "" {
				config.Mode = strings.ToUpper(mode)
			}
			continue
		}

		// 解析执行次数
		if strings.HasPrefix(line, "【times】") {
			timesStr := strings.TrimPrefix(line, "【times】")
			timesStr = strings.TrimSpace(timesStr)
			if timesStr != "" {
				if times, err := strconv.Atoi(timesStr); err == nil {
					config.Times = times
				}
			}
			continue
		}

		// 解析代理
		if strings.HasPrefix(line, "【proxy】") {
			proxy := strings.TrimPrefix(line, "【proxy】")
			proxy = strings.TrimSpace(proxy)
			if proxy != "" {
				config.Proxy = proxy
			}
			continue
		}

		// 解析POST数据
		if strings.HasPrefix(line, "【data】") {
			data := strings.TrimPrefix(line, "【data】")
			data = strings.TrimSpace(data)
			if data != "" {
				config.Data = data
			}
			continue
		}

		// 解析Cookie
		if strings.HasPrefix(line, "【cookies】") {
			cookies := strings.TrimPrefix(line, "【cookies】")
			cookies = strings.TrimSpace(cookies)
			if cookies != "" {
				config.Cookies = cookies
			}
			continue
		}

		// 解析自定义结果判断字符串
		if strings.HasPrefix(line, "【result】") {
			result := strings.TrimPrefix(line, "【result】")
			result = strings.TrimSpace(result)
			if result != "" {
				config.Result = result
			}
			continue
		}
	}

	if config.URL == "" {
		return nil, fmt.Errorf("URL不能为空")
	}

	return config, nil
}

// detectEncoding 检测响应编码
func detectEncoding(body []byte, contentType string) string {
	// 从Content-Type中检测编码
	if strings.Contains(strings.ToLower(contentType), "charset=") {
		parts := strings.Split(contentType, "charset=")
		if len(parts) > 1 {
			encoding := strings.TrimSpace(parts[1])
			encoding = strings.Trim(encoding, ";\"'")
			return strings.ToLower(encoding)
		}
	}

	// 从响应体中检测编码（简单检测）
	if len(body) > 3 && body[0] == 0xEF && body[1] == 0xBB && body[2] == 0xBF {
		return "utf-8"
	}

	// 默认返回utf-8
	return "utf-8"
}

// convertToUTF8 转换编码为UTF-8
func convertToUTF8(body []byte, encoding string) ([]byte, error) {
	if len(body) == 0 {
		return body, nil
	}
	if utf8.Valid(body) {
		return body, nil
	}
	// Windows下尝试GBK转UTF-8
	if runtime.GOOS == "windows" {
		reader := transform.NewReader(bytes.NewReader(body), simplifiedchinese.GBK.NewDecoder())
		utf8Bytes, err := io.ReadAll(reader)
		if err == nil && utf8.Valid(utf8Bytes) {
			return utf8Bytes, nil
		}
		// 转换失败则返回原始
		return body, err
	}
	// 其他情况直接返回
	return body, nil
}

// 通用命令执行函数，支持详细和简要返回
func executeCommandJobV2(job *Jobs, needDetail bool) (success bool, command string, exitCode int, stdout string, stderr string, err error) {
	config, err := parseCommandConfig(job.Command)
	if err != nil {
		return false, config.Command, 0, "", "", fmt.Errorf("解析命令配置失败: %v", err)
	}
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", config.Command)
	} else {
		cmd = exec.Command("bash", "-c", config.Command)
	}
	if config.WorkDir != "" {
		cmd.Dir = config.WorkDir
	}
	if len(config.Env) > 0 {
		cmd.Env = append(os.Environ(), config.Env...)
	}
	ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
	defer cancel()
	cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	startTime := time.Now()
	err = cmd.Run()
	_ = time.Since(startTime) // duration 仅用于统计，可忽略
	stdoutBytes := stdoutBuf.Bytes()
	stderrBytes := stderrBuf.Bytes()
	stdoutUTF8, _ := convertToUTF8(stdoutBytes, "")
	stderrUTF8, _ := convertToUTF8(stderrBytes, "")
	command = config.Command
	stdout = string(stdoutUTF8)
	stderr = string(stderrUTF8)
	exitCode = 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	success = err == nil && exitCode == 0
	if !needDetail {
		// 兼容原 executeCommandJob 返回 (bool, string, error)
		if err != nil && !strings.Contains(err.Error(), "timeout") {
			return success, command, exitCode, stdout, stderr, err
		}
		return success, command, exitCode, stdout, stderr, nil
	}
	return success, command, exitCode, stdout, stderr, err
}

// 替换 executeCommandJob
func executeCommandJob(job *Jobs) (bool, string, error) {
	success, _, _, stdout, _, err := executeCommandJobV2(job, false)
	return success, stdout, err
}

// 替换 executeCommandJobForSummary
func executeCommandJobForSummary(job *Jobs) (success bool, command string, exitCode int, stdout string, stderr string, err error) {
	return executeCommandJobV2(job, true)
}

// CommandConfig 命令任务配置结构
type CommandConfig struct {
	Command string        `json:"command"`  // 要执行的命令
	WorkDir string        `json:"work_dir"` // 工作目录
	Env     []string      `json:"env"`      // 环境变量
	Timeout time.Duration `json:"timeout"`  // 超时时间
}

// parseCommandConfig 解析命令任务配置
func parseCommandConfig(command string) (*CommandConfig, error) {
	config := &CommandConfig{
		Command: command,          // 默认整个command就是要执行的命令
		Timeout: 30 * time.Second, // 默认30秒超时
		Env:     make([]string, 0),
	}

	lines := strings.Split(command, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 解析命令
		if strings.HasPrefix(line, "【command】") {
			cmd := strings.TrimPrefix(line, "【command】")
			cmd = strings.TrimSpace(cmd)
			if cmd != "" {
				config.Command = cmd
			}
			continue
		}

		// 解析工作目录
		if strings.HasPrefix(line, "【workdir】") {
			workDir := strings.TrimPrefix(line, "【workdir】")
			workDir = strings.TrimSpace(workDir)
			if workDir != "" {
				config.WorkDir = workDir
			}
			continue
		}

		// 解析环境变量
		if strings.HasPrefix(line, "【env】") {
			envStr := strings.TrimPrefix(line, "【env】")
			envStr = strings.TrimSpace(envStr)
			if envStr != "" {
				envVars := strings.Split(envStr, "|||")
				for _, envVar := range envVars {
					envVar = strings.TrimSpace(envVar)
					if envVar != "" {
						config.Env = append(config.Env, envVar)
					}
				}
			}
			continue
		}

		// 解析超时时间
		if strings.HasPrefix(line, "【timeout】") {
			timeoutStr := strings.TrimPrefix(line, "【timeout】")
			timeoutStr = strings.TrimSpace(timeoutStr)
			if timeoutStr != "" {
				if timeout, err := strconv.Atoi(timeoutStr); err == nil {
					config.Timeout = time.Duration(timeout) * time.Second
				}
			}
			continue
		}
	}

	// 如果没有找到【command】标记，则整个command就是命令
	if !strings.Contains(command, "【command】") {
		config.Command = strings.TrimSpace(command)
	}

	if config.Command == "" {
		return nil, fmt.Errorf("命令不能为空")
	}

	return config, nil
}

// executeFunctionJob 执行函数任务
func executeFunctionJob(job *Jobs) (bool, string, error) {
	// 创建任务日志管理器
	jobLogger := NewJobLogger(job.ID, job.Name)

	// 解析函数配置
	config, err := parseFunctionConfig(job.Command)
	if err != nil {
		jobLogger.Error(fmt.Sprintf("解析函数配置失败: %v", err))
		return false, "", fmt.Errorf("解析函数配置失败: %v", err)
	}

	// 从统一函数管理中获取函数
	fn, exists := GetFunction(config.Name)
	if !exists {
		jobLogger.Error(fmt.Sprintf("未找到函数: %s", config.Name))
		return false, "", fmt.Errorf("未找到函数: %s", config.Name)
	}

	// 执行函数
	startTime := time.Now()
	result, err := fn(config.Args)
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// 记录函数执行输出
	jobLogger.FunctionOutput(config.Name, config.Args, result, duration)

	if err != nil {
		jobLogger.Error(fmt.Sprintf("函数执行失败: %v", err))
		return false, result, err
	}

	return true, result, nil
}

// 移除定时任务
func RemoveJob(jobId uint) error {
	ZapLog.Info("移除任务", LogField("id", jobId), LogField("job_id", TaskList[jobId]))
	if cron.EntryID(TaskList[jobId]) == 0 {
		return fmt.Errorf("任务不存在")
	}
	Timer.Remove(cron.EntryID(TaskList[jobId]))
	deleteTaskId(jobId)

	return nil
}

// 添加任务ID映射
func AddTaskId(taskId uint, entryId cron.EntryID) {
	TaskList[taskId] = entryId
}

// 删除任务ID映射
func deleteTaskId(taskId uint) {
	delete(TaskList, taskId)
}

// FunctionConfig 函数任务配置结构
type FunctionConfig struct {
	Name string   `json:"name"` // 函数名
	Args []string `json:"args"` // 函数参数
}

// parseFunctionConfig 解析函数任务配置
func parseFunctionConfig(command string) (*FunctionConfig, error) {
	config := &FunctionConfig{
		Args: make([]string, 0),
	}

	lines := strings.Split(command, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 解析函数名
		if strings.HasPrefix(line, "【name】") {
			name := strings.TrimPrefix(line, "【name】")
			name = strings.TrimSpace(name)
			if name != "" {
				config.Name = name
			}
			continue
		}

		// 解析函数参数
		if strings.HasPrefix(line, "【arg】") {
			argsStr := strings.TrimPrefix(line, "【arg】")
			argsStr = strings.TrimSpace(argsStr)

			// 如果参数为空或不存在，则使用空参数列表
			if argsStr == "" {
				config.Args = make([]string, 0)
			} else {
				// 解析参数，支持逗号分隔
				args := parseFunctionArgs(argsStr)
				config.Args = args
			}
			continue
		}
	}

	if config.Name == "" {
		return nil, fmt.Errorf("函数名不能为空")
	}

	return config, nil
}

// parseFunctionArgs 解析函数参数
func parseFunctionArgs(argsStr string) []string {
	var args []string
	var currentArg strings.Builder
	inQuotes := false
	escapeNext := false

	// 如果参数字符串为空，直接返回空列表
	if strings.TrimSpace(argsStr) == "" {
		return args
	}

	for _, char := range argsStr {
		if escapeNext {
			currentArg.WriteRune(char)
			escapeNext = false
			continue
		}

		if char == '\\' {
			escapeNext = true
			continue
		}

		if char == '"' {
			inQuotes = !inQuotes
			continue
		}

		if char == ',' && !inQuotes {
			// 参数结束
			arg := strings.TrimSpace(currentArg.String())
			// 即使是空参数也添加，保持参数位置
			args = append(args, arg)
			currentArg.Reset()
			continue
		}

		currentArg.WriteRune(char)
	}

	// 添加最后一个参数
	arg := strings.TrimSpace(currentArg.String())
	args = append(args, arg)

	return args
}

// 新增：http模式的聚合执行
func executeHTTPJobForSummary(job *Jobs) (success bool, stdout string, err error) {
	config, err := parseHTTPConfig(job.Command)
	if err != nil {
		return false, "", fmt.Errorf("解析HTTP配置失败: %v", err)
	}

	if config.URL == "" {
		return false, "", fmt.Errorf("URL不能为空")
	}

	// 创建自定义Transport
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableCompression:  false,
	}

	// 构建请求信息
	var requestInfo strings.Builder
	requestInfo.WriteString(fmt.Sprintf("请求地址: %s\n", config.URL))
	requestInfo.WriteString(fmt.Sprintf("请求方式: %s\n", config.Mode))

	// 代理信息
	if config.Proxy != "" {
		requestInfo.WriteString(fmt.Sprintf("代理状态: 使用代理 %s\n", config.Proxy))
	} else {
		requestInfo.WriteString("代理状态: 无代理\n")
	}

	// 设置代理
	if config.Proxy != "" {
		proxyURL, err := url.Parse(config.Proxy)
		if err != nil {
			errorMsg := fmt.Sprintf("代理错误: 解析代理URL失败 - %v", err)
			requestInfo.WriteString(errorMsg + "\n")
			return false, requestInfo.String(), fmt.Errorf("解析代理URL失败: %v", err)
		}

		// 根据代理类型设置不同的处理方式
		if strings.HasPrefix(config.Proxy, "socks") {
			// SOCKS代理需要特殊处理
			dialer, err := proxy.SOCKS5("tcp", proxyURL.Host, nil, proxy.Direct)
			if err != nil {
				errorMsg := fmt.Sprintf("代理错误: 创建SOCKS代理拨号器失败 - %v", err)
				requestInfo.WriteString(errorMsg + "\n")
				return false, requestInfo.String(), fmt.Errorf("创建SOCKS代理拨号器失败: %v", err)
			}
			transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialer.Dial(network, addr)
			}
		} else {
			// HTTP/HTTPS代理
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	// 创建HTTP客户端，增加超时时间
	client := &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second, // 增加到60秒超时
	}

	// 创建请求
	var req *http.Request
	var err2 error

	if config.Mode == "POST" {
		// POST请求
		var body io.Reader
		if config.Data != "" {
			body = strings.NewReader(config.Data)
			requestInfo.WriteString(fmt.Sprintf("POST数据: %s\n", config.Data))
		}
		req, err2 = http.NewRequest("POST", config.URL, body)
	} else {
		// GET请求（默认）
		req, err2 = http.NewRequest("GET", config.URL, nil)
	}

	if err2 != nil {
		errorMsg := fmt.Sprintf("请求错误: 创建HTTP请求失败 - %v", err2)
		requestInfo.WriteString(errorMsg + "\n")
		return false, requestInfo.String(), fmt.Errorf("创建HTTP请求失败: %v", err2)
	}

	// 设置请求头
	if len(config.Headers) > 0 {
		requestInfo.WriteString("请求头:\n")
		for key, value := range config.Headers {
			req.Header.Set(key, value)
			requestInfo.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
		}
	}

	// 设置Cookie
	if config.Cookies != "" {
		req.Header.Set("Cookie", config.Cookies)
		requestInfo.WriteString(fmt.Sprintf("Cookie: %s\n", config.Cookies))
	}

	// 执行请求
	resp, err := client.Do(req)
	if err != nil {
		errorMsg := fmt.Sprintf("请求错误: HTTP请求失败 - %v", err)
		requestInfo.WriteString(errorMsg + "\n")
		return false, requestInfo.String(), fmt.Errorf("HTTP请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 添加响应状态信息
	requestInfo.WriteString(fmt.Sprintf("响应状态: %s (%d)\n", resp.Status, resp.StatusCode))

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		errorMsg := fmt.Sprintf("响应错误: 读取响应体失败 - %v", err)
		requestInfo.WriteString(errorMsg + "\n")
		return false, requestInfo.String(), fmt.Errorf("读取响应体失败: %v", err)
	}

	// 检测并转换编码
	encoding := detectEncoding(body, resp.Header.Get("Content-Type"))
	utf8Body, err := convertToUTF8(body, encoding)
	if err != nil {
		errorMsg := fmt.Sprintf("编码错误: 编码转换失败 - %v", err)
		requestInfo.WriteString(errorMsg + "\n")
		return false, requestInfo.String(), fmt.Errorf("编码转换失败: %v", err)
	}

	// 添加响应头信息
	if len(resp.Header) > 0 {
		requestInfo.WriteString("响应头:\n")
		for key, values := range resp.Header {
			for _, value := range values {
				requestInfo.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
			}
		}
	}

	// 添加响应内容
	responseContent := string(utf8Body)
	if len(responseContent) > 1000 {
		responseContent = responseContent[:1000] + "\n... (响应内容已截断)"
	}
	requestInfo.WriteString(fmt.Sprintf("响应内容:\n%s", responseContent))

	// 判断是否成功
	success = resp.StatusCode >= 200 && resp.StatusCode < 300

	// 如果有自定义结果判断
	if config.Result != "" {
		success = strings.Contains(responseContent, config.Result)
		requestInfo.WriteString(fmt.Sprintf("\n自定义结果判断: 查找 '%s' - %s", config.Result, map[bool]string{true: "找到", false: "未找到"}[success]))
	}

	return success, requestInfo.String(), nil
}

// 新增：function模式的聚合执行
func executeFunctionJobForSummary(job *Jobs) (success bool, stdout string, err error) {
	config, err := parseFunctionConfig(job.Command)
	if err != nil {
		return false, "", fmt.Errorf("解析函数配置失败: %v", err)
	}

	// 从统一函数管理中获取函数
	fn, exists := GetFunction(config.Name)
	if !exists {
		return false, "", fmt.Errorf("未找到函数: %s", config.Name)
	}

	// 执行函数
	result, err := fn(config.Args)

	if err != nil {
		return false, result, err
	}

	return true, result, nil
}
