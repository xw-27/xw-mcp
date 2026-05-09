package plugin

import "github.com/dop251/goja"

// Plugin 是 MCP 工具的数据结构
type Plugin struct {
    Name        string
    Description string
    InputSchema map[string]interface{}
    FilePath    string
    execute     func(params interface{}) (interface{}, error)
}

// Execute 执行插件的处理逻辑
func (p *Plugin) Execute(params interface{}) (interface{}, error) {
    if p.execute == nil {
        return nil, nil
    }
    return p.execute(params)
}

// SetExecute 设置执行函数
func (p *Plugin) SetExecute(fn func(params interface{}) (interface{}, error)) {
    p.execute = fn
}

// ToValue 将 Plugin 转换为 goja.Value
func (p *Plugin) ToValue(vm *goja.Runtime) goja.Value {
    obj := vm.NewObject()
    obj.Set("name", p.Name)
    obj.Set("description", p.Description)
    obj.Set("inputSchema", p.InputSchema)
    return obj
}