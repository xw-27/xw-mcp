package bundle

import (
	"sync"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

// LoadState Bundle加载状态
type LoadState int

// LoadState 常量定义
const (
	LoadStateNone LoadState = iota  // 未加载
	LoadStateMeta                    // 半加载（只解析元数据）
	LoadStateFull                    // 全加载（完整加载）
	LoadStateError                   // 加载失败
)

// Bundle 插件项目单元，每个Bundle拥有独立的Runtime和ActionRegistry
type Bundle struct {
	name           string          // Bundle名称（目录名）
	path           string          // Bundle路径
	indexFile      string          // 入口文件名
	runtime        *goja.Runtime   // JavaScript运行时（每个Bundle独立）
	registry       *ActionRegistry // 动作注册表
	enabled        bool            // 是否启用
	loadState      LoadState       // 当前加载状态
	loadError      error           // 加载错误信息
	mu             sync.RWMutex    // 读写锁（保护 loadState/loadError）
	loadMu         sync.Mutex      // 加载互斥锁（防止 Load/Unload 并发）
	requireRegistry *require.Registry // 模块注册表（用于重置Runtime）
}

// NewBundle 创建新的Bundle实例
func NewBundle(name, path string) *Bundle {
	return &Bundle{
		name:      name,
		path:      path,
		indexFile: "index.js",
		registry:  NewActionRegistry(),
		enabled:   true,
		loadState: LoadStateNone,
	}
}

// Name 获取Bundle名称
func (b *Bundle) Name() string {
	return b.name
}

// Path 获取Bundle路径
func (b *Bundle) Path() string {
	return b.path
}

// IndexFile 获取入口文件名
func (b *Bundle) IndexFile() string {
	return b.indexFile
}

// Registry 获取动作注册表
func (b *Bundle) Registry() *ActionRegistry {
	return b.registry
}

// Enabled 获取启用状态
func (b *Bundle) Enabled() bool {
	return b.enabled
}

// SetEnabled 设置启用状态
func (b *Bundle) SetEnabled(enabled bool) {
	b.enabled = enabled
}

// SetIndexFile 设置入口文件名
func (b *Bundle) SetIndexFile(filename string) {
	b.indexFile = filename
}

// initRuntime 初始化JavaScript运行时
func (b *Bundle) initRuntime() error {
	vm := goja.New()
	reg := require.NewRegistry()
	reg.Enable(vm)

	b.runtime = vm
	b.requireRegistry = reg
	return nil
}

// GetRuntime 获取JavaScript运行时，不存在则创建
func (b *Bundle) GetRuntime() *goja.Runtime {
	if b.runtime == nil {
		b.initRuntime()
	}
	return b.runtime
}

// ResetRuntime 重置Runtime（重新创建干净的Runtime，避免模块缓存累积）
func (b *Bundle) ResetRuntime() {
	b.runtime = nil
	b.requireRegistry = nil
	b.initRuntime()
}

// LoadState 获取当前加载状态
func (b *Bundle) LoadState() LoadState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.loadState
}

// SetLoadState 设置加载状态
func (b *Bundle) SetLoadState(state LoadState) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.loadState = state
}

// LoadError 获取加载错误信息
func (b *Bundle) LoadError() error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.loadError
}

// SetLoadError 设置加载错误信息
func (b *Bundle) SetLoadError(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.loadError = err
}

// LockLoad 获取加载互斥锁
func (b *Bundle) LockLoad() {
	b.loadMu.Lock()
}

// UnlockLoad 释放加载互斥锁
func (b *Bundle) UnlockLoad() {
	b.loadMu.Unlock()
}

// Close 关闭Bundle，释放资源
func (b *Bundle) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.runtime != nil {
		b.runtime = nil
	}
	b.registry.Clear()
	b.loadState = LoadStateNone
	b.loadError = nil
	return nil
}

// RegisterTool 注册工具到当前Bundle
func (b *Bundle) RegisterTool(tool *ToolAction) {
	b.registry.RegisterTool(tool)
}

// RegisterPrompt 注册提示词到当前Bundle
func (b *Bundle) RegisterPrompt(prompt *PromptAction) {
	b.registry.RegisterPrompt(prompt)
}

// RegisterResource 注册资源到当前Bundle
func (b *Bundle) RegisterResource(resource *ResourceAction) {
	b.registry.RegisterResource(resource)
}

// RegisterTemplate 注册资源模板到当前Bundle
func (b *Bundle) RegisterTemplate(template *ResourceTemplateAction) {
	b.registry.RegisterTemplate(template)
}

// GetTool 获取工具
func (b *Bundle) GetTool(name string) (*ToolAction, bool) {
	return b.registry.GetTool(name)
}

// GetPrompt 获取提示词
func (b *Bundle) GetPrompt(name string) (*PromptAction, bool) {
	return b.registry.GetPrompt(name)
}

// GetResource 获取资源
func (b *Bundle) GetResource(name string) (*ResourceAction, bool) {
	return b.registry.GetResource(name)
}

// GetTemplate 获取资源模板
func (b *Bundle) GetTemplate(name string) (*ResourceTemplateAction, bool) {
	return b.registry.GetTemplate(name)
}