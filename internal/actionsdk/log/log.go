package log

import (
	"log"

	"github.com/dop251/goja"
)

// LogModule 日志模块，实现 ActionModule 接口
type LogModule struct {
	callCount int
}

func (m *LogModule) Name() string {
	return "log"
}

// Register 将日志方法注册到VM全局
func (m *LogModule) Register(vm *goja.Runtime) {
	logObj := vm.NewObject()
	logObj.Set("debug", func(msg string) { m.log(msg) })
	logObj.Set("info", func(msg string) { m.log(msg) })
	logObj.Set("warn", func(msg string) { m.log(msg) })
	logObj.Set("error", func(msg string) { m.log(msg) })
	vm.Set("log", logObj)
}

func (m *LogModule) log(msg string) {
	m.callCount++
	log.Print(msg)
}

// Destroy 销毁方法，打印最终调用计数
func (m *LogModule) Destroy() {
	log.Printf("log module destroyed, callCount: %d", m.callCount)
}