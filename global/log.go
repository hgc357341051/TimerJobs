package global

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	ZapLog        *zap.Logger     // 全局日志器
	ZapLogLevels  map[string]bool // 存储启用的日志级别
	GinLogMethods map[string]bool // 存储启用的HTTP方法

)

// dailyWriter 每日日志写入器
// 实现按日期自动轮转日志文件的功能
type dailyWriter struct {
	currentFile *os.File   // 当前日志文件句柄
	currentDate string     // 当前日志文件的日期（YYYYMMDD）
	keepDays    int        // 保留日志的天数
	logDir      string     // 日志存放目录
	mu          sync.Mutex // 互斥锁，保证并发安全
}

// Write 实现io.Writer接口
// 写入日志数据到当前日志文件
// 参数:
//   - p: []byte - 要写入的数据
//
// 返回值:
//   - n: int - 写入的字节数
//   - err: error - 写入错误
func (w *dailyWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// 检查日期是否变化，需要轮转日志文件
	today := time.Now().Format("20060102")
	if w.currentDate != today {
		if err := w.rotate(today); err != nil {
			return 0, err
		}
	}

	return w.currentFile.Write(p)
}

// Sync 同步日志到磁盘
// 确保日志数据写入到磁盘文件
// 返回值: error - 同步错误
func (w *dailyWriter) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.currentFile != nil {
		return w.currentFile.Sync()
	}
	return nil
}

// Close 关闭日志文件
// 安全关闭当前日志文件句柄
// 返回值: error - 关闭错误
func (w *dailyWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.currentFile != nil {
		return w.currentFile.Close()
	}
	return nil
}

// rotate 轮转日志文件
// 根据新日期创建新的日志文件，并清理旧文件
// 参数:
//   - newDate: string - 新的日期字符串
//
// 返回值: error - 轮转错误
func (w *dailyWriter) rotate(newDate string) error {
	// 关闭当前文件
	if w.currentFile != nil {
		w.currentFile.Close()
	}

	// 确保日志目录存在
	if err := os.MkdirAll(w.logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 创建新的日志文件
	logFile := filepath.Join(w.logDir, fmt.Sprintf("logs_%s.log", newDate))
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("创建日志文件失败: %v", err)
	}

	w.currentFile = file
	w.currentDate = newDate

	// 异步清理旧日志文件
	go w.cleanOldLogs()

	return nil
}

// cleanOldLogs 清理旧日志文件
// 根据保留天数删除过期的日志文件
func (w *dailyWriter) cleanOldLogs() {
	// 获取所有日志文件
	files, err := os.ReadDir(w.logDir)
	if err != nil {
		return
	}

	// 计算清理时间
	cutoffTime := time.Now().AddDate(0, 0, -w.keepDays)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// 检查文件名格式
		if !strings.HasPrefix(file.Name(), "logs_") || !strings.HasSuffix(file.Name(), ".log") {
			continue
		}

		// 提取日期
		dateStr := strings.TrimPrefix(strings.TrimSuffix(file.Name(), ".log"), "logs_")
		if len(dateStr) != 8 { // YYYYMMDD
			continue
		}

		// 解析日期
		fileDate, err := time.Parse("20060102", dateStr)
		if err != nil {
			continue
		}

		// 如果文件日期早于清理时间，删除文件
		if fileDate.Before(cutoffTime) {
			filePath := filepath.Join(w.logDir, file.Name())
			os.Remove(filePath)
		}
	}
}

