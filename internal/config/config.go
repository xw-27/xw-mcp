package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
)

// Config 配置管理结构体
//
// 提供 YAML 配置文件的加载、监听、访问和关闭功能
type Config struct {
	path    string
	data    map[string]interface{}
	watcher *fsnotify.Watcher
	mu      sync.RWMutex
	once    sync.Once
	closed  bool
	closeMu sync.Mutex
}

// New 创建 Config 实例，加载配置并启动热更新监听
//
// 参数:
//   - path: 配置文件路径
//
// 返回:
//   - *Config: 配置实例
//   - error: 加载或监听失败时返回错误
func New(path string) (*Config, error) {
	c := &Config{path: path}
	if err := c.load(); err != nil {
		return nil, err
	}
	if err := c.watch(); err != nil {
		return nil, err
	}
	return c, nil
}

// load 从文件加载配置
func (c *Config) load() error {
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

// watch 启动文件监听
func (c *Config) watch() error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher failed: %w", err)
	}
	c.watcher = w

	if err := w.Add(c.path); err != nil {
		return fmt.Errorf("watch config file failed: %w", err)
	}

	go c.watchLoop()
	return nil
}

// watchLoop 文件监听循环
func (c *Config) watchLoop() {
	for {
		select {
		case event, ok := <-c.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Println("[config] config file changed, reloading...")
				if err := c.load(); err != nil {
					log.Printf("[config] reload failed: %v", err)
				} else {
					log.Println("[config] reload success")
				}
			}
		case err, ok := <-c.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("[config] watcher error: %v", err)
		}
	}
}

// Get 根据配置名称获取配置值
//
// 支持点号分隔的嵌套路径访问，如 "server.port"
//
// 参数:
//   - name: 配置名称
//
// 返回:
//   - *Value: 配置值包装
//   - bool: 配置项是否存在
func (c *Config) Get(name string) (*Value, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	keys := strings.Split(name, ".")
	data := c.data
	for i, key := range keys {
		if i == len(keys)-1 {
			if v, ok := data[key]; ok {
				return &Value{val: v, ok: true}, true
			}
			return &Value{}, false
		}
		if next, ok := data[key].(map[string]interface{}); ok {
			data = next
		} else {
			return &Value{}, false
		}
	}
	return &Value{}, false
}

// Close 关闭配置监听器并清理资源
//
// 支持多次调用，线程安全。使用 defer 自动回收。
//
// 返回:
//   - error: 关闭失败时返回错误
func (c *Config) Close() error {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true

	c.mu.Lock()
	for k := range c.data {
		delete(c.data, k)
	}
	c.data = nil
	c.mu.Unlock()

	c.once.Do(func() {
		if c.watcher != nil {
			c.watcher.Close()
		}
	})
	return nil
}