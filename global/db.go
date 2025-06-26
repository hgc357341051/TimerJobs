// 在function/db.go文件中确保InitDB函数正确初始化DB
package global

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"xiaohuAdmin/models/admins"
	"xiaohuAdmin/models/jobs"

	"gorm.io/driver/mysql"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

// DB 全局数据库连接实例
var DB *gorm.DB

type MySQLConfig struct {
	Host            string `mapstructure:"host"`
	Port            string `mapstructure:"port"`
	Username        string `mapstructure:"username"`
	Password        string `mapstructure:"password"`
	Dbname          string `mapstructure:"dbname"`
	Charset         string `mapstructure:"charset"`
	TablePrefix     string `mapstructure:"tableprefix"`
	MaxOpenConns    int    `mapstructure:"maxopenconns"`
	MaxIdleConns    int    `mapstructure:"maxidleconns"`
	ConnMaxLifetime int    `mapstructure:"connmaxlifetime"`
}

type SQLiteConfig struct {
	Path            string `mapstructure:"path"`
	TablePrefix     string `mapstructure:"tableprefix"`
	MaxOpenConns    int    `mapstructure:"maxopenconns"`
	MaxIdleConns    int    `mapstructure:"maxidleconns"`
	ConnMaxLifetime int    `mapstructure:"connmaxlifetime"`
}

// InitDB 初始化数据库连接
// 根据配置文件中的数据库类型创建相应的数据库连接
// 支持MySQL和SQLite两种数据库类型
// 返回值: error - 初始化失败时返回错误信息
func InitDB() error {
	if ZapLog != nil {
		ZapLog.Info("开始初始化数据库连接")
	} else {
		fmt.Println("[数据库] 开始初始化数据库连接")
	}

	// 获取数据库类型
	dbType := GetDatabaseType()
	if ZapLog != nil {
		ZapLog.Info("数据库类型", LogField("type", dbType))
	} else {
		fmt.Printf("[数据库] 数据库类型: %s\n", dbType)
	}

	var err error
	switch dbType {
	case "mysql":
		err = initMySQLDB()
	case "sqlite":
		err = initSQLiteDB()
	default:
		return fmt.Errorf("不支持的数据库类型: %s", dbType)
	}

	if err != nil {
		if ZapLog != nil {
			ZapLog.Error("数据库初始化失败", LogError(err))
		} else {
			fmt.Printf("[数据库] 数据库初始化失败: %v\n", err)
		}
		return err
	}

	// 自动迁移表结构
	if err := autoMigrate(); err != nil {
		if ZapLog != nil {
			ZapLog.Error("数据库迁移失败", LogError(err))
		} else {
			fmt.Printf("[数据库] 数据库迁移失败: %v\n", err)
		}
		return err
	}

	// 初始化默认数据
	if err := initDefaultData(); err != nil {
		if ZapLog != nil {
			ZapLog.Error("初始化默认数据失败", LogError(err))
		} else {
			fmt.Printf("[数据库] 初始化默认数据失败: %v\n", err)
		}
		return err
	}

	if ZapLog != nil {
		ZapLog.Info("数据库初始化完成")
	} else {
		fmt.Println("[数据库] 数据库初始化完成")
	}
	return nil
}

// initMySQLDB 初始化MySQL数据库连接
// 使用GORM连接MySQL数据库，配置连接池参数
// 返回值: error - 连接失败时返回错误信息
func initMySQLDB() error {
	config := GetMySQLConfig()

	// 构建DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=Local&timeout=10s&readTimeout=30s&writeTimeout=30s",
		config.Username,
		config.Password,
		config.Host,
		config.Port,
		config.Dbname,
		config.Charset,
	)

	// 配置GORM日志级别
	gormLogger := logger.Default.LogMode(logger.Info)
	if !GlobalConfigInstance.Logs.ZapLogSwitch {
		gormLogger = logger.Default.LogMode(logger.Silent)
	}

	// 连接MySQL数据库
	var err error
	DB, err = gorm.Open(mysql.New(mysql.Config{
		DSN:                       dsn,
		DefaultStringSize:         256,
		DisableDatetimePrecision:  true,
		DontSupportRenameIndex:    true,
		DontSupportRenameColumn:   true,
		SkipInitializeWithVersion: false,
	}), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   config.TablePrefix,
			SingularTable: true,
		},
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	})
	if err != nil {
		return fmt.Errorf("连接MySQL数据库失败: %v", err)
	}

	// 获取底层sql.DB对象
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("获取MySQL数据库实例失败: %v", err)
	}

	// 配置连接池
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(config.ConnMaxLifetime) * time.Minute)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("MySQL数据库连接测试失败: %v", err)
	}

	if ZapLog != nil {
		ZapLog.Info("MySQL数据库连接成功",
			LogField("host", config.Host),
			LogField("port", config.Port),
			LogField("database", config.Dbname))
	} else {
		fmt.Printf("[数据库] MySQL数据库连接成功 - host:%s port:%s database:%s\n", config.Host, config.Port, config.Dbname)
	}

	return nil
}

