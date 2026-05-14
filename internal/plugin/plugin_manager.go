package plugin

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
	"github.com/rjeczalik/notify"

	"xw-mcp/internal/config"
)

// PluginManager 是插件管理器
type PluginManager struct {
	cfg        *config.ConfigWatcher
	maxDepth   int
	plugins    map[string]*Plugin
	paths      map[string]string
	runtime    *goja.Runtime
	registry   *require.Registry
	watcher    chan notify.EventInfo
	mu         sync.RWMutex
	events     []eventEntry
	eventsMu   sync.RWMutex
	nextID     uintptr
}

// New 创建 PluginManager 实例
// cfg: ConfigWatcher 实例，用于获取插件目录等配置
// maxDepth: 最大扫描深度
// callbacks: 可选的回调函数列表
func New(cfg *config.ConfigWatcher, maxDepth int, callbacks ...EventCallback) (*PluginManager, error) {
	_, ok := cfg.Get("server.plugins-dir")
	if !ok {
		return nil, fmt.Errorf("plugins-dir not found in config")
	}

	pm := &PluginManager{
		cfg:      cfg,
		maxDepth: maxDepth,
		plugins:  make(map[string]*Plugin),
		paths:    make(map[string]string),
	}

	// 注册回调
	for _, callback := range callbacks {
		pm.events = append(pm.events, eventEntry{
			callback: callback,
			id:       pm.nextID,
		})
		pm.nextID++
	}

	if err := pm.initRuntime(); err != nil {
		return nil, fmt.Errorf("init runtime failed: %w", err)
	}

	if err := pm.scanAndLoad(); err != nil {
		return nil, fmt.Errorf("scan and load failed: %w", err)
	}

	if err := pm.watch(); err != nil {
		log.Printf("[plugin] watch failed: %v", err)
	}

	return pm, nil
}

// initRuntime 初始化 goja Runtime 和 require
func (pm *PluginManager) initRuntime() error {
	pm.runtime = goja.New()
	pm.registry = require.NewRegistry()
	pm.registry.Enable(pm.runtime)
	return nil
}

// scanAndLoad 扫描目录并加载所有插件
func (pm *PluginManager) scanAndLoad() error {
	pluginsDir, _ := pm.cfg.Get("server.plugins-dir")
	files, err := pm.scanDir(pluginsDir.ToString(), 0)
	if err != nil {
		return fmt.Errorf("scan dir failed: %w", err)
	}

	for _, file := range files {
		if err := pm.loadFile(file); err != nil {
			log.Printf("[plugin] load file %s failed: %v", file, err)
			continue
		}
	}

	return nil
}

// scanDir 递归扫描目录，限制深度
// path: 当前路径
// depth: 当前深度
// 返回: .js 文件路径列表
func (pm *PluginManager) scanDir(path string, depth int) ([]string, error) {
	if depth > pm.maxDepth {
		return nil, nil
	}

	var files []string

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read dir %s failed: %w", path, err)
	}

	for _, entry := range entries {
		fullPath := filepath.Join(path, entry.Name())

		if entry.IsDir() {
			subFiles, err := pm.scanDir(fullPath, depth+1)
			if err != nil {
				log.Printf("[plugin] scan subdir %s failed: %v", fullPath, err)
				continue
			}
			files = append(files, subFiles...)
		} else if strings.HasSuffix(entry.Name(), ".js") {
			files = append(files, fullPath)
		}
	}

	return files, nil
}

