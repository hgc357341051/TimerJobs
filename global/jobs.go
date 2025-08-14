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
	"sync"
	"time"
	"unicode/utf8"

	"xiaohuAdmin/models/jobs"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"golang.org/x/net/proxy"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"gorm.io/gorm"
)

var (
	Timer        *cron.Cron
	TaskList     map[uint]cron.EntryID
	TimerRunning bool // 新增状态标志

	taskMu    sync.RWMutex
	runningMu sync.RWMutex

	// 手动执行并发控制：每个任务一个容量为1的信号量，用于串行/跳过/排队
	jobSemaphores sync.Map // map[uint]chan struct{}
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

	taskMu.Lock()
	TaskList = make(map[uint]cron.EntryID)
	taskMu.Unlock()
	cronLogger := &CronLogger{}
	Timer = cron.New(
		cron.WithSeconds(),
		cron.WithLocation(time.Local),
		cron.WithLogger(cronLogger),
		cron.WithChain(cron.Recover(cronLogger)),
	)

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
	runningMu.Lock()
	TimerRunning = true // 设置初始状态
	runningMu.Unlock()

}

// 修改停止方法
func StopTimer() {
	if Timer != nil {

		ctx := Timer.Stop()

		// 等待所有任务完成并检查状态
		select {
		case <-ctx.Done():
			// 额外检查是否有残留任务
			remaining := GetTaskListSnapshot()
			if len(remaining) > 0 {
				if ZapLog != nil {
					ZapLog.Warn("检测到残留任务",
						LogField("count", len(remaining)))
				} else {
					fmt.Printf("[任务] 检测到残留任务: %d个\n", len(remaining))
				}

				// 强制清除所有任务
				for jobId, entryId := range remaining {
					Timer.Remove(cron.EntryID(entryId))
					deleteTaskId(jobId)
				}
			}
			runningMu.Lock()
			TimerRunning = false
			runningMu.Unlock()
		case <-time.After(30 * time.Second):
			pending := GetTaskCount()
			if ZapLog != nil {
				ZapLog.Error("停止任务超时，强制终止",
					LogField("pending_tasks", pending))
			} else {
				fmt.Printf("[任务] 停止任务超时，强制终止，待处理任务: %d个\n", pending)
			}
			runningMu.Lock()
			TimerRunning = false
			runningMu.Unlock()
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
	// 根据 AllowMode 设置并发策略（支持全局默认）
	allow := job.AllowMode
	if allow == 0 { // 0 表示并行；若全局配置指定了1或2，则作为默认
		cfgDefault := GetJobsConfigInt("jobs.default_allow_mode", 0)
		if cfgDefault == 1 || cfgDefault == 2 {
			allow = cfgDefault
		}
	}
	var j cron.Job = handle_Jobs(job)
	switch allow {
	case 1: // 串行，仍在执行时跳过
		j = cron.NewChain(cron.SkipIfStillRunning(&CronLogger{})).Then(j)
	case 2: // 串行，仍在执行时排队
		j = cron.NewChain(cron.DelayIfStillRunning(&CronLogger{})).Then(j)
	}
	eid, err := Timer.AddJob(job.CronExpr, j)
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
		if err := RemoveJob(job.ID); err != nil {
			if ZapLog != nil {
				ZapLog.Error("从调度器移除旧任务失败", LogError(err))
			}
		}
		// 再添加新任务
		if err := AddJob(job); err != nil {
			return err
		}
	} else {
		// 如果任务停止，从调度器中移除
		if err := RemoveJob(job.ID); err != nil {
			if ZapLog != nil {
				ZapLog.Error("从调度器移除停止的任务失败", LogError(err))
			}
		}
	}

	return nil
}

// 手动执行任务
func RunJobManually(job *Jobs) string {
	execID := uuid.NewString()
	go executeJobWithExecID(job, execID)
	return execID
}

// 手动执行任务（带并发策略），返回 execID/是否跳过/原因
func RunJobManuallyWithPolicy(job *Jobs) (execID string, skipped bool, reason string) {
	// 是否允许手动并发
	allowConc := GetJobsConfigBool("jobs.manual_allow_concurrent", true)
	if allowConc {
		return RunJobManually(job), false, ""
	}

	// 不允许手动并发时，按 AllowMode 决定策略
	switch job.AllowMode {
	case 1: // Skip: 仍在执行时跳过
		ch := getJobSemaphore(job.ID)
		select {
		case ch <- struct{}{}:
			// 获得执行权
			execID = uuid.NewString()
			go func() {
				defer func() { <-ch }()
				executeJobWithExecID(job, execID)
			}()
			return execID, false, ""
		default:
			// 正在执行，跳过
			return "", true, "任务仍在执行，已按策略跳过"
		}
	case 2: // Delay: 排队直到可执行
		ch := getJobSemaphore(job.ID)
		execID = uuid.NewString()
		go func() {
			ch <- struct{}{}
			defer func() { <-ch }()
			executeJobWithExecID(job, execID)
		}()
		return execID, false, ""
	default: // 并行
		return RunJobManually(job), false, ""
	}
}

// 执行任务
func executeJob(job *Jobs) bool {
	jobLogger := NewJobLogger(job.ID, job.Name)
	startTime := time.Now()
	// running++
	MetricsSetRunning(1)

	log := &JobExecLog{
		Time:    startTime.Format("2006-01-02 15:04:05.000"),
		JobID:   job.ID,
		JobName: job.Name,
		Mode:    job.Mode,
		ExecID:  uuid.NewString(),
		Source:  "cron",
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
	// 指标
	MetricsIncExec(strconv.Itoa(int(job.ID)), job.Name, job.Mode)
	if !success {
		MetricsIncFail(strconv.Itoa(int(job.ID)), job.Name, job.Mode)
	}
	MetricsObserveDuration(strconv.Itoa(int(job.ID)), job.Name, job.Mode, float64(log.DurationMs)/1000.0)
	// running--
	MetricsSetRunning(-1)
	return success
}

// 带外部执行ID的执行函数（用于手动执行返回可跟踪ID）
func executeJobWithExecID(job *Jobs, execID string) bool {
	jobLogger := NewJobLogger(job.ID, job.Name)
	startTime := time.Now()
	MetricsSetRunning(1)

	log := &JobExecLog{
		Time:    startTime.Format("2006-01-02 15:04:05.000"),
		JobID:   job.ID,
		JobName: job.Name,
		Mode:    job.Mode,
		ExecID:  execID,
		Source:  "manual",
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
	MetricsIncExec(strconv.Itoa(int(job.ID)), job.Name, job.Mode)
	if !success {
		MetricsIncFail(strconv.Itoa(int(job.ID)), job.Name, job.Mode)
	}
	MetricsObserveDuration(strconv.Itoa(int(job.ID)), job.Name, job.Mode, float64(log.DurationMs)/1000.0)
	MetricsSetRunning(-1)
	return success
}

func handle_Jobs(job *Jobs) cron.Job {
	return cron.FuncJob(func() {

		// 读取数据库中的最新计数与上限
		var current Jobs
		if err := DB.Select("id,max_run_count,run_count,state").First(&current, job.ID).Error; err == nil {
			if current.MaxRunCount > 0 && current.RunCount >= current.MaxRunCount {
				// 达到上限：置停止并移除
				DB.Model(&jobs.Jobs{}).Where("id=?", job.ID).Update("state", 2)
				if err := RemoveJob(job.ID); err != nil {
					if ZapLog != nil {
						ZapLog.Error("从调度器移除任务失败", LogError(err))
					}
				}
				return
			}
		}

		// 执行前置状态：执行中
		if err := DB.Model(&jobs.Jobs{}).Where("id=?", job.ID).Update("state", 1).Error; err != nil {
			if ZapLog != nil {
				ZapLog.Warn("更新任务为执行中失败", LogError(err), LogField("job_id", job.ID))
			}
		}

		// 执行任务
		success := executeJob(job)

		// 统计：原子自增
		if err := DB.Model(&jobs.Jobs{}).Where("id=?", job.ID).UpdateColumn("run_count", gorm.Expr("run_count + ?", 1)).Error; err != nil {
			if ZapLog != nil {
				ZapLog.Warn("更新任务运行次数失败", LogError(err), LogField("job_id", job.ID))
			}
		}

		// 读取最新计数判断是否达到上限
		if err := DB.Select("id,max_run_count,run_count").First(&current, job.ID).Error; err == nil {
			if current.MaxRunCount > 0 && current.RunCount >= current.MaxRunCount {
				DB.Model(&jobs.Jobs{}).Where("id=?", job.ID).Update("state", 2)
				if err := RemoveJob(job.ID); err != nil {
					if ZapLog != nil {
						ZapLog.Error("从调度器移除任务失败", LogError(err))
					}
				}
				return
			}
		}

		// 执行结束：若仍启用则置为等待
		if success {
			DB.Model(&jobs.Jobs{}).Where("id=? AND state<>?", job.ID, 2).Update("state", 0)
		} else {
			// 失败也置回等待（可根据需要扩展失败状态）
			DB.Model(&jobs.Jobs{}).Where("id=? AND state<>?", job.ID, 2).Update("state", 0)
		}

	})
}

// executeHTTPJob 执行HTTP任务
// HTTPConfig HTTP任务配置结构
type HTTPConfig struct {
	URL      string            `json:"url"`
	Headers  map[string]string `json:"headers"`
	Mode     string            `json:"mode"`
	Times    int               `json:"times"`
	Interval int               `json:"interval"` // 次数间隔秒
	Proxy    string            `json:"proxy"`
	Data     string            `json:"data"`
	Cookies  string            `json:"cookies"`
	Result   string            `json:"result"`  // 自定义结果判断字符串
	Timeout  int               `json:"timeout"` // 超时时间（秒）
}

// parseHTTPConfig 解析HTTP任务配置
func parseHTTPConfig(command string) (*HTTPConfig, error) {
	config := &HTTPConfig{
		Headers: make(map[string]string),
		Mode:    "GET",                                                // 默认GET
		Times:   0,                                                    // 默认0表示不限制
		Timeout: GetJobsConfigInt("jobs.default_timeout_seconds", 60), // 默认超时
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

		// 解析间隔秒数
		if strings.HasPrefix(line, "【interval】") {
			iv := strings.TrimPrefix(line, "【interval】")
			iv = strings.TrimSpace(iv)
			if iv != "" {
				if secs, err := strconv.Atoi(iv); err == nil {
					config.Interval = secs
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

		// 解析超时时间
		if strings.HasPrefix(line, "【timeout】") {
			timeoutStr := strings.TrimPrefix(line, "【timeout】")
			timeoutStr = strings.TrimSpace(timeoutStr)
			if timeoutStr != "" {
				if timeout, err := strconv.Atoi(timeoutStr); err == nil {
					config.Timeout = timeout
				}
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

// 替换 executeCommandJobForSummary
func executeCommandJobForSummary(job *Jobs) (success bool, command string, exitCode int, stdout string, stderr string, err error) {
	cfg, perr := parseCommandConfig(job.Command)
	if perr != nil {
		return false, cfg.Command, 0, "", "", fmt.Errorf("解析命令配置失败: %v", perr)
	}
	// 次数与间隔
	attempts := cfg.Times
	if attempts <= 0 {
		attempts = 1
	}
	var outB strings.Builder
	var errB strings.Builder
	var lastExit int
	var lastErr error
	anySuccess := false
	for i := 1; i <= attempts; i++ {
		s, cmdStr, code, out, er, e := executeCommandJobV2(job, true)
		if i == 1 {
			command = cmdStr
		}
		lastExit = code
		lastErr = e
		outB.WriteString(fmt.Sprintf("\n=== 第 %d/%d 次执行 ===\n", i, attempts))
		if out != "" {
			outB.WriteString(out)
		}
		if er != "" {
			errB.WriteString(fmt.Sprintf("\n[attempt %d] %s\n", i, er))
		}
		if s {
			anySuccess = true
		}
		if i < attempts && cfg.Interval > 0 {
			time.Sleep(time.Duration(cfg.Interval) * time.Second)
		}
	}
	stdout = outB.String()
	stderr = errB.String()
	exitCode = lastExit
	if anySuccess {
		return true, command, exitCode, stdout, stderr, nil
	}
	return false, command, exitCode, stdout, stderr, lastErr
}

// CommandConfig 命令任务配置结构
type CommandConfig struct {
	Command  string        `json:"command"`  // 要执行的命令
	WorkDir  string        `json:"work_dir"` // 工作目录
	Env      []string      `json:"env"`      // 环境变量
	Timeout  time.Duration `json:"timeout"`  // 超时时间
	Times    int           `json:"times"`
	Interval int           `json:"interval"`
}

// parseCommandConfig 解析命令任务配置
func parseCommandConfig(command string) (*CommandConfig, error) {
	config := &CommandConfig{
		Command: command,                                                                           // 默认整个command就是要执行的命令
		Timeout: time.Duration(GetJobsConfigInt("jobs.default_timeout_seconds", 30)) * time.Second, // 默认超时
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

		// 解析执行次数
		if strings.HasPrefix(line, "【times】") {
			t := strings.TrimPrefix(line, "【times】")
			t = strings.TrimSpace(t)
			if t != "" {
				if n, err := strconv.Atoi(t); err == nil {
					config.Times = n
				}
			}
			continue
		}

		// 解析间隔
		if strings.HasPrefix(line, "【interval】") {
			iv := strings.TrimPrefix(line, "【interval】")
			iv = strings.TrimSpace(iv)
			if iv != "" {
				if n, err := strconv.Atoi(iv); err == nil {
					config.Interval = n
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

// 移除定时任务
func RemoveJob(jobId uint) error {
	ZapLog.Info("移除任务", LogField("id", jobId), LogField("job_id", func() cron.EntryID { taskMu.RLock(); defer taskMu.RUnlock(); return TaskList[jobId] }()))
	taskMu.RLock()
	entryID := TaskList[jobId]
	taskMu.RUnlock()
	if cron.EntryID(entryID) == 0 {
		return fmt.Errorf("任务不存在")
	}
	Timer.Remove(cron.EntryID(entryID))
	deleteTaskId(jobId)

	return nil
}

// 添加任务ID映射
func AddTaskId(taskId uint, entryId cron.EntryID) {
	taskMu.Lock()
	TaskList[taskId] = entryId
	taskMu.Unlock()
}

// 删除任务ID映射
func deleteTaskId(taskId uint) {
	taskMu.Lock()
	delete(TaskList, taskId)
	taskMu.Unlock()
}

// 并发安全：获取任务数量
func GetTaskCount() int {
	taskMu.RLock()
	defer taskMu.RUnlock()
	return len(TaskList)
}

// 并发安全：获取任务快照（用于只读遍历）
func GetTaskListSnapshot() map[uint]cron.EntryID {
	result := make(map[uint]cron.EntryID)
	taskMu.RLock()
	for k, v := range TaskList {
		result[k] = v
	}
	taskMu.RUnlock()
	return result
}

// 并发安全：获取调度器运行状态
func IsTimerRunning() bool {
	runningMu.RLock()
	defer runningMu.RUnlock()
	return TimerRunning
}

// FunctionConfig 函数任务配置结构
type FunctionConfig struct {
	Name     string   `json:"name"` // 函数名
	Args     []string `json:"args"` // 函数参数
	Times    int      `json:"times"`
	Interval int      `json:"interval"`
	Timeout  int      `json:"timeout"` // 超时时间（秒）
}

// parseFunctionConfig 解析函数任务配置
func parseFunctionConfig(command string) (*FunctionConfig, error) {
	config := &FunctionConfig{
		Args:    make([]string, 0),
		Timeout: GetJobsConfigInt("jobs.default_timeout_seconds", 30), // 默认超时
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

		// 解析执行次数
		if strings.HasPrefix(line, "【times】") {
			t := strings.TrimPrefix(line, "【times】")
			t = strings.TrimSpace(t)
			if t != "" {
				if n, err := strconv.Atoi(t); err == nil {
					config.Times = n
				}
			}
			continue
		}

		// 解析间隔
		if strings.HasPrefix(line, "【interval】") {
			iv := strings.TrimPrefix(line, "【interval】")
			iv = strings.TrimSpace(iv)
			if iv != "" {
				if n, err := strconv.Atoi(iv); err == nil {
					config.Interval = n
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
					config.Timeout = timeout
				}
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
		proxyURL, perr := url.Parse(config.Proxy)
		if perr != nil {
			errorMsg := fmt.Sprintf("代理错误: 解析代理URL失败 - %v", perr)
			requestInfo.WriteString(errorMsg + "\n")
			return false, requestInfo.String(), fmt.Errorf("解析代理URL失败: %v", perr)
		}

		// 根据代理类型设置不同的处理方式
		if strings.HasPrefix(config.Proxy, "socks") {
			// SOCKS代理
			dialer, derr := proxy.SOCKS5("tcp", proxyURL.Host, nil, proxy.Direct)
			if derr != nil {
				errorMsg := fmt.Sprintf("代理错误: 创建SOCKS代理拨号器失败 - %v", derr)
				requestInfo.WriteString(errorMsg + "\n")
				return false, requestInfo.String(), fmt.Errorf("创建SOCKS代理拨号器失败: %v", derr)
			}
			transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				// 该接口不支持ctx取消，只能尽量复用传入的上下文
				return dialer.Dial(network, addr)
			}
		} else {
			// HTTP/HTTPS代理
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	// 创建HTTP客户端，使用配置的超时时间
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(config.Timeout) * time.Second,
	}

	// 设置请求头/通用信息
	if len(config.Headers) > 0 {
		requestInfo.WriteString("请求头:\n")
		for key, value := range config.Headers {
			requestInfo.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
		}
	}
	if config.Cookies != "" {
		requestInfo.WriteString(fmt.Sprintf("Cookie: %s\n", config.Cookies))
	}
	if config.Data != "" && strings.ToUpper(config.Mode) == "POST" {
		requestInfo.WriteString(fmt.Sprintf("POST数据: %s\n", config.Data))
	}

	// Times 支持：<=0 视为 1 次
	attempts := config.Times
	if attempts <= 0 {
		attempts = 1
	}

	// 响应截断长度从配置获取
	maxBytes := GetJobsConfigInt("jobs.http_response_max_bytes", 1000)
	if maxBytes <= 0 {
		maxBytes = 1000
	}

	// 执行循环
	anySuccess := false
	for i := 1; i <= attempts; i++ {
		requestInfo.WriteString(fmt.Sprintf("\n=== 第 %d/%d 次请求 ===\n", i, attempts))

		// 创建请求
		var req *http.Request
		var reqErr error
		method := strings.ToUpper(config.Mode)
		if method == "POST" {
			var body io.Reader
			if config.Data != "" {
				body = strings.NewReader(config.Data)
			}
			req, reqErr = http.NewRequest("POST", config.URL, body)
		} else {
			req, reqErr = http.NewRequest("GET", config.URL, nil)
		}
		if reqErr != nil {
			errorMsg := fmt.Sprintf("请求错误: 创建HTTP请求失败 - %v", reqErr)
			requestInfo.WriteString(errorMsg + "\n")
			// 本次失败，继续下一次
			continue
		}

		// 设置头/Cookie
		for key, value := range config.Headers {
			req.Header.Set(key, value)
		}
		if config.Cookies != "" {
			req.Header.Set("Cookie", config.Cookies)
		}

		// 执行请求
		resp, doErr := client.Do(req)
		if doErr != nil {
			errorMsg := fmt.Sprintf("请求错误: HTTP请求失败 - %v", doErr)
			requestInfo.WriteString(errorMsg + "\n")
			continue
		}
		func() {
			defer resp.Body.Close()

			// 状态
			requestInfo.WriteString(fmt.Sprintf("响应状态: %s (%d)\n", resp.Status, resp.StatusCode))

			// 读取响应
			body, rerr := io.ReadAll(resp.Body)
			if rerr != nil {
				requestInfo.WriteString(fmt.Sprintf("响应错误: 读取响应体失败 - %v\n", rerr))
				return
			}
			encoding := detectEncoding(body, resp.Header.Get("Content-Type"))
			utf8Body, cerr := convertToUTF8(body, encoding)
			if cerr != nil {
				requestInfo.WriteString(fmt.Sprintf("编码错误: 编码转换失败 - %v\n", cerr))
				return
			}

			// 响应头（仅第一次打印以控制体积）
			if i == 1 && len(resp.Header) > 0 {
				requestInfo.WriteString("响应头:\n")
				for key, values := range resp.Header {
					for _, value := range values {
						requestInfo.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
					}
				}
			}

			// 响应内容（截断）
			responseContent := string(utf8Body)
			if maxBytes > 0 && len(responseContent) > maxBytes {
				responseContent = responseContent[:maxBytes] + "\n... (响应内容已截断)"
			}
			requestInfo.WriteString("响应内容:\n")
			requestInfo.WriteString(responseContent)

			// 判断是否成功
			s := resp.StatusCode >= 200 && resp.StatusCode < 300
			if config.Result != "" {
				s = strings.Contains(responseContent, config.Result)
				requestInfo.WriteString(fmt.Sprintf("\n自定义结果判断: 查找 '%s' - %s", config.Result, map[bool]string{true: "找到", false: "未找到"}[s]))
			}
			if s {
				anySuccess = true
			}
		}()
		// 间隔控制（最后一次不等待）
		if i < attempts && config.Interval > 0 {
			time.Sleep(time.Duration(config.Interval) * time.Second)
		}
	}

	return anySuccess, requestInfo.String(), nil
}

// 新增：function模式的聚合执行
func executeFunctionJobForSummary(job *Jobs) (success bool, stdout string, err error) {
	config, e := parseFunctionConfig(job.Command)
	if e != nil {
		return false, "", fmt.Errorf("解析函数配置失败: %v", e)
	}

	// 从统一函数管理中获取函数
	fn, exists := GetFunction(config.Name)
	if !exists {
		return false, "", fmt.Errorf("未找到函数: %s", config.Name)
	}

	attempts := config.Times
	if attempts <= 0 {
		attempts = 1
	}

	var b strings.Builder
	anySuccess := false
	var lastErr error

	for i := 1; i <= attempts; i++ {
		b.WriteString(fmt.Sprintf("\n=== 第 %d/%d 次执行 ===\n", i, attempts))

		// 创建带超时的上下文
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(config.Timeout)*time.Second)

		// 使用通道实现超时控制
		resultChan := make(chan struct {
			result string
			err    error
		}, 1)

		go func() {
			res, ferr := fn(config.Args)
			resultChan <- struct {
				result string
				err    error
			}{res, ferr}
		}()

		select {
		case result := <-resultChan:
			if result.result != "" {
				b.WriteString(result.result)
			}
			if result.err == nil {
				anySuccess = true
			} else {
				lastErr = result.err
				b.WriteString(fmt.Sprintf("\n[attempt %d] error: %v\n", i, result.err))
			}
		case <-ctx.Done():
			lastErr = fmt.Errorf("函数执行超时（%d秒）", config.Timeout)
			b.WriteString(fmt.Sprintf("\n[attempt %d] timeout: %v\n", i, lastErr))
		}

		cancel()

		// 间隔控制（最后一次不等待）
		if i < attempts && config.Interval > 0 {
			time.Sleep(time.Duration(config.Interval) * time.Second)
		}
	}

	if anySuccess {
		return true, b.String(), nil
	}
	return false, b.String(), lastErr
}

// 获取任务信号量（容量为1），用于手动执行的并发控制
func getJobSemaphore(jobID uint) chan struct{} {
	if ch, ok := jobSemaphores.Load(jobID); ok {
		return ch.(chan struct{})
	}
	ch := make(chan struct{}, 1)
	actual, _ := jobSemaphores.LoadOrStore(jobID, ch)
	return actual.(chan struct{})
}
