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

// ConfigManager 配置管理结构体
//
// 提供 YAML 配置文件的加载、监听、访问和关闭功能
type ConfigManager struct {
	path        string
	data        map[string]interface{}
	watcherChan chan notify.EventInfo
	mu          sync.RWMutex
	once        sync.Once
	closed      bool
	closeMu     sync.Mutex
}

func New(path string) (*ConfigManager, error) {
	c := &ConfigManager{path: path}
	if err := c.load(); err != nil {
		return nil, err
	}
	if err := c.watch(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *ConfigManager) load() error {
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

func (c *ConfigManager) watch() error {
	c.watcherChan = make(chan notify.EventInfo, 100)

	if err := notify.Watch(c.path, c.watcherChan, notify.All); err != nil {
		return fmt.Errorf("watch config file failed: %w", err)
	}

	go c.watchLoop()
	return nil
}

func (c *ConfigManager) watchLoop() {
	for {
		select {
		case event, ok := <-c.watcherChan:
			if !ok {
				return
			}
			if event.Event() == notify.Write {
				log.Println("[config] config file changed, reloading...")
				if err := c.load(); err != nil {
					log.Printf("[config] reload failed: %v", err)
				} else {
					log.Println("[config] reload success")
				}
			}
		}
	}
}

func (c *ConfigManager) Get(name string) (*Value, bool) {
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

func (c *ConfigManager) Close() error {
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
		if c.watcherChan != nil {
			notify.Stop(c.watcherChan)
		}
	})
	return nil
}