// loadFile 加载单个 JS 文件
func (pm *PluginManager) loadFile(filePath string) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("[plugin] load file %s failed: %v", filePath, err)
		pm.removePlugin(filePath)
		return nil
	}

	vm := goja.New()
	registry := new(require.Registry)
	registry.Enable(vm)

	module := vm.NewObject()
	exports := vm.NewObject()
	module.Set("exports", exports)
	vm.Set("module", module)
	vm.Set("exports", exports)

	_, err = vm.RunString(string(content))
	if err != nil {
		log.Printf("[plugin] execute script %s failed: %v", filePath, err)
		pm.removePlugin(filePath)
		return nil
	}

	result := module.Get("exports")

	if result == nil || goja.IsUndefined(result) || goja.IsNull(result) {
		log.Printf("[plugin] script %s has no exports, treating as empty", filePath)
		pm.removePlugin(filePath)
		return nil
	}

	obj := result.ToObject(vm)
	if obj == nil {
		log.Printf("[plugin] exports object is nil in %s", filePath)
		pm.removePlugin(filePath)
		return nil
	}

	if obj.ClassName() == "Array" {
		arrLen := obj.Get("length").ToInteger()
		for i := int64(0); i < arrLen; i++ {
			item := obj.Get(fmt.Sprintf("%d", i))
			if plugin := pm.parsePlugin(item, filePath, vm); plugin != nil {
				pm.registerPlugin(plugin)
			}
		}
	} else {
		if plugin := pm.parsePlugin(result, filePath, vm); plugin != nil {
			pm.registerPlugin(plugin)
		}
	}

	return nil
}

// parsePlugin 解析 goja.Value 为 Plugin
func (pm *PluginManager) parsePlugin(value goja.Value, filePath string, vm *goja.Runtime) *Plugin {
	if goja.IsUndefined(value) || goja.IsNull(value) {
		log.Printf("[plugin] value is undefined/null in %s", filePath)
		return nil
	}

	obj := value.ToObject(vm)
	if obj == nil {
		log.Printf("[plugin] object is nil in %s", filePath)
		return nil
	}

	nameValue := obj.Get("name")
	if nameValue == nil || goja.IsUndefined(nameValue) || goja.IsNull(nameValue) {
		log.Printf("[plugin] name is undefined/null in %s", filePath)
		return nil
	}
	name := nameValue.ToString().String()
	if name == "" {
		log.Printf("[plugin] name is empty in %s", filePath)
		return nil
	}

	descriptionValue := obj.Get("description")
	var description string
	if descriptionValue != nil && !goja.IsUndefined(descriptionValue) && !goja.IsNull(descriptionValue) {
		description = descriptionValue.ToString().String()
	}

	var inputSchema map[string]interface{}
	inputSchemaValue := obj.Get("inputSchema")
	if inputSchemaValue != nil && !goja.IsUndefined(inputSchemaValue) && !goja.IsNull(inputSchemaValue) {
		if inputObj := inputSchemaValue.ToObject(vm); inputObj != nil {
			if m, ok := inputObj.Export().(map[string]interface{}); ok {
				inputSchema = m
			}
		}
	}

	var executeFn func(params interface{}) (interface{}, error)
	executeValue := obj.Get("execute")
	if executeValue != nil && !goja.IsUndefined(executeValue) && !goja.IsNull(executeValue) {
		if fn, ok := goja.AssertFunction(executeValue); ok {
			executeFn = func(params interface{}) (interface{}, error) {
				result, err := fn(goja.Undefined(), vm.ToValue(params))
				if err != nil {
					return nil, err
				}
				return result.Export(), nil
			}
		}
	}

	return &Plugin{
		Name:        name,
		Description: description,
		InputSchema: inputSchema,
		FilePath:    filePath,
		execute:     executeFn,
	}
}

// registerPlugin 注册插件到管理器
func (pm *PluginManager) registerPlugin(plugin *Plugin) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.plugins[plugin.Name] = plugin
	pm.paths[plugin.Name] = plugin.FilePath

	log.Printf("[plugin] registered: %s from %s", plugin.Name, plugin.FilePath)

	pm.emit(PluginEventData{
		Event:  EventPluginAdd,
		Plugin: plugin,
	})
}

// Plugins 返回所有插件列表
func (pm *PluginManager) Plugins() []*Plugin {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := make([]*Plugin, 0, len(pm.plugins))
	for _, plugin := range pm.plugins {
		result = append(result, plugin)
	}

	return result
}

// Get 根据名称获取插件
func (pm *PluginManager) Get(name string) (*Plugin, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, ok := pm.plugins[name]
	return plugin, ok
}

