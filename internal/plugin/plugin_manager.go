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
)

// PluginManager 是插件管理器
type PluginManager struct {
	dir      string
	maxDepth int
	plugins  map[string]*Plugin
	paths    map[string]string
	runtime  *goja.Runtime
	registry *require.Registry
	watcher  chan notify.EventInfo
	mu       sync.RWMutex
}

// New 创建 PluginManager 实例
// dir: 插件目录路径
// maxDepth: 最大扫描深度
func New(dir string, maxDepth int) (*PluginManager, error) {
	pm := &PluginManager{
		dir:      dir,
		maxDepth: maxDepth,
		plugins:  make(map[string]*Plugin),
		paths:    make(map[string]string),
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
	files, err := pm.scanDir(pm.dir, 0)
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
		return fmt.Errorf("read file failed: %w", err)
	}

	// 执行 JS 代码获取 module.exports
	vm := goja.New()

	// 启用 require
	registry := new(require.Registry)
	registry.Enable(vm)

	// 设置全局 module 和 exports
	module := vm.NewObject()
	exports := vm.NewObject()
	module.Set("exports", exports)
	vm.Set("module", module)
	vm.Set("exports", exports)

	// 执行 JS 代码
	_, err = vm.RunString(string(content))
	if err != nil {
		return fmt.Errorf("execute script failed: %w", err)
	}

	// 获取 module.exports
	result := module.Get("exports")

	// 解析结果，可能是对象或数组
	if result != nil && !goja.IsUndefined(result) && !goja.IsNull(result) {
		obj := result.ToObject(vm)
		if obj.ClassName() == "Array" {
			// 数组形式：module.exports = [{ name: ..., execute: ... }, ...]
			arrLen := obj.Get("length").ToInteger()
			for i := int64(0); i < arrLen; i++ {
				item := obj.Get(fmt.Sprintf("%d", i))
				if plugin := pm.parsePlugin(item, filePath, vm); plugin != nil {
					pm.registerPlugin(plugin)
				}
			}
		} else {
			// 对象形式：module.exports = { name: ..., execute: ... }
			if plugin := pm.parsePlugin(result, filePath, vm); plugin != nil {
				pm.registerPlugin(plugin)
			}
		}
	}

	return nil
}

// parsePlugin 解析 goja.Value 为 Plugin
func (pm *PluginManager) parsePlugin(value goja.Value, filePath string, vm *goja.Runtime) *Plugin {
	obj := value.ToObject(vm)

	name := obj.Get("name").ToString().String()
	if name == "" {
		log.Printf("[plugin] plugin name is empty in %s", filePath)
		return nil
	}

	description := obj.Get("description").ToString().String()

	var inputSchema map[string]interface{}
	if v := obj.Get("inputSchema"); v != nil && !goja.IsUndefined(v) && !goja.IsNull(v) {
		if m, ok := v.ToObject(vm).Export().(map[string]interface{}); ok {
			inputSchema = m
		}
	}

	// 获取 execute 函数
	var executeFn func(params interface{}) (interface{}, error)
	if v := obj.Get("execute"); !goja.IsUndefined(v) && !goja.IsNull(v) {
		if fn, ok := goja.AssertFunction(v); ok {
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

	// 递归监控，使用 "..." 语法
	watchPath := fmt.Sprintf("%s/...", pm.dir)
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

	// 移除旧插件
	for name, path := range pm.paths {
		if path == filePath {
			delete(pm.plugins, name)
			delete(pm.paths, name)
		}
	}

	pm.mu.Unlock()

	// 重新加载
	return pm.loadFile(filePath)
}

// removePlugin 移除插件
func (pm *PluginManager) removePlugin(filePath string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for name, path := range pm.paths {
		if path == filePath {
			delete(pm.plugins, name)
			delete(pm.paths, name)
			log.Printf("[plugin] removed: %s", name)
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

	return nil
}