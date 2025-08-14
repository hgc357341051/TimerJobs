package global

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/viper"
)

// Viper 全局配置管理器
var Viper *viper.Viper

// GlobalConfig 全局配置结构体
// 用于统一管理应用程序的所有配置项，避免频繁读取配置文件
// swagger:model GlobalConfig
type GlobalConfig struct {
	// App 应用程序基础配置
	App struct {
		Name    string `mapstructure:"name"`    // 应用名称
		Version string `mapstructure:"version"` // 应用版本
	} `mapstructure:"app"`

	// Jobs 任务系统配置
	Jobs struct {
		DefaultAllowMode      int  `mapstructure:"default_allow_mode"`
		ManualAllowConcurrent bool `mapstructure:"manual_allow_concurrent"`
		DefaultTimeoutSeconds int  `mapstructure:"default_timeout_seconds"`
		HTTPResponseMaxBytes  int  `mapstructure:"http_response_max_bytes"`
		LogSummaryEnabled     bool `mapstructure:"log_summary_enabled"`
		LogLineTruncate       int  `mapstructure:"log_line_truncate"`
	} `mapstructure:"jobs"`

	// Database 数据库配置
	Database struct {
		Type string `mapstructure:"type"` // 数据库类型: mysql 或 sqlite
		// MySQL 数据库配置
		MySQL struct {
			Charset      string `mapstructure:"charset"`      // 字符集
			DBName       string `mapstructure:"dbname"`       // 数据库名
			Host         string `mapstructure:"host"`         // 主机地址
			MaxIdleConns int    `mapstructure:"maxidleconns"` // 最大空闲连接数
			MaxOpenConns int    `mapstructure:"maxopenconns"` // 最大打开连接数
			Password     string `mapstructure:"password"`     // 密码
			Port         int    `mapstructure:"port"`         // 端口
			TablePrefix  string `mapstructure:"tableprefix"`  // 表前缀
			Username     string `mapstructure:"username"`     // 用户名
		} `mapstructure:"mysql"`
		// SQLite 数据库配置
		SQLite struct {
			Path         string `mapstructure:"path"`         // 数据库文件路径
			TablePrefix  string `mapstructure:"tableprefix"`  // 表前缀
			MaxOpenConns int    `mapstructure:"maxopenconns"` // 最大打开连接数
			MaxIdleConns int    `mapstructure:"maxidleconns"` // 最大空闲连接数
		} `mapstructure:"sqlite"`
	} `mapstructure:"database"`

	// Logs 日志配置
	Logs struct {
		ZapLogDays    int      `mapstructure:"zaplogdays"`      // 日志保留天数
		ZapLogSwitch  bool     `mapstructure:"zaplogswitch"`    // 日志开关
		ZapLogLevels  []string `mapstructure:"zap_log_levels"`  // 日志级别
		GinLogMethods []string `mapstructure:"gin_log_methods"` // Gin日志方法
	} `mapstructure:"logs"`

	// Server 服务器配置
	Server struct {
		Port string `mapstructure:"port"` // 服务端口
	} `mapstructure:"server"`

	// Service 服务配置
	Service struct {
		Name        string `mapstructure:"name"`         // 服务名称
		DisplayName string `mapstructure:"display_name"` // 显示名称
		Description string `mapstructure:"description"`  // 服务描述
	} `mapstructure:"service"`

	// Daemon 守护进程配置
	Daemon struct {
		MaxRestarts  int `mapstructure:"max_restarts"`  // 最大重启次数
		RestartDelay int `mapstructure:"restart_delay"` // 重启延迟（秒）
	} `mapstructure:"daemon"`

	// JobLogKeepCount 任务日志保留数量
	JobLogKeepCount int `mapstructure:"job_log_keep_count"`

	// IPControl IP访问控制配置
	IPControl struct {
		Enabled   bool     `mapstructure:"enabled"`   // 是否启用IP控制
		Whitelist []string `mapstructure:"whitelist"` // IP白名单
		Blacklist []string `mapstructure:"blacklist"` // IP黑名单
	} `mapstructure:"ip_control"`
}

// 全局配置实例和锁
var (
	GlobalConfigInstance *GlobalConfig // 全局配置实例
	configMutex          sync.RWMutex  // 配置读写锁，保证并发安全
	StartTime            time.Time     // 程序启动时间
)

