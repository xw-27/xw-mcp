package sql

// Config 数据库连接配置
type Config struct {
	DriverName string // 驱动名称：sqlite3, mysql 等
	DSN        string // 数据库连接字符串，格式由驱动定义
}

// QueryResult 查询结果
type QueryResult struct {
	Columns     []string
	Rows        []map[string]interface{}
	RowsAffected int64
}

// ExecResult 执行结果
type ExecResult struct {
	RowsAffected int64
	LastInsertId int64
}

// supportedDrivers 支持的驱动集合
var supportedDrivers = map[string]struct{}{
	"sqlite3": {},
	"mysql":   {},
}