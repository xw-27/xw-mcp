package bundle

import (
	"log"
	"sync"
)

// BundleContext Bundle全局上下文，管理所有Bundle实例
// 组合BundleWatcher和BundleLoader，实现插件的扫描、监控、加载、卸载
type BundleContext struct {
	*BundleWatcher  // 嵌入Watcher（监控文件变化）
	*BundleLoader    // 嵌入Loader（加载Bundle）
	mu      sync.RWMutex
	bundles map[string]*Bundle         // Bundle集合（name -> Bundle）
}

// NewBundleContext 创建新的BundleContext实例
func NewBundleContext() *BundleContext {
	c := &BundleContext{
		BundleWatcher: NewBundleWatcher(),
		BundleLoader: NewBundleLoader(),
		bundles: make(map[string]*Bundle),
	}

	c.initCallbacks()
	return c
}

// initCallbacks 初始化内部回调
// 处理Watcher触发的load/unload事件，调用Loader完成加载/卸载逻辑
func (c *BundleContext) initCallbacks() {
	c.AddCallback(func(event BundleWatchEvent) {
		switch event.EventType {
		case BundleEventLoad:
			// load事件：Bundle新增或更新
			bundle, exists := c.Get(event.BundleName)
			if !exists {
				// 新增Bundle：创建并注册到Context
				bundle = NewBundle(event.BundleName, event.BundlePath)
				c.Register(bundle)
			}
			// 调用Loader半加载（只解析元数据）
			if err := c.BundleLoader.MetaLoad(bundle); err != nil {
				log.Printf("[bundle] load failed: %s, path: %s, error: %v", event.BundleName, event.BundlePath, err)
				return
			}
			// 输出加载日志
			c.logBundleInfo(bundle)

		case BundleEventUnload:
			// unload事件：Bundle被删除
			bundle, exists := c.Get(event.BundleName)
			if exists {
				log.Printf("[bundle] unloading: %s, path: %s", event.BundleName, event.BundlePath)
				// 调用Loader卸载Bundle
				c.BundleLoader.Unload(bundle)
				// 从Context注销Bundle
				c.Unregister(event.BundleName)
				log.Printf("[bundle] unloaded: %s", event.BundleName)
			}
		}
	})
}

// logBundleInfo 打印Bundle详细信息
func (c *BundleContext) logBundleInfo(b *Bundle) {
	registry := b.Registry()

	log.Printf("[bundle] loaded: %s, path: %s", b.Name(), b.Path())

	// 打印工具列表
	tools := registry.ListTools()
	if len(tools) > 0 {
		log.Printf("[bundle]   tools (%d):", len(tools))
		for _, t := range tools {
			log.Printf("[bundle]     - %s: %s", t.Name, t.Description)
		}
	}

	// 打印提示词列表
	prompts := registry.ListPrompts()
	if len(prompts) > 0 {
		log.Printf("[bundle]   prompts (%d):", len(prompts))
		for _, p := range prompts {
			log.Printf("[bundle]     - %s: %s", p.Name, p.Description)
		}
	}

	// 打印资源列表
	resources := registry.ListResources()
	if len(resources) > 0 {
		log.Printf("[bundle]   resources (%d):", len(resources))
		for _, r := range resources {
			log.Printf("[bundle]     - %s: %s", r.Name, r.Description)
		}
	}

	// 打印资源模板列表
	templates := registry.ListTemplates()
	if len(templates) > 0 {
		log.Printf("[bundle]   templates (%d):", len(templates))
		for _, t := range templates {
			log.Printf("[bundle]     - %s: %s", t.Name, t.Description)
		}
	}
}

// StartWatcher 启动文件监控
// bundlesPath: 监控目录路径
// depth: 监控深度（层数）
func (c *BundleContext) StartWatcher(bundlesPath string, depth int) error {
	return c.Watch(bundlesPath, depth)
}

// Register 注册Bundle到上下文
func (c *BundleContext) Register(b *Bundle) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.bundles[b.Name()]; exists {
		return ErrBundleAlreadyExists
	}

	c.bundles[b.Name()] = b
	log.Printf("[bundle] registered: %s, path: %s", b.Name(), b.Path())
	return nil
}

// Unregister 从上下文取消注册Bundle
func (c *BundleContext) Unregister(name string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	bundle, exists := c.bundles[name]
	if !exists {
		return ErrBundleNotFound
	}

	bundle.Close()
	delete(c.bundles, name)
	log.Printf("[bundle] unregistered: %s", name)
	return nil
}

// Get 根据名称获取Bundle
func (c *BundleContext) Get(name string) (*Bundle, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	bundle, ok := c.bundles[name]
	return bundle, ok
}

// List 获取所有Bundle列表
func (c *BundleContext) List() []*Bundle {
	c.mu.RLock()
	defer c.mu.RUnlock()

	bundles := make([]*Bundle, 0, len(c.bundles))
	for _, bundle := range c.bundles {
		bundles = append(bundles, bundle)
	}
	return bundles
}

// GetAction 根据Bundle名称、Action名称和类型获取Action
func (c *BundleContext) GetAction(bundleName, actionName string, actionType string) (interface{}, bool) {
	bundle, ok := c.Get(bundleName)
	if !ok {
		return nil, false
	}

	switch actionType {
	case ActionTypeTool:
		return bundle.GetTool(actionName)
	case ActionTypePrompt:
		return bundle.GetPrompt(actionName)
	case ActionTypeResource:
		return bundle.GetResource(actionName)
	case ActionTypeResourceTemplate:
		return bundle.GetTemplate(actionName)
	default:
		return nil, false
	}
}

// Close 关闭Context，释放所有Bundle资源
func (c *BundleContext) Close() error {
	// 关闭Watcher（停止文件监控）
	c.BundleWatcher.Close()

	c.mu.Lock()
	defer c.mu.Unlock()

	// 关闭所有Bundle
	for name, bundle := range c.bundles {
		bundle.Close()
		delete(c.bundles, name)
	}

	log.Printf("[bundle] context closed")
	return nil
}

// BundleCount 获取当前Bundle数量
func (c *BundleContext) BundleCount() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.bundles)
}