// AddEventHandler 注册事件回调，返回移除用的 ID
func (pm *PluginManager) AddEventHandler(callback EventCallback) uintptr {
	pm.eventsMu.Lock()
	defer pm.eventsMu.Unlock()

	id := pm.nextID
	pm.nextID++

	pm.events = append(pm.events, eventEntry{
		callback: callback,
		id:       id,
	})

	return id
}

// RemoveEventHandler 移除事件回调
func (pm *PluginManager) RemoveEventHandler(id uintptr) {
	pm.eventsMu.Lock()
	defer pm.eventsMu.Unlock()

	for i, entry := range pm.events {
		if entry.id == id {
			pm.events = append(pm.events[:i], pm.events[i+1:]...)
			return
		}
	}
}

// emit 触发事件（内部使用）
func (pm *PluginManager) emit(data PluginEventData) {
	pm.eventsMu.RLock()
	defer pm.eventsMu.RUnlock()
	for _, entry := range pm.events {
		entry.callback(data)
	}
}

// Call 根据名称调用插件
func (pm *PluginManager) Call(name string, params interface{}) (interface{}, error) {
	plugin, ok := pm.Get(name)
	if !ok {
		return nil, fmt.Errorf("plugin not found: %s", name)
	}

	return plugin.Execute(params)
}

// watch 启动文件监控
func (pm *PluginManager) watch() error {
	pm.watcher = make(chan notify.EventInfo, 100)

	pluginsDir, _ := pm.cfg.Get("server.plugins-dir")
	watchPath := fmt.Sprintf("%s/...", pluginsDir.ToString())
	if err := notify.Watch(watchPath, pm.watcher, notify.All); err != nil {
		return fmt.Errorf("notify watch failed: %w", err)
	}

	go pm.watchLoop()

	return nil
}

// watchLoop 文件监控循环
func (pm *PluginManager) watchLoop() {
	for {
		select {
		case event, ok := <-pm.watcher:
			if !ok {
				return
			}

			// 只处理 .js 文件
			if !strings.HasSuffix(event.Path(), ".js") {
				continue
			}

			// 处理文件变更
			if event.Event() == notify.Write || event.Event() == notify.Create {
				log.Printf("[plugin] file changed: %s", event.Path())
				if err := pm.reloadFile(event.Path()); err != nil {
					log.Printf("[plugin] reload failed: %v", err)
				}
			} else if event.Event() == notify.Remove {
				log.Printf("[plugin] file removed: %s", event.Path())
				pm.removePlugin(event.Path())
			}
		}
	}
}

// reloadFile 重新加载单个文件
func (pm *PluginManager) reloadFile(filePath string) error {
	pm.mu.Lock()

	// 移除旧插件，触发删除事件
	for name, path := range pm.paths {
		if path == filePath {
			oldPlugin := pm.plugins[name]
			delete(pm.plugins, name)
			delete(pm.paths, name)

			pm.emit(PluginEventData{
				Event:     EventPluginDelete,
				OldPlugin: oldPlugin,
			})
		}
	}

	pm.mu.Unlock()

	// 重新加载
	if err := pm.loadFile(filePath); err != nil {
		return err
	}

	return nil
}

// removePlugin 移除插件
func (pm *PluginManager) removePlugin(filePath string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for name, path := range pm.paths {
		if path == filePath {
			oldPlugin := pm.plugins[name]
			delete(pm.plugins, name)
			delete(pm.paths, name)
			log.Printf("[plugin] removed: %s", name)

			pm.emit(PluginEventData{
				Event:     EventPluginDelete,
				OldPlugin: oldPlugin,
			})
		}
	}
}

// Close 关闭插件管理器
func (pm *PluginManager) Close() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.watcher != nil {
		notify.Stop(pm.watcher)
		pm.watcher = nil
	}

	pm.plugins = nil
	pm.paths = nil

	pm.eventsMu.Lock()
	pm.events = nil
	pm.eventsMu.Unlock()

	return nil
}