// LoadGlobalConfig 加载全局配置
// 将Viper配置解析到全局配置结构体中，实现配置的统一管理
// 返回值: error - 加载失败时返回错误信息
func LoadGlobalConfig() error {
	configMutex.Lock()
	defer configMutex.Unlock()

	var config GlobalConfig
	if err := Viper.Unmarshal(&config); err != nil {
		return err
	}

	GlobalConfigInstance = &config
	return nil
}

// GetGlobalConfig 获取全局配置（只读）
// 返回全局配置实例的副本，保证线程安全
// 返回值: *GlobalConfig - 全局配置实例
func GetGlobalConfig() *GlobalConfig {
	configMutex.RLock()
	defer configMutex.RUnlock()
	return GlobalConfigInstance
}

// InitConfig 初始化配置系统
// 初始化Viper配置管理器，读取配置文件并加载到全局配置中
// 返回值: error - 初始化失败时返回错误信息
func InitConfig() error {
	// 确保配置目录存在
	configDir := "config"
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %v", err)
	}

	configFile := filepath.Join(configDir, "config.yaml")

	// 检查配置文件是否存在，不存在则创建默认配置
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		if err := createDefaultConfig(configFile); err != nil {
			return fmt.Errorf("创建默认配置文件失败: %v", err)
		}
		fmt.Printf("✅ 已创建默认配置文件: %s\n", configFile)
	}

	// 初始化Viper实例
	Viper = viper.New()

	// 设置配置文件路径
	Viper.AddConfigPath(configDir)
	Viper.SetConfigName("config")
	Viper.SetConfigType("yaml")

	// 读取配置文件
	if err := Viper.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 设置默认值
	setDefaultValues()

	// 加载全局配置
	if err := LoadGlobalConfig(); err != nil {
		return fmt.Errorf("加载全局配置失败: %v", err)
	}

	return nil
}

// 创建默认配置文件
func createDefaultConfig(configFile string) error {
	defaultConfig := `app:
    name: 小胡测试系统
    version: 1.0.0

# 数据库配置
# type: sqlite（推荐开发/单机） | mysql（推荐生产，需用环境变量覆盖）
database:
    type: sqlite  # sqlite 或 mysql
    
    # MySQL配置（生产环境建议用环境变量覆盖下方默认值）
    mysql:
        host: 127.0.0.1
        port: 3306
        username: root
        password: root123456
        dbname: xiaohu_jobs
        charset: utf8mb4
        tableprefix: xiaohus_
        maxopenconns: 100
        maxidleconns: 20
    
    # SQLite配置（推荐开发/单机，data 目录自动挂载）
    sqlite:
        path: data/xiaohu_jobs.db
        tableprefix: xiaohus_
        maxopenconns: 1
        maxidleconns: 1

logs:
    zaplogdays: 3
    zaplogswitch: true
    zap_log_levels: ["info","error", "warn"] # 可多选:debug/info/warn/error
    gin_log_methods: [] # POST/GET

server:
    port: 36363

# IP访问控制配置
ip_control:
    enabled: true                    # 是否启用IP控制
    whitelist:                       # IP白名单（优先级高）
        - "127.0.0.1"
        - "::1"
    blacklist:                       # IP黑名单
        # 黑名单可为空，因白名单模式会自动拒绝不在白名单中的IP
    # 注意：白名单优先级高于黑名单，如在白名单中则跳过黑名单检查

service:
    name: "XiaohuJobService"
    display_name: "小胡专用定时任务系统QQ357341051"
    description: "小胡专用跨平台任务调度服务"

daemon:
    max_restarts: 10      # 守护进程最大重启次数
    restart_delay: 5      # 重启间隔（秒）

job_log_keep_count: 3

# 生产环境建议用环境变量覆盖敏感配置，如：
#   DATABASE_TYPE=mysql
#   DATABASE_MYSQL_HOST=mysql
#   DATABASE_MYSQL_PORT=3306
#   DATABASE_MYSQL_USERNAME=xiaohu
#   DATABASE_MYSQL_PASSWORD=xiaohu123
#   DATABASE_MYSQL_DBNAME=xiaohu_jobs
#   DATABASE_MYSQL_CHARSET=utf8mb4
#   DATABASE_MYSQL_TABLEPREFIX=xiaohus_
#   DATABASE_MYSQL_MAXOPENCONNS=100
#   DATABASE_MYSQL_MAXIDLECONNS=20
`

	return os.WriteFile(configFile, []byte(defaultConfig), 0644)
}

