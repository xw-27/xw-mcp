// Package config 提供 YAML 配置文件的热重载加载功能。
package config

import (
	"fmt"
	"strconv"
)

// Val 配置值包装类型
//
// 用于存储配置项的值和存在性状态。提供类型安全的访问方法。
//
// 字段说明:
//   - val: 原始配置值，interface{} 类型
//   - ok: 标识配置项是否存在于配置中
type Val struct {
	val interface{}
	ok  bool
}

// exists 判断配置值是否存在
//
// 返回: true 表示配置项存在且值不为 nil；false 表示未找到或值为 nil
func (v *Val) Exists() bool {
	return v.ok
}

// toString 获取字符串值
//
// 返回: 配置值转换为字符串；如果未找到或类型不匹配返回空字符串 ""
func (v *Val) ToString() string {
	if !v.ok || v.val == nil {
		return ""
	}
	return fmt.Sprintf("%v", v.val)
}

// toInt 获取整数值
//
// 支持类型: float64、int、string
//   - float64: 转换为 int（截断）
//   - int: 直接返回
//   - string: 尝试解析为整数，解析失败返回 0
//
// 返回: 配置值转换为整数；如果未找到或类型不匹配返回 0
func (v *Val) ToInt() int {
	if !v.ok || v.val == nil {
		return 0
	}
	switch val := v.val.(type) {
	case float64:
		return int(val)
	case int:
		return val
	case string:
		i, _ := strconv.Atoi(val)
		return i
	}
	return 0
}

// toFloat 获取浮点数值
//
// 返回: 配置值转换为 float64；如果未找到或类型不是 float64 返回 0.0
func (v *Val) ToFloat() float64 {
	if !v.ok || v.val == nil {
		return 0
	}
	if f, ok := v.val.(float64); ok {
		return f
	}
	return 0
}

// toBool 获取布尔值
//
// 返回: 配置值转换为 bool；如果未找到或类型不是 bool 返回 false
func (v *Val) ToBool() bool {
	if !v.ok || v.val == nil {
		return false
	}
	if b, ok := v.val.(bool); ok {
		return b
	}
	return false
}

// toMap 获取对象/字典值
//
// 返回: 配置值转换为 map[string]interface{}；如果未找到或类型不匹配返回 nil
func (v *Val) ToMap() map[string]interface{} {
	if !v.ok || v.val == nil {
		return nil
	}
	if m, ok := v.val.(map[string]interface{}); ok {
		return m
	}
	return nil
}

// toSlice 获取数组值
//
// 返回: 配置值转换为 []interface{}；如果未找到或类型不匹配返回 nil
func (v *Val) ToSlice() []interface{} {
	if !v.ok || v.val == nil {
		return nil
	}
	if s, ok := v.val.([]interface{}); ok {
		return s
	}
	return nil
}