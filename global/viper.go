package global

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

var (
	Viper *viper.Viper
)

// 配置验证结构体
type ConfigValidator struct {
	Database DatabaseConfigValidator `mapstructure:"database"`
	MySQL    MySQLConfigValidator    `mapstructure:"db_mysql"`
	Server   ServerConfigValidator   `mapstructure:"server"`
	Logs     LogsConfigValidator     `mapstructure:"logs"`
}

type DatabaseConfigValidator struct {
	Type   string                `mapstructure:"type" validate:"required,oneof=mysql sqlite"`
	MySQL  MySQLConfigValidator  `mapstructure:"mysql"`
	SQLite SQLiteConfigValidator `mapstructure:"sqlite"`
}

type MySQLConfigValidator struct {
	Host            string `mapstructure:"host" validate:"required"`
	Port            string `mapstructure:"port" validate:"required"`
	Username        string `mapstructure:"username" validate:"required"`
	Password        string `mapstructure:"password" validate:"required"`
	Dbname          string `mapstructure:"dbname" validate:"required"`
	Charset         string `mapstructure:"charset"`
	TablePrefix     string `mapstructure:"tableprefix"`
	MaxOpenConns    int    `mapstructure:"maxopenconns"`
	MaxIdleConns    int    `mapstructure:"maxidleconns"`
	ConnMaxLifetime int    `mapstructure:"connmaxlifetime"`
}

type SQLiteConfigValidator struct {
	Path            string `mapstructure:"path" validate:"required"`
	TablePrefix     string `mapstructure:"tableprefix"`
	MaxOpenConns    int    `mapstructure:"maxopenconns"`
	MaxIdleConns    int    `mapstructure:"maxidleconns"`
	ConnMaxLifetime int    `mapstructure:"connmaxlifetime"`
}

type ServerConfigValidator struct {
	Port string `mapstructure:"port" validate:"required"`
}

type LogsConfigValidator struct {
	ZapLogSwitch bool     `mapstructure:"zaplogswitch"`
	ZapLogDays   int      `mapstructure:"zaplogdays"`
	ZapLogLevels []string `mapstructure:"zap_log_levels"`
}

func InitViper() error {
	Viper = viper.GetViper()

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
	}

	// 设置配置文件
	Viper.SetConfigName("config")
	Viper.SetConfigType("yaml")
	Viper.AddConfigPath(configDir)

	// 读取配置文件
	if err := Viper.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 设置默认值
	setDefaultValues()

	// 验证配置
	if err := validateConfig(); err != nil {
		return fmt.Errorf("配置验证失败: %v", err)
	}

	// 监听配置文件变化
	Viper.WatchConfig()

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
    port: 8080

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
	Viper.SetDefault("database.type", "mysql")
	Viper.SetDefault("database.mysql.charset", "utf8mb4")
	Viper.SetDefault("database.mysql.maxidleconns", 20)
	Viper.SetDefault("database.mysql.maxopenconns", 100)
	Viper.SetDefault("database.mysql.connmaxlifetime", 60)
	Viper.SetDefault("database.sqlite.tableprefix", "xiaohus_")
	Viper.SetDefault("database.sqlite.maxopenconns", 1)
	Viper.SetDefault("database.sqlite.maxidleconns", 1)
	Viper.SetDefault("database.sqlite.connmaxlifetime", 60)

	// 兼容旧配置的默认值
	Viper.SetDefault("db_mysql.charset", "utf8mb4")
	Viper.SetDefault("db_mysql.maxidleconns", 20)
	Viper.SetDefault("db_mysql.maxopenconns", 100)

	// 日志默认值
	Viper.SetDefault("logs.zaplogdays", 7)
	Viper.SetDefault("logs.zap_log_levels", []string{"info", "error", "warn"})
	Viper.SetDefault("logs.gin_log_methods", []string{})
	Viper.SetDefault("server.port", "8080")
}

// 验证配置
func validateConfig() error {
	var config ConfigValidator
	if err := Viper.Unmarshal(&config); err != nil {
		return fmt.Errorf("解析配置失败: %v", err)
	}

	// 验证数据库类型
	if config.Database.Type == "" {
		return fmt.Errorf("数据库类型不能为空")
	}

	// 根据数据库类型验证相应配置
	switch config.Database.Type {
	case "mysql":
		// 验证MySQL配置
		if config.Database.MySQL.Host == "" {
			return fmt.Errorf("MySQL主机地址不能为空")
		}
		if config.Database.MySQL.Port == "" {
			return fmt.Errorf("MySQL端口不能为空")
		}
		if config.Database.MySQL.Username == "" {
			return fmt.Errorf("MySQL用户名不能为空")
		}
		if config.Database.MySQL.Password == "" {
			return fmt.Errorf("MySQL密码不能为空")
		}
		if config.Database.MySQL.Dbname == "" {
			return fmt.Errorf("MySQL数据库名不能为空")
		}
	case "sqlite":
		// 验证SQLite配置
		if config.Database.SQLite.Path == "" {
			return fmt.Errorf("SQLite数据库文件路径不能为空")
		}
	default:
		return fmt.Errorf("不支持的数据库类型: %s", config.Database.Type)
	}

	// 验证服务器配置
	if config.Server.Port == "" {
		return fmt.Errorf("服务器端口不能为空")
	}

	// 验证日志配置
	if config.Logs.ZapLogDays <= 0 {
		return fmt.Errorf("Zap日志保留天数必须大于0")
	}

	return nil
}

// 获取数据库类型
func GetDatabaseType() string {
	return Viper.GetString("database.type")
}

// 获取MySQL配置
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

// 获取SQLite配置
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

// 获取配置值的安全方法
func GetString(key string) string {
	return Viper.GetString(key)
}

func GetInt(key string) int {
	return Viper.GetInt(key)
}

func GetBool(key string) bool {
	return Viper.GetBool(key)
}

func GetStringSlice(key string) []string {
	return Viper.GetStringSlice(key)
}