// 设置默认值
func setDefaultValues() {
	// 数据库默认值
	Viper.SetDefault("database.type", "sqlite")
	Viper.SetDefault("database.mysql.charset", "utf8mb4")
	Viper.SetDefault("database.mysql.maxidleconns", 20)
	Viper.SetDefault("database.mysql.maxopenconns", 100)
	Viper.SetDefault("database.mysql.connmaxlifetime", 60)
	Viper.SetDefault("database.sqlite.tableprefix", "xiaohus_")
	Viper.SetDefault("database.sqlite.maxopenconns", 1)
	Viper.SetDefault("database.sqlite.maxidleconns", 1)
	Viper.SetDefault("database.sqlite.connmaxlifetime", 60)

	// 任务系统默认值
	Viper.SetDefault("jobs.default_allow_mode", 0)
	Viper.SetDefault("jobs.manual_allow_concurrent", true)
	Viper.SetDefault("jobs.default_timeout_seconds", 60)
	Viper.SetDefault("jobs.http_response_max_bytes", 1000)
	Viper.SetDefault("jobs.log_summary_enabled", true)
	Viper.SetDefault("jobs.log_line_truncate", 1000)

	// 兼容旧配置的默认值
	Viper.SetDefault("db_mysql.charset", "utf8mb4")
	Viper.SetDefault("db_mysql.maxidleconns", 20)
	Viper.SetDefault("db_mysql.maxopenconns", 100)

	// 日志默认值
	Viper.SetDefault("logs.zaplogdays", 7)
	Viper.SetDefault("logs.zap_log_levels", []string{"info", "error", "warn"})
	Viper.SetDefault("logs.gin_log_methods", []string{})
	Viper.SetDefault("server.port", "36363")
}

// Job 配置便捷获取
func GetJobsConfigInt(key string, def int) int {
	v := Viper.GetInt(key)
	if v == 0 {
		return def
	}
	return v
}

func GetJobsConfigBool(key string, def bool) bool {
	if !Viper.IsSet(key) {
		return def
	}
	return Viper.GetBool(key)
}

// validateCriticalConfig 验证关键配置
func validateCriticalConfig() error {
	// 验证数据库配置
	dbType := GetDatabaseType()
	if dbType == "" {
		return fmt.Errorf("数据库类型未配置")
	}

	switch dbType {
	case "mysql":
		config := GetMySQLConfig()
		if config.Host == "" || config.Port == "" || config.Username == "" || config.Dbname == "" {
			return fmt.Errorf("MySQL配置不完整")
		}
	case "sqlite":
		config := GetSQLiteConfig()
		if config.Path == "" {
			return fmt.Errorf("SQLite数据库路径未配置")
		}
	default:
		return fmt.Errorf("不支持的数据库类型: %s", dbType)
	}

	// 验证服务器配置
	port := Viper.GetString("server.port")
	if port == "" {
		return fmt.Errorf("服务器端口未配置")
	}

	return nil
}

// GetConfigValue 获取配置值
func GetConfigValue(key string) interface{} {
	return Viper.Get(key)
}

// GetConfigString 获取字符串配置
func GetConfigString(key string) string {
	return Viper.GetString(key)
}

// GetConfigInt 获取整数配置
func GetConfigInt(key string) int {
	return Viper.GetInt(key)
}

// GetConfigBool 获取布尔配置
func GetConfigBool(key string) bool {
	return Viper.GetBool(key)
}

// GetConfigStringSlice 获取字符串切片配置
func GetConfigStringSlice(key string) []string {
	return Viper.GetStringSlice(key)
}

// SetConfigValue 设置配置值（仅内存，需调用 SaveConfig 持久化）
func SetConfigValue(key string, value interface{}) {
	Viper.Set(key, value)
}

// SaveConfig 保存当前配置到文件
func SaveConfig() error {
	return Viper.WriteConfig()
}

// ReloadConfig 重新加载配置文件，支持热更新
func ReloadConfig() error {
	if err := Viper.ReadInConfig(); err != nil {
		return err
	}
	return LoadGlobalConfig()
}

// GetAppInfo 获取应用信息
func GetAppInfo() map[string]interface{} {
	return map[string]interface{}{
		"name":    Viper.GetString("app.name"),
		"version": Viper.GetString("app.version"),
		"port":    Viper.GetString("server.port"),
		"db_type": GetDatabaseType(),
	}
}

// GetServiceInfo 获取服务信息
func GetServiceInfo() map[string]interface{} {
	return map[string]interface{}{
		"name":         Viper.GetString("service.name"),
		"display_name": Viper.GetString("service.display_name"),
		"description":  Viper.GetString("service.description"),
	}
}