// InitLogger 初始化日志系统
// 配置Zap日志库，支持日志轮转、级别控制等功能
func InitLogger() {
	fmt.Println("[global] 日志初始化")

	// 初始化日志级别配置
	ZapLogLevels = make(map[string]bool)
	levels := Viper.GetStringSlice("logs.zap_log_levels")
	if len(levels) == 0 {
		levels = []string{"debug", "info", "warn", "error"}
	}
	for _, level := range levels {
		ZapLogLevels[strings.ToUpper(level)] = true
	}

	// 初始化Gin方法配置
	GinLogMethods = make(map[string]bool)
	methods := Viper.GetStringSlice("logs.gin_log_methods")
	for _, method := range methods {
		GinLogMethods[strings.ToUpper(method)] = true
	}

	// 获取配置
	zapSwitch := Viper.GetBool("logs.zaplogswitch")
	zapLogDays := Viper.GetInt("logs.zaplogdays")

	// 创建日志核心
	zapCore := createLogCore(zapSwitch, "runtime", zapLogDays)

	// 构建日志器
	if zapSwitch {
		ZapLog = zap.New(zapCore, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	} else {
		ZapLog = zap.NewNop()
	}

	ZapLog.Info("日志系统初始化完成")
}

// createLogCore 创建日志核心
// 根据配置创建Zap日志核心，支持文件和控制台输出
// 参数:
//   - enabled: bool - 是否启用日志
//   - logDir: string - 日志目录
//   - keepDays: int - 保留天数
//
// 返回值: zapcore.Core - 日志核心
func createLogCore(enabled bool, logDir string, keepDays int) zapcore.Core {
	if !enabled {
		return zapcore.NewNopCore()
	}

	dw := &dailyWriter{
		keepDays: keepDays,
		logDir:   logDir,
	}

	today := time.Now().Format("20060102")
	if err := dw.rotate(today); err != nil {
		panic(fmt.Sprintf("初始化日志失败: %v", err))
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05")
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.TimeKey = "time"
	encoderConfig.MessageKey = "msg"
	encoderConfig.LevelKey = "level"
	encoderConfig.CallerKey = "caller"

	levelFilter := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		levelStr := strings.ToUpper(lvl.String())
		_, ok := ZapLogLevels[levelStr]
		return ok
	})

	return zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(dw),
		levelFilter,
	)
}

// CronLogger Cron任务日志适配器
// 实现cron.Logger接口，用于Cron任务的日志记录
type CronLogger struct{}

// Info 记录Cron任务信息日志
// 参数:
//   - msg: string - 日志消息
//   - keysAndValues: ...interface{} - 键值对参数
func (l *CronLogger) Info(msg string, keysAndValues ...interface{}) {
	if ZapLog != nil {
		fields := make([]zap.Field, 0, len(keysAndValues)/2)
		for i := 0; i < len(keysAndValues); i += 2 {
			if i+1 < len(keysAndValues) {
				fields = append(fields, zap.Any(fmt.Sprintf("%v", keysAndValues[i]), keysAndValues[i+1]))
			}
		}
		ZapLog.Info(msg, fields...)
	}
}

// Error 记录Cron任务错误日志
// 参数:
//   - err: error - 错误信息
//   - msg: string - 日志消息
//   - keysAndValues: ...interface{} - 键值对参数
func (l *CronLogger) Error(err error, msg string, keysAndValues ...interface{}) {
	if ZapLog != nil {
		fields := make([]zap.Field, 0, len(keysAndValues)/2+1)
		fields = append(fields, zap.Error(err))
		for i := 0; i < len(keysAndValues); i += 2 {
			if i+1 < len(keysAndValues) {
				fields = append(fields, zap.Any(fmt.Sprintf("%v", keysAndValues[i]), keysAndValues[i+1]))
			}
		}
		ZapLog.Error(msg, fields...)
	}
}

// GinLogger Gin框架日志中间件
// 为Gin框架提供结构化的HTTP请求日志记录
// 返回值: gin.HandlerFunc - Gin中间件函数
func GinLogger() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		if ZapLog != nil {
			// 检查是否记录此HTTP方法
			if _, ok := GinLogMethods[param.Method]; !ok {
				return ""
			}

			ZapLog.Info("HTTP请求",
				zap.String("method", param.Method),
				zap.String("path", param.Path),
				zap.Int("status", param.StatusCode),
				zap.String("ip", param.ClientIP),
				zap.String("user_agent", param.Request.UserAgent()),
				zap.Duration("latency", param.Latency),
				zap.String("error", param.ErrorMessage),
			)
		}
		return ""
	})
}