// initSQLiteDB 初始化SQLite数据库连接
// 使用GORM连接SQLite数据库，配置连接池参数
// 返回值: error - 连接失败时返回错误信息
func initSQLiteDB() error {
	config := GetSQLiteConfig()

	if ZapLog != nil {
		ZapLog.Info("开始初始化SQLite数据库", LogField("path", config.Path))
	} else {
		fmt.Printf("[数据库] 开始初始化SQLite数据库: %s\n", config.Path)
	}

	// 确保SQLite数据库文件目录存在
	dbDir := filepath.Dir(config.Path)
	if ZapLog != nil {
		ZapLog.Info("创建数据库目录", LogField("dir", dbDir))
	} else {
		fmt.Printf("[数据库] 创建数据库目录: %s\n", dbDir)
	}
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		if ZapLog != nil {
			ZapLog.Error("创建SQLite数据库目录失败", LogError(err))
		} else {
			fmt.Printf("[数据库] 创建SQLite数据库目录失败: %v\n", err)
		}
		return fmt.Errorf("创建SQLite数据库目录失败: %v", err)
	}

	// 如果数据库文件不存在则创建空文件
	if _, err := os.Stat(config.Path); os.IsNotExist(err) {
		file, err := os.Create(config.Path)
		if err != nil {
			if ZapLog != nil {
				ZapLog.Error("创建SQLite数据库文件失败", LogError(err))
			} else {
				fmt.Printf("[数据库] 创建SQLite数据库文件失败: %v\n", err)
			}
			return fmt.Errorf("创建SQLite数据库文件失败: %v", err)
		}
		file.Close()
		if ZapLog != nil {
			ZapLog.Info("已创建空的SQLite数据库文件", LogField("path", config.Path))
		} else {
			fmt.Printf("[数据库] 已创建空的SQLite数据库文件: %s\n", config.Path)
		}
	}

	// 配置GORM日志级别
	gormLogger := logger.Default.LogMode(logger.Info)
	if !GlobalConfigInstance.Logs.ZapLogSwitch {
		gormLogger = logger.Default.LogMode(logger.Silent)
	}

	// 连接SQLite数据库（新版写法，支持modernc.org/sqlite扩展）
	if ZapLog != nil {
		ZapLog.Info("正在连接SQLite数据库", LogField("path", config.Path))
	} else {
		fmt.Printf("[数据库] 正在连接SQLite数据库: %s\n", config.Path)
	}
	dsn := fmt.Sprintf("file:%s?cache=shared&mode=rwc&_pragma=journal_mode(WAL)", config.Path)
	var err error
	DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   config.TablePrefix,
			SingularTable: true,
		},
		Logger: gormLogger,
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	})
	if err != nil {
		if ZapLog != nil {
			ZapLog.Error("SQLite数据库连接失败", LogError(err))
		} else {
			fmt.Printf("[数据库] SQLite数据库连接失败: %v\n", err)
		}
		return fmt.Errorf("SQLite数据库连接失败: %v", err)
	}

	// 获取底层sql.DB对象
	if ZapLog != nil {
		ZapLog.Info("获取数据库实例")
	} else {
		fmt.Println("[数据库] 获取数据库实例")
	}
	sqlDB, err := DB.DB()
	if err != nil {
		if ZapLog != nil {
			ZapLog.Error("获取SQLite数据库实例失败", LogError(err))
		} else {
			fmt.Printf("[数据库] 获取SQLite数据库实例失败: %v\n", err)
		}
		return fmt.Errorf("获取SQLite数据库实例失败: %v", err)
	}

	// 配置连接池（SQLite连接池配置较小）
	if ZapLog != nil {
		ZapLog.Info("配置连接池",
			LogField("max_open_conns", config.MaxOpenConns),
			LogField("max_idle_conns", config.MaxIdleConns))
	} else {
		fmt.Printf("[数据库] 配置连接池 - max_open_conns:%d max_idle_conns:%d\n", config.MaxOpenConns, config.MaxIdleConns)
	}
	sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Duration(config.ConnMaxLifetime) * time.Minute)

	// 测试连接
	if ZapLog != nil {
		ZapLog.Info("测试数据库连接")
	} else {
		fmt.Println("[数据库] 测试数据库连接")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		if ZapLog != nil {
			ZapLog.Error("SQLite数据库连接测试失败", LogError(err))
		} else {
			fmt.Printf("[数据库] SQLite数据库连接测试失败: %v\n", err)
		}
		return fmt.Errorf("SQLite数据库连接测试失败: %v", err)
	}

	if ZapLog != nil {
		ZapLog.Info("数据库实例已设置")
	} else {
		fmt.Println("[数据库] 数据库实例已设置")
	}

	if ZapLog != nil {
		ZapLog.Info("SQLite数据库连接成功", LogField("path", config.Path))
	} else {
		fmt.Printf("[数据库] SQLite数据库连接成功: %s\n", config.Path)
	}
	return nil
}

