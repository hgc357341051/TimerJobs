package admins

import (
	"time"
)

// Admin 管理员模型
type Admin struct {
	ID        uint      `gorm:"primaryKey;autoIncrement:true" json:"id"`
	Username  string    `gorm:"size:50;not null;unique;comment:用户名" json:"username"`
	Password  string    `gorm:"size:100;not null;comment:密码" json:"-"`
	Email     string    `gorm:"size:100;comment:邮箱" json:"email"`
	Role      string    `gorm:"size:20;default:'admin';comment:角色" json:"role"`
	Status    int       `gorm:"type:tinyint;default:1;comment:状态" json:"status"` // 0禁用 1启用
	LastLogin time.Time `gorm:"type:timestamp;comment:最后登录时间" json:"last_login"`
	CreatedAt time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP;comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP;comment:更新时间" json:"updated_at"`
}

// TableName 指定表名
func (Admin) TableName() string {
	return "xiaohus_admins"
}

// SystemConfig 系统配置模型
type SystemConfig struct {
	ID          uint      `gorm:"primaryKey;autoIncrement:true" json:"id"`
	ConfigKey   string    `gorm:"size:100;not null;unique;comment:配置键" json:"config_key"`
	ConfigValue string    `gorm:"type:text;comment:配置值" json:"config_value"`
	Description string    `gorm:"size:200;comment:配置描述" json:"description"`
	CreatedAt   time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP;comment:创建时间" json:"created_at"`
	UpdatedAt   time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP;comment:更新时间" json:"updated_at"`
}

// TableName 指定表名
func (SystemConfig) TableName() string {
	return "xiaohus_system_configs"
}
