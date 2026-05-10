package bundle

// ActionType 常量定义
const (
	ActionTypeTool              = "tool"               // 工具类型
	ActionTypePrompt            = "prompt"             // 提示词类型
	ActionTypeResource          = "resource"           // 资源类型
	ActionTypeResourceTemplate  = "resource-template"  // 资源模板类型
)

// PromptArgument 提示词参数定义
type PromptArgument struct {
	Name        string  // 参数名称
	Title       string  // 显示标题
	Description string  // 参数描述
	Required    bool    // 是否必需
}

// PromptMessage 提示词消息结构
type PromptMessage struct {
	Role    string  // 角色: user/assistant/system
	Content string  // 消息内容（支持模板变量 {{变量名}}）
}

// ResourceContents 资源内容结构
type ResourceContents struct {
	URI      string  // 资源URI
	MIMEType string  // MIME类型
	Text     string  // 文本内容
	Blob     []byte  // 二进制内容（base64编码）
}

// BaseAction 所有Action的基类
type BaseAction struct {
	Type        string                           // Action类型常量
	Name        string                           // 动作名称（唯一标识）
	Description string                          // 动作描述
	Execute     func(params interface{}) (interface{}, error)  // 执行函数
}

// ToolAction 工具动作
type ToolAction struct {
	BaseAction  // 嵌入基类（含 Execute）
	InputSchema map[string]interface{}  // 输入参数Schema
}

// PromptAction 提示词动作
type PromptAction struct {
	BaseAction  // 嵌入基类（含 Execute）
	Arguments   []*PromptArgument  // 参数列表
	Messages    []*PromptMessage  // 消息模板列表
}

// ResourceAction 资源动作
type ResourceAction struct {
	BaseAction  // 嵌入基类（含 Execute）
	URI         string  // 资源URI
	MIMEType    string  // MIME类型
}

// ResourceTemplateAction 资源模板动作
type ResourceTemplateAction struct {
	BaseAction   // 嵌入基类（含 Execute）
	URITemplate  string  // URI模板（如 file://docs/{docId}）
	MIMEType     string  // MIME类型
}

// GetType 获取Action类型
func (a *BaseAction) GetType() string {
	return a.Type
}

// GetName 获取Action名称
func (a *BaseAction) GetName() string {
	return a.Name
}

// GetDescription 获取Action描述
func (a *BaseAction) GetDescription() string {
	return a.Description
}