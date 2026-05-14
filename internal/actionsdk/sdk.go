package actionsdk

import (
	"sync"

	"github.com/dop251/goja"

	"xw-mcp/internal/actionsdk/conf"
	"xw-mcp/internal/actionsdk/log"
	"xw-mcp/internal/actionsdk/sql"
	configpkg "xw-mcp/internal/config"
)

// Destroyable 资源销毁接口
type Destroyable interface {
	Destroy()
}

// ActionModule SDK模块接口
type ActionModule interface {
	Destroyable
	Register(vm *goja.Runtime)
}

// LifecycleManager 生命周期管理器
type LifecycleManager struct {
	mu      sync.Mutex
	modules []ActionModule
}

// NewLifecycleManager 创建生命周期管理器，初始化模块列表
//
// cfg: 配置监听器，可为 nil
func NewLifecycleManager(cfg *configpkg.ConfigWatcher) *LifecycleManager {
	modules := []ActionModule{
		sql.NewModule(),
		&log.LogModule{},
	}
	if cfg != nil {
		modules = append(modules, conf.NewModule(cfg))
	}
	return &LifecycleManager{modules: modules}
}

// RegisterAllModules 注册所有模块到VM全局
func (lm *LifecycleManager) RegisterAllModules(vm *goja.Runtime) {
	for _, module := range lm.modules {
		module.Register(vm)
	}
}

// DestroyAll 销毁所有模块
func (lm *LifecycleManager) DestroyAll() {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	for _, module := range lm.modules {
		func() {
			defer func() {
				if r := recover(); r != nil {
					println("WARN: Destroy() panicked:", r)
				}
			}()
			module.Destroy()
		}()
	}
}