// GetLogConfig 获取日志配置
func GetLogConfig() map[string]interface{} {
	return map[string]interface{}{
		"zap_log_switch": Viper.GetBool("logs.zaplogswitch"),
		"zap_log_days":   Viper.GetInt("logs.zaplogdays"),
		"zap_log_levels": Viper.GetStringSlice("logs.zap_log_levels"),
	}
}

// GetDaemonConfig 获取守护进程配置
func GetDaemonConfig() map[string]interface{} {
	return map[string]interface{}{
		"max_restarts":  Viper.GetInt("daemon.max_restarts"),
		"restart_delay": Viper.GetInt("daemon.restart_delay"),
	}
}

// ValidateConfig 验证配置
func ValidateConfig() error {
	if ZapLog != nil {
		ZapLog.Info("开始验证配置")
	} else {
		fmt.Println("[配置] 开始验证配置")
	}

	// 验证关键配置
	if err := validateCriticalConfig(); err != nil {
		if ZapLog != nil {
			ZapLog.Error("关键配置验证失败", LogError(err))
		} else {
			fmt.Printf("[配置] 关键配置验证失败: %v\n", err)
		}
		return fmt.Errorf("配置验证失败: %v", err)
	}

	if ZapLog != nil {
		ZapLog.Info("配置验证完成")
	} else {
		fmt.Println("[配置] 配置验证完成")
	}
	return nil
}

// GetDatabaseInfo 获取数据库信息
// 返回当前数据库连接的详细信息
// 返回值: map[string]interface{} - 数据库信息
func GetDatabaseInfo() map[string]interface{} {
	if DB == nil {
		return map[string]interface{}{
			"status": "未连接",
		}
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return map[string]interface{}{
			"status": "获取连接失败",
			"error":  err.Error(),
		}
	}

	return map[string]interface{}{
		"status":         "已连接",
		"max_open_conns": sqlDB.Stats().MaxOpenConnections,
		"open_conns":     sqlDB.Stats().OpenConnections,
		"in_use":         sqlDB.Stats().InUse,
		"idle":           sqlDB.Stats().Idle,
	}
}

// UpdateIPControlConfig 更新IP控制配置
// 动态更新IP控制配置，支持运行时修改
// 参数:
//   - enabled: bool - 是否启用IP控制
//   - whitelist: []string - IP白名单列表
//   - blacklist: []string - IP黑名单列表
func UpdateIPControlConfig(enabled bool, whitelist, blacklist []string) {
	configMutex.Lock()
	defer configMutex.Unlock()

	if GlobalConfigInstance != nil {
		GlobalConfigInstance.IPControl.Enabled = enabled
		GlobalConfigInstance.IPControl.Whitelist = whitelist
		GlobalConfigInstance.IPControl.Blacklist = blacklist
	}

	// 同时更新Viper配置，保持一致性
	Viper.Set("ip_control.enabled", enabled)
	Viper.Set("ip_control.whitelist", whitelist)
	Viper.Set("ip_control.blacklist", blacklist)
}

// GetIPControlConfig 获取IP控制配置
// 返回IP控制的当前配置状态
// 返回值:
//   - bool - 是否启用IP控制
//   - []string - IP白名单列表
//   - []string - IP黑名单列表
func GetIPControlConfig() (bool, []string, []string) {
	configMutex.RLock()
	defer configMutex.RUnlock()

	if GlobalConfigInstance != nil {
		return GlobalConfigInstance.IPControl.Enabled,
			GlobalConfigInstance.IPControl.Whitelist,
			GlobalConfigInstance.IPControl.Blacklist
	}
	return false, nil, nil
}

// GetDatabaseType 获取数据库类型
func GetDatabaseType() string {
	return Viper.GetString("database.type")
}

// GetMySQLConfig 获取MySQL配置
func GetMySQLConfig() *MySQLConfig {
	var config MySQLConfig
	if err := Viper.UnmarshalKey("database.mysql", &config); err != nil {
		// 记录错误但返回默认配置
		if ZapLog != nil {
			ZapLog.Error("解析MySQL配置失败", LogError(err))
		}
	}
	return &config
}

// GetSQLiteConfig 获取SQLite配置
func GetSQLiteConfig() *SQLiteConfig {
	var config SQLiteConfig
	if err := Viper.UnmarshalKey("database.sqlite", &config); err != nil {
		// 记录错误但返回默认配置
		if ZapLog != nil {
			ZapLog.Error("解析SQLite配置失败", LogError(err))
		}
	}
	return &config
}
