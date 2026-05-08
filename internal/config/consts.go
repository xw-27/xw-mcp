package config

// ConfigType 配置值类型枚举
//
// 支持的类型:
//   - ConfigTypeString: 字符串
//   - ConfigTypeInt: 整数
//   - ConfigTypeFloat: 浮点数
//   - ConfigTypeBool: 布尔值
//   - ConfigTypeArray: 数组
//   - ConfigTypeObject: 对象/字典
//   - ConfigTypeNull: 空值
type ConfigType int

const (
	ConfigTypeString  ConfigType = iota // 字符串类型
	ConfigTypeInt                       // 整数类型
	ConfigTypeFloat                     // 浮点数类型
	ConfigTypeBool                      // 布尔类型
	ConfigTypeArray                     // 数组类型
	ConfigTypeObject                    // 对象类型
	ConfigTypeNull                      // 空值类型
)