// LogError 创建错误日志字段
// 将错误信息包装为日志字段，便于错误追踪
// 参数:
//   - err: error - 错误信息
//
// 返回值: zap.Field - 错误日志字段
func LogError(err error) zap.Field {
	return zap.Error(err)
}

// LogField 创建日志字段
// 用于创建结构化的日志字段，便于日志分析和查询
// 参数:
//   - key: string - 字段键名
//   - value: interface{} - 字段值
//
// 返回值: zap.Field - 日志字段
func LogField(key string, value interface{}) zap.Field {
	return zap.Any(key, value)
}

// LogInfo 记录信息日志
// 记录一般信息日志，用于程序运行状态跟踪
// 参数:
//   - msg: string - 日志消息
//   - fields: ...zap.Field - 可选字段
func LogInfo(msg string, fields ...zap.Field) {
	if ZapLog != nil {
		ZapLog.Info(msg, fields...)
	}
}

// LogWarn 记录警告日志
// 记录警告信息日志，用于潜在问题提醒
// 参数:
//   - msg: string - 日志消息
//   - fields: ...zap.Field - 可选字段
func LogWarn(msg string, fields ...zap.Field) {
	if ZapLog != nil {
		ZapLog.Warn(msg, fields...)
	}
}

// LogDebug 记录调试日志
// 记录调试信息日志，用于开发调试
// 参数:
//   - msg: string - 日志消息
//   - fields: ...zap.Field - 可选字段
func LogDebug(msg string, fields ...zap.Field) {
	if ZapLog != nil {
		ZapLog.Debug(msg, fields...)
	}
}

// LogFatal 记录致命错误日志
// 记录致命错误日志并退出程序
// 参数:
//   - msg: string - 日志消息
//   - fields: ...zap.Field - 可选字段
func LogFatal(msg string, fields ...zap.Field) {
	if ZapLog != nil {
		ZapLog.Fatal(msg, fields...)
	}
}

// LogWithContext 记录带上下文的日志
// 为日志添加上下文信息，便于追踪请求流程
// 参数:
//   - ctx: interface{} - 上下文
//   - msg: string - 日志消息
//   - fields: ...zap.Field - 可选字段
func LogWithContext(ctx interface{}, msg string, fields ...zap.Field) {
	if ZapLog != nil {
		// 添加上下文信息
		contextFields := []zap.Field{
			LogField("context", fmt.Sprintf("%v", ctx)),
			LogField("timestamp", time.Now().Format(time.RFC3339)),
		}
		allFields := append(contextFields, fields...)
		ZapLog.Info(msg, allFields...)
	}
}

// LogPerformance 记录性能日志
// 记录性能相关的日志，用于性能监控
// 参数:
//   - operation: string - 操作名称
//   - duration: time.Duration - 执行时间
//   - fields: ...zap.Field - 可选字段
func LogPerformance(operation string, duration time.Duration, fields ...zap.Field) {
	if ZapLog != nil {
		performanceFields := []zap.Field{
			LogField("operation", operation),
			LogField("duration_ms", duration.Milliseconds()),
		}
		allFields := append(performanceFields, fields...)
		ZapLog.Info("性能监控", allFields...)
	}
}

// LogSecurity 记录安全日志
// 记录安全相关的日志，用于安全审计
// 参数:
//   - event: string - 安全事件
//   - ip: string - IP地址
//   - user: string - 用户信息
//   - fields: ...zap.Field - 可选字段
func LogSecurity(event, ip, user string, fields ...zap.Field) {
	if ZapLog != nil {
		securityFields := []zap.Field{
			LogField("security_event", event),
			LogField("ip_address", ip),
			LogField("user", user),
			LogField("timestamp", time.Now().Format(time.RFC3339)),
		}
		allFields := append(securityFields, fields...)
		ZapLog.Warn("安全事件", allFields...)
	}
}

// SyncLog 同步日志
// 确保所有日志都写入到磁盘
func SyncLog() {
	if ZapLog != nil {
		ZapLog.Sync()
	}
}
