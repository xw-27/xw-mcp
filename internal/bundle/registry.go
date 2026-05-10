package bundle

import "sync"

// ActionRegistry 动作注册表，管理Bundle内的所有Action
type ActionRegistry struct {
	mu        sync.RWMutex                  // 读写锁
	tools     map[string]*ToolAction        // 工具集合
	prompts   map[string]*PromptAction       // 提示词集合
	resources map[string]*ResourceAction     // 资源集合
	templates map[string]*ResourceTemplateAction  // 资源模板集合
}

// NewActionRegistry 创建新的动作注册表
func NewActionRegistry() *ActionRegistry {
	return &ActionRegistry{
		tools:     make(map[string]*ToolAction),
		prompts:   make(map[string]*PromptAction),
		resources: make(map[string]*ResourceAction),
		templates: make(map[string]*ResourceTemplateAction),
	}
}

// RegisterTool 注册工具
func (r *ActionRegistry) RegisterTool(tool *ToolAction) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name] = tool
}

// RegisterPrompt 注册提示词
func (r *ActionRegistry) RegisterPrompt(prompt *PromptAction) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.prompts[prompt.Name] = prompt
}

// RegisterResource 注册资源
func (r *ActionRegistry) RegisterResource(resource *ResourceAction) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.resources[resource.Name] = resource
}

// RegisterTemplate 注册资源模板
func (r *ActionRegistry) RegisterTemplate(template *ResourceTemplateAction) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.templates[template.Name] = template
}

// GetTool 获取工具
func (r *ActionRegistry) GetTool(name string) (*ToolAction, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, ok := r.tools[name]
	return tool, ok
}

// GetPrompt 获取提示词
func (r *ActionRegistry) GetPrompt(name string) (*PromptAction, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	prompt, ok := r.prompts[name]
	return prompt, ok
}

// GetResource 获取资源
func (r *ActionRegistry) GetResource(name string) (*ResourceAction, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	resource, ok := r.resources[name]
	return resource, ok
}

// GetTemplate 获取资源模板
func (r *ActionRegistry) GetTemplate(name string) (*ResourceTemplateAction, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	template, ok := r.templates[name]
	return template, ok
}

// ListTools 列出所有工具
func (r *ActionRegistry) ListTools() []*ToolAction {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tools := make([]*ToolAction, 0, len(r.tools))
	for _, tool := range r.tools {
		tools = append(tools, tool)
	}
	return tools
}

// ListPrompts 列出所有提示词
func (r *ActionRegistry) ListPrompts() []*PromptAction {
	r.mu.RLock()
	defer r.mu.RUnlock()
	prompts := make([]*PromptAction, 0, len(r.prompts))
	for _, prompt := range r.prompts {
		prompts = append(prompts, prompt)
	}
	return prompts
}

// ListResources 列出所有资源
func (r *ActionRegistry) ListResources() []*ResourceAction {
	r.mu.RLock()
	defer r.mu.RUnlock()
	resources := make([]*ResourceAction, 0, len(r.resources))
	for _, resource := range r.resources {
		resources = append(resources, resource)
	}
	return resources
}

// ListTemplates 列出所有资源模板
func (r *ActionRegistry) ListTemplates() []*ResourceTemplateAction {
	r.mu.RLock()
	defer r.mu.RUnlock()
	templates := make([]*ResourceTemplateAction, 0, len(r.templates))
	for _, template := range r.templates {
		templates = append(templates, template)
	}
	return templates
}

// RemoveTool 移除工具
func (r *ActionRegistry) RemoveTool(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tools, name)
}

// RemovePrompt 移除提示词
func (r *ActionRegistry) RemovePrompt(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.prompts, name)
}

// RemoveResource 移除资源
func (r *ActionRegistry) RemoveResource(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.resources, name)
}

// RemoveTemplate 移除资源模板
func (r *ActionRegistry) RemoveTemplate(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.templates, name)
}

// Clear 清空所有注册表
func (r *ActionRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools = make(map[string]*ToolAction)
	r.prompts = make(map[string]*PromptAction)
	r.resources = make(map[string]*ResourceAction)
	r.templates = make(map[string]*ResourceTemplateAction)
}