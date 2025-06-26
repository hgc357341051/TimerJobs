package admins

import (
	"time"

	"gorm.io/gorm"
)

type Callback struct {
	Id           uint   `json:"id" gorm:"primary_key;type:int(11);"`
	Name         string `json:"name,omitempty"`
	CreatedAt    int    `json:"-"`
	CreatedAtStr string `json:"created_time,omitempty"`
	Remark       string `json:"remark,omitempty"`
	Mode         string `json:"mode,omitempty"`
	Content      string `json:"content"`
	UpdatedAt    int    `json:"-"`
	UpdatedAtStr string `json:"updated_time,omitempty"`
}

//func (t Tasks) TableName() string { return "xiaohus_tasks" }

// 获取任务列表
func (Callback) GetList(db *gorm.DB, where map[string]interface{}, fields string, order string, offset int, limit int) []Callback {
	var callback []Callback
	format := func(ts int) string {
		if ts <= 0 {
			return "/"
		}
		return time.Unix(int64(ts), 0).Format("2006-01-02 15:04:05")
	}
	err := db.Select(fields).Where(where).Order(order).Offset(offset).Limit(limit).Find(&callback).Error
	if err != nil {
		panic(err)
	}

	for i := range callback {
		callback[i].UpdatedAtStr = format(callback[i].UpdatedAt)
		callback[i].CreatedAtStr = format(callback[i].CreatedAt)
	}

	return callback
}

func (Callback) Read(db *gorm.DB, id string) Callback {
	var callback Callback
	format := func(ts int) string {
		if ts <= 0 {
			return "/"
		}
		return time.Unix(int64(ts), 0).Format("2006-01-02 15:04:05")
	}
	err := db.First(&callback, id).Error
	if err != nil {
		panic(err)
	}
	callback.UpdatedAtStr = format(callback.UpdatedAt)
	return callback
}
