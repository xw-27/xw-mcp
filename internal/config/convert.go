package config

import (
	"fmt"
	"strconv"
)

// Value 配置值包装类型
//
// 用于 Go 端获取配置，提供类型安全的访问方法
type Value struct {
	val interface{}
	ok  bool
}

// Ok 判断配置值是否有效
//
// 返回:
//   - bool: 配置项是否存在且值不为 nil
func (v *Value) Ok() bool {
	return v.ok
}

// String 获取字符串值
//
// 返回:
//   - string: 配置值转换为字符串，未找到或类型不匹配时返回空字符串
func (v *Value) String() string {
	if !v.ok || v.val == nil {
		return ""
	}
	return fmt.Sprintf("%v", v.val)
}

// Int 获取整数值
//
// 支持 float64、int、string 类型转换，string 类型会尝试解析为整数
//
// 返回:
//   - int: 配置值转换为整数，未找到或类型不匹配时返回 0
func (v *Value) Int() int {
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

// Float 获取浮点数值
//
// 返回:
//   - float64: 配置值转换为浮点数，未找到或类型不匹配时返回 0.0
func (v *Value) Float() float64 {
	if !v.ok || v.val == nil {
		return 0
	}
	if f, ok := v.val.(float64); ok {
		return f
	}
	return 0
}

// Bool 获取布尔值
//
// 返回:
//   - bool: 配置值转换为布尔值，未找到或类型不匹配时返回 false
func (v *Value) Bool() bool {
	if !v.ok || v.val == nil {
		return false
	}
	if b, ok := v.val.(bool); ok {
		return b
	}
	return false
}

// Map 获取对象/字典值
//
// 返回:
//   - map[string]interface{}: 配置值转换为 map，未找到或类型不匹配时返回 nil
func (v *Value) Map() map[string]interface{} {
	if !v.ok || v.val == nil {
		return nil
	}
	if m, ok := v.val.(map[string]interface{}); ok {
		return m
	}
	return nil
}

// Slice 获取数组值
//
// 返回:
//   - []interface{}: 配置值转换为切片，未找到或类型不匹配时返回 nil
func (v *Value) Slice() []interface{} {
	if !v.ok || v.val == nil {
		return nil
	}
	if s, ok := v.val.([]interface{}); ok {
		return s
	}
	return nil
}

// convertType 将配置值转换为指定类型
//
// 参数:
//   - val: 原始配置值
//   - typ: 目标类型
//
// 返回:
//   - interface{}: 转换后的值，类型不匹配时返回 nil
func convertType(val interface{}, typ ConfigType) interface{} {
	if val == nil {
		return nil
	}
	switch typ {
	case ConfigTypeString:
		return fmt.Sprintf("%v", val)
	case ConfigTypeInt:
		switch v := val.(type) {
		case float64:
			return int(v)
		case int:
			return v
		case string:
			i, _ := strconv.Atoi(v)
			return i
		}
	case ConfigTypeFloat:
		if v, ok := val.(float64); ok {
			return v
		}
	case ConfigTypeBool:
		if v, ok := val.(bool); ok {
			return v
		}
	case ConfigTypeArray:
		if v, ok := val.([]interface{}); ok {
			return v
		}
	case ConfigTypeObject:
		if v, ok := val.(map[string]interface{}); ok {
			return v
		}
	case ConfigTypeNull:
		return nil
	}
	return nil
}