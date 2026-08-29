// 字典域的 GORM 模型。表结构以 SQL migration 为准,模型仅作查询映射。
package model

import (
	"time"
)

// 字典项状态。
const (
	DictItemStatusEnabled  = 1 // 启用:业务按类型键取子项时只返回启用项
	DictItemStatusDisabled = 2 // 停用:保留数据但不下发
)

// SysDictType 字典类型:一组可枚举子项的容器(如 性别/证件类型)。
// 业务模块按 dict_key 读取,不感知子项 ID。
type SysDictType struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:64" json:"name"`
	DictKey   string    `gorm:"column:dict_key;size:64;uniqueIndex" json:"key"`
	Remark    string    `gorm:"size:255" json:"remark"`
	Builtin   bool      `gorm:"default:false" json:"builtin"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func (SysDictType) TableName() string { return "sys_dict_type" }

// SysDictItem 字典项:类型下的子数据。Value 是业务存储值(字符串,兼容非数字编码),
// Label 是展示文本;同一类型内 value 唯一。TagType 是前端标签配色标记,
// 业务读取后可按它把选项渲染成彩色 tag。
type SysDictItem struct {
	ID          int64     `gorm:"primaryKey" json:"id"`
	TypeID      int64     `gorm:"index;uniqueIndex:uk_dict_item_type_value" json:"typeId"`
	Label       string    `gorm:"size:64" json:"label"`
	Description string    `gorm:"size:255" json:"description"`
	Value       string    `gorm:"size:128;uniqueIndex:uk_dict_item_type_value" json:"value"`
	Sort        int       `gorm:"default:0" json:"sort"`
	TagType     string    `gorm:"size:32" json:"tagType"`
	Status      int       `gorm:"default:1" json:"status"`
	Remark      string    `gorm:"size:255" json:"remark"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (SysDictItem) TableName() string { return "sys_dict_item" }
