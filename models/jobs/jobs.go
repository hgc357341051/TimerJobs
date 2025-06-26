package jobs

import (
	"time"
)

// Jobs 定时任务模型
// swagger:model Jobs
// 示例：{"id":1,"name":"测试任务","desc":"描述","cron_expr":"* * * * * *","mode":"command","command":"echo hello","state":0,"allow_mode":0,"max_run_count":0,"run_count":2,"created_at":"2024-06-25T12:00:00Z","updated_at":"2024-06-25T12:00:00Z"}
type Jobs struct {
	ID          uint      `gorm:"primaryKey;autoIncrement:true" json:"id"` // 主键ID
	Name        string    `gorm:"size:100;not null;comment:任务名称" json:"name"`
	Desc        string    `gorm:"size:500;comment:任务描述" json:"desc"`
	CronExpr    string    `gorm:"size:100;not null;comment:cron表达式" json:"cron_expr"`
	Mode        string    `gorm:"size:20;not null;default:'http';comment:执行模式" json:"mode"` // http/command/func
	Command     string    `gorm:"type:text;not null;comment:执行命令或URL" json:"command"`
	State       int       `gorm:"type:tinyint;default:0;comment:任务状态" json:"state"`      // 0等待 1执行中 2停止
	AllowMode   int       `gorm:"type:tinyint;default:0;comment:执行模式" json:"allow_mode"` // 0默认并行 1串行 2立即执行
	MaxRunCount uint      `gorm:"default:0;comment:最大执行次数" json:"max_run_count"`         // 0=无限制
	RunCount    uint      `gorm:"default:0;comment:已执行次数" json:"run_count"`
	CreatedAt   time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP;comment:创建时间" json:"created_at"`
	UpdatedAt   time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP;comment:更新时间" json:"updated_at"`
}

// TableName 指定表名
func (Jobs) TableName() string {
	return "xiaohus_jobs"
}
