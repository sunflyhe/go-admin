// 系统参数的 GORM 模型。表结构以 SQL migration 为准,模型仅作查询映射。
package model

import (
	"time"
)

// SysConfig 系统参数。键值对由管理端维护,业务模块按需读取;
// Builtin 标记的参数是底座预留的保护位:可改值,不可删、不可改键名。
type SysConfig struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:64" json:"name"`
	ConfigKey   string    `gorm:"column:config_key;size:64;uniqueIndex" json:"key"`
	Value     string    `gorm:"size:512" json:"value"`
	Remark    string    `gorm:"size:255" json:"remark"`
	Builtin   bool      `gorm:"default:false" json:"builtin"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (SysConfig) TableName() string { return "sys_config" }