// CloseDB 关闭数据库连接
// 安全关闭数据库连接，释放资源
func CloseDB() {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err == nil {
			sqlDB.Close()
			if ZapLog != nil {
				ZapLog.Info("数据库连接已关闭")
			} else {
				fmt.Println("[数据库] 数据库连接已关闭")
			}
		}
	}
}

// autoMigrate 自动迁移表结构
func autoMigrate() error {
	if ZapLog != nil {
		ZapLog.Info("开始自动迁移表结构")
	} else {
		fmt.Println("[数据库] 开始自动迁移表结构")
	}

	// 迁移所有模型（去除JobLog表）
	err := DB.AutoMigrate(
		&jobs.Jobs{},
		&admins.Admin{},
		&admins.SystemConfig{},
	)

	if err != nil {
		if ZapLog != nil {
			ZapLog.Error("数据库迁移失败", LogError(err))
		} else {
			fmt.Printf("[数据库] 数据库迁移失败: %v\n", err)
		}
		return fmt.Errorf("数据库迁移失败: %v", err)
	}

	if ZapLog != nil {
		ZapLog.Info("数据库迁移完成")
	} else {
		fmt.Println("[数据库] 数据库迁移完成")
	}
	return nil
}

// initDefaultData 初始化默认数据
func initDefaultData() error {
	if ZapLog != nil {
		ZapLog.Info("开始初始化默认数据")
	} else {
		fmt.Println("[数据库] 开始初始化默认数据")
	}

	// 检查是否已有管理员账户
	var adminCount int64
	DB.Model(&admins.Admin{}).Count(&adminCount)

	if adminCount == 0 {
		// 创建默认管理员账户
		defaultAdmin := admins.Admin{
			Username: "admin",
			Password: "admin123", // 实际项目中应该使用加密密码
			Email:    "admin@example.com",
			Role:     "admin",
			Status:   1,
		}

		if err := DB.Create(&defaultAdmin).Error; err != nil {
			if ZapLog != nil {
				ZapLog.Error("创建默认管理员失败", LogError(err))
			} else {
				fmt.Printf("[数据库] 创建默认管理员失败: %v\n", err)
			}
			return err
		}

		if ZapLog != nil {
			ZapLog.Info("已创建默认管理员账户", LogField("username", "admin"))
		} else {
			fmt.Println("[数据库] 已创建默认管理员账户: admin")
		}
	}

	// 初始化系统配置
	initSystemConfigs()

	if ZapLog != nil {
		ZapLog.Info("默认数据初始化完成")
	} else {
		fmt.Println("[数据库] 默认数据初始化完成")
	}
	return nil
}

// initSystemConfigs 初始化系统配置
func initSystemConfigs() {
	configs := []admins.SystemConfig{
		{
			ConfigKey:   "system_name",
			ConfigValue: "小胡定时任务系统",
			Description: "系统名称",
		},
		{
			ConfigKey:   "system_version",
			ConfigValue: "1.0.0",
			Description: "系统版本",
		},
		{
			ConfigKey:   "max_concurrent_jobs",
			ConfigValue: "10",
			Description: "最大并发任务数",
		},
		{
			ConfigKey:   "job_timeout",
			ConfigValue: "300",
			Description: "任务超时时间(秒)",
		},
	}

	for _, config := range configs {
		var existingConfig admins.SystemConfig
		if err := DB.Where("config_key = ?", config.ConfigKey).First(&existingConfig).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				DB.Create(&config)
			}
		}
	}
}

// PingDB 测试数据库连接
func PingDB() error {
	if DB == nil {
		return fmt.Errorf("数据库未初始化")
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("获取数据库实例失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return sqlDB.PingContext(ctx)
}

// GetDB 获取数据库连接实例
// 返回全局数据库连接实例，用于其他模块访问数据库
// 返回值: *gorm.DB - 数据库连接实例
func GetDB() *gorm.DB {
	return DB
}

// IsDBConnected 检查数据库连接状态
// 检查数据库是否正常连接
// 返回值: bool - 连接状态
func IsDBConnected() bool {
	if DB == nil {
		return false
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return false
	}

	// 尝试ping数据库
	err = sqlDB.Ping()
	return err == nil
}

// GetDBStats 获取数据库统计信息
// 返回数据库连接池的详细统计信息
// 返回值: map[string]interface{} - 数据库统计信息
func GetDBStats() map[string]interface{} {
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

	stats := sqlDB.Stats()
	return map[string]interface{}{
		"status":                "已连接",
		"max_ope n_connections": stats.MaxOpenConnections,
		"ope n_connections":     stats.OpenConnections,
		"in_use":                stats.InUse,
		"idle":                  stats.Idle,
		"wait_count":            stats.WaitCount,
		" wait_duration":        stats.WaitDuration.String(),
		"ma x_idle_closed":      stats.MaxIdleClosed,
		"max_li fetime_closed":  stats.MaxLifetimeClosed,
	}
}
