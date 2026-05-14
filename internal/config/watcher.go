// Package config 提供 YAML 配置文件的热重载加载功能。
//
// 示例:
//
//	cfg, err := config.NewWatcher("config.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer cfg.Close()
//
//	if port, ok := cfg.Get("server.port"); ok {
//	    fmt.Println(port.Int())
//	}
package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/rjeczalik/notify"
	"gopkg.in/yaml.v3"
)

// ConfigWatcher 配置监听器
//
// 监听 YAML 配置文件的变化并自动热重载。线程安全，支持并发读取。
//
// 字段说明:
//   - path: 配置文件路径
//   - data: 配置数据 map
//   - watcherChan: 文件系统事件通道
//   - mu: 读写锁，保护 data 字段
//   - wg: 等待 watchLoop goroutine 退出
//   - closed: 是否已关闭
//   - closeMu: 保护 closed 标志的互斥锁
//   - running: 是否运行中，用于控制 watchLoop 退出
//   - runningMu: 保护 running 标志的互斥锁
type ConfigWatcher struct {
	path        string
	data        map[string]interface{}
	watcherChan chan notify.EventInfo
	mu          sync.RWMutex
	wg          sync.WaitGroup
	closed      bool
	closeMu     sync.Mutex
	running     bool
	runningMu   sync.Mutex
}

// NewWatcher 创建 ConfigWatcher 实例
//
// path: YAML 配置文件路径。立即加载并开始监听文件变化。
// 加载失败时返回空配置监听器，不会导致任务失败。
// 返回: 成功返回 ConfigWatcher，失败返回错误。
func NewWatcher(path string) (*ConfigWatcher, error) {
	c := &ConfigWatcher{path: path, running: true}
	if err := c.load(); err != nil {
		log.Printf("[config] warning: %v, starting with empty config", err)
	}
	if err := c.watch(); err != nil {
		log.Printf("[config] warning: watch failed: %v", err)
	}
	return c, nil
}

// load 加载配置文件内容
//
// 从 path 读取文件并解析为 YAML。内部使用，线程不安全。
// 返回: 解析失败返回错误。
func (c *ConfigWatcher) load() error {
	c.closeMu.Lock()
	if c.closed {
		c.closeMu.Unlock()
		return nil
	}
	c.closeMu.Unlock()

	data, err := os.ReadFile(c.path)
	if err != nil {
		return fmt.Errorf("read config file failed: %w", err)
	}

	var cfg map[string]interface{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse yaml failed: %w", err)
	}

	c.mu.Lock()
	c.data = cfg
	c.mu.Unlock()
	return nil
}

// watch 创建文件监听
//
// 使用 notify 库监听配置文件变化。内部使用。
// 返回: 监听失败返回错误。
func (c *ConfigWatcher) watch() error {
	c.watcherChan = make(chan notify.EventInfo, 100)

	if err := notify.Watch(c.path, c.watcherChan, notify.Write); err != nil {
		return fmt.Errorf("watch config file failed: %w", err)
	}

	c.wg.Add(1)
	go c.watchLoop()
	return nil
}

// watchLoop 文件变化监听循环
//
// 在后台 goroutine 运行，监听 watcherChan 上的文件事件。
// 当检测到文件写入事件时自动调用 load() 重载配置。
// 通过 wg.Done() 退出时通知 Wait()。
// running 为 false 时主动退出循环。
func (c *ConfigWatcher) watchLoop() {
	defer c.wg.Done()
	for event := range c.watcherChan {
		c.runningMu.Lock()
		if !c.running {
			c.runningMu.Unlock()
			return
		}
		c.runningMu.Unlock()

		if event.Event() == notify.Write {
			log.Printf("[config] file changed, reloading: path=%s", c.path)
			if err := c.load(); err != nil {
				log.Printf("[config] reload failed: path=%s, error=%v", c.path, err)
			}
		}
	}
}

// Get 根据点号路径获取配置值
//
// name: 点号分隔的路径，如 "server.port"。
// 返回: (*Val, bool) - 值和是否存在。
//
// 示例: cfg.Get("database.host") 等价于 data["database"]["host"]
func (c *ConfigWatcher) Get(name string) (*Val, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := strings.Split(name, ".")
	data := c.data
	for i, key := range keys {
		if i == len(keys)-1 {
			if v, ok := data[key]; ok {
				return &Val{val: v, ok: true}, true
			}
			return &Val{}, false
		}
		if next, ok := data[key].(map[string]interface{}); ok {
			data = next
		} else {
			return &Val{}, false
		}
	}
	return &Val{}, false
}

// Close 关闭配置监听器，释放资源
//
// 停止文件监听，等待 watchLoop goroutine 退出。
// 多次调用安全，会忽略后续调用。
// 返回: 始终返回 nil。
func (c *ConfigWatcher) Close() error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true

	c.runningMu.Lock()
	c.running = false
	c.runningMu.Unlock()

	c.mu.Lock()
	c.data = nil
	c.mu.Unlock()

	if c.watcherChan != nil {
		notify.Stop(c.watcherChan)
	}
	c.wg.Wait()
	return nil
}