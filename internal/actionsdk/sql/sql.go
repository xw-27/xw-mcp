package sql

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strconv"

	_ "github.com/go-sql-driver/mysql" // 空白导入，注册 MySQL 驱动
	_ "github.com/mattn/go-sqlite3"

	"github.com/dop251/goja"
)

var ErrUnsupportedDriver = errors.New("unsupported driver")

// query 执行查询操作
func query(db *sql.DB, sqlStr string, args ...interface{}) (QueryResult, error) {
	// 展开 []interface{} 为单个参数（JS 传入数组时会包装成 []interface{}）
	if len(args) == 1 {
		if arr, ok := args[0].([]interface{}); ok {
			args = arr
		}
	}

	rows, err := db.Query(sqlStr, args...)
	if err != nil {
		return QueryResult{}, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return QueryResult{}, err
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return QueryResult{}, err
	}

	results := make([]map[string]interface{}, 0)
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}
		if err := rows.Scan(valuePtrs...); err != nil {
			return QueryResult{}, err
		}
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]

			if val == nil {
				row[col] = nil
				continue
			}
			// 核心逻辑：根据数据库列的真实类型转换
			if b, ok := val.([]byte); ok {
				scanType := columnTypes[i].ScanType()
				// 如果目标类型是字符串，直接转 string
				if scanType.Kind() == reflect.String {
					row[col] = string(b)
				} else {
					// 否则尝试将字节转为目标类型
					str := string(b)
					// 使用 scanType 创建实例并解析
					ptr := reflect.New(scanType)
					if _, err := fmt.Sscan(str, ptr.Interface()); err == nil {
						row[col] = reflect.Indirect(ptr).Interface()
					} else {
						row[col] = str
					}
				}
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	if err := rows.Err(); err != nil {
		return QueryResult{}, err
	}

	return QueryResult{
		Columns: columns,
		Rows:    results,
	}, nil
}

// toJSObject 将 Go 结构体转换为 JS 对象
func toJSObject(vm *goja.Runtime, result QueryResult) interface{} {
	jsObj := vm.NewObject()

	jsObj.Set("columns", result.Columns)
	jsObj.Set("rowsAffected", result.RowsAffected)

	rowsArr := vm.NewArray()
	for i, row := range result.Rows {
		rowObj := vm.NewObject()
		for key, val := range row {
			if val == nil {
				rowObj.Set(key, nil)
			} else {
				rowObj.Set(key, val) // 已经是正确的类型
			}
		}
		rowsArr.Set(strconv.Itoa(i), rowObj)
	}
	jsObj.Set("rows", rowsArr)

	return jsObj
}

// execResultToJSObject 将 ExecResult 转换为 JS 对象
func execResultToJSObject(vm *goja.Runtime, result ExecResult) interface{} {
	obj := vm.NewObject()
	obj.Set("rowsAffected", result.RowsAffected)
	obj.Set("lastInsertId", result.LastInsertId)
	return obj
}

// exec 执行写操作
func exec(db *sql.DB, sqlStr string, args ...interface{}) (ExecResult, error) {
	// 展开 []interface{} 为单个参数
	if len(args) == 1 {
		if arr, ok := args[0].([]interface{}); ok {
			args = arr
		}
	}

	result, err := db.Exec(sqlStr, args...)
	if err != nil {
		return ExecResult{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return ExecResult{}, err
	}

	lastInsertId, err := result.LastInsertId()
	if err != nil {
		return ExecResult{}, err
	}

	return ExecResult{
		RowsAffected: rowsAffected,
		LastInsertId: lastInsertId,
	}, nil
}

// txExec 在事务中执行 SQL
func txExec(tx *sql.Tx, sqlStr string, args ...interface{}) (ExecResult, error) {
	// 展开 []interface{} 为单个参数
	if len(args) == 1 {
		if arr, ok := args[0].([]interface{}); ok {
			args = arr
		}
	}

	result, err := tx.Exec(sqlStr, args...)
	if err != nil {
		return ExecResult{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return ExecResult{}, err
	}

	lastInsertId, err := result.LastInsertId()
	if err != nil {
		return ExecResult{}, err
	}

	return ExecResult{
		RowsAffected: rowsAffected,
		LastInsertId: lastInsertId,
	}, nil
}

// Open 根据配置打开数据库连接
// 1. 校验 DriverName 是否支持
// 2. 使用 DriverName + DSN 调用 sql.Open
// 3. 通过 Ping 验证连接是否可用
// 4. 返回 *sql.DB 或错误
func Open(config Config) (*sql.DB, error) {
	if _, ok := supportedDrivers[config.DriverName]; !ok {
		return nil, ErrUnsupportedDriver
	}

	db, err := sql.Open(config.DriverName, config.DSN)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// Module 数据库模块，实现 ActionModule 接口
type Module struct {
	connections map[*sql.DB]bool
}

// NewModule 创建 Module
func NewModule() *Module {
	return &Module{
		connections: make(map[*sql.DB]bool),
	}
}

// Register 将 sql 模块注册到 VM 全局
func (m *Module) Register(vm *goja.Runtime) {
	sqlObj := vm.NewObject()

	// 注册 open 方法，返回 JS 代理对象
	sqlObj.Set("open", func(config map[string]interface{}) interface{} {
		driverNameVal, ok := config["driverName"]
		if !ok {
			panic(errors.New("config missing 'driverName' field"))
		}
		driverName, ok := driverNameVal.(string)
		if !ok || driverName == "" {
			panic(errors.New("config 'driverName' must be non-empty string"))
		}

		dsnVal, ok := config["dsn"]
		if !ok {
			panic(errors.New("config missing 'dsn' field"))
		}
		dsn, ok := dsnVal.(string)
		if !ok || dsn == "" {
			panic(errors.New("config 'dsn' must be non-empty string"))
		}

		cfg := Config{
			DriverName: driverName,
			DSN:        dsn,
		}

		sqlDB, err := Open(cfg)
		if err != nil {
			panic(err)
		}
		m.connections[sqlDB] = true

		// 创建 DB 代理对象
		dbObj := vm.NewObject()
		dbObj.Set("query", func(sql string, args ...interface{}) interface{} {
			result, err := query(sqlDB, sql, args...)
			if err != nil {
				panic(err)
			}
			return toJSObject(vm, result)
		})
		dbObj.Set("exec", func(sql string, args ...interface{}) interface{} {
			result, err := exec(sqlDB, sql, args...)
			if err != nil {
				panic(err)
			}
			return execResultToJSObject(vm, result)
		})
		dbObj.Set("begin", func() interface{} {
			tx, err := sqlDB.Begin()
			if err != nil {
				panic(err)
			}
			// 创建 Tx 代理对象
			txObj := vm.NewObject()
			txObj.Set("exec", func(sql string, args ...interface{}) interface{} {
				result, err := txExec(tx, sql, args...)
				if err != nil {
					panic(err)
				}
				return execResultToJSObject(vm, result)
			})
			txObj.Set("commit", func() {
				tx.Commit()
			})
			txObj.Set("rollback", func() {
				tx.Rollback()
			})
			txObj.Set("close", func() {
				tx.Rollback()
			})
			return txObj
		})
		dbObj.Set("close", func() {
			sqlDB.Close()
		})

		return dbObj
	})

	vm.Set("sql", sqlObj)
}

// Destroy 销毁所有连接
func (m *Module) Destroy() {
	for db := range m.connections {
		db.Close()
	}
	m.connections = make(map[*sql.DB]bool)
}
