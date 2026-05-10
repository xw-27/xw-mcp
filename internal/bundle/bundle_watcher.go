package bundle

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rjeczalik/notify"
)

// BundleEventType 事件类型常量
const (
	BundleEventLoad   = "load"    // 加载事件（新增或更新）
	BundleEventUnload = "unload" // 卸载事件（目录删除或index.js删除）
)

const (
	defaultWatchDepth   = 3                     // 默认监控深度（层数）
	defaultScanInterval = 2 * time.Second      // 定时扫描间隔（用于检测unload）
)

// BundleWatchEvent 监控事件结构
type BundleWatchEvent struct {
	EventType  string // 事件类型：load/unload
	BundleName string // 插件名称（目录名）
	BundlePath string // 插件路径（完整路径）
}

// BundleWatchCallback 回调函数类型
// 当插件发生加载或卸载事件时，调用此回调通知外部
type BundleWatchCallback func(event BundleWatchEvent)

// BundleWatcher Bundle文件监控器
// 负责监控bundles目录下插件文件的变化，通知外部
type BundleWatcher struct {
	mu           sync.RWMutex           // 读写锁
	watchPath    string                 // 监控目录路径
	depth        int                    // 监控深度（层数）
	watcher      chan notify.EventInfo  // 文件监控通道
	callbacks    []BundleWatchCallback  // 回调函数列表
	running      bool                   // 是否正在运行
	knownBundles map[string]string      // 已知的bundle：name -> path
	stopCh       chan struct{}          // 停止信号通道
}

// NewBundleWatcher 创建新的BundleWatcher实例
func NewBundleWatcher() *BundleWatcher {
	return &BundleWatcher{
		depth:        defaultWatchDepth,
		knownBundles: make(map[string]string),
		stopCh:       make(chan struct{}),
	}
}

// Watch 启动监控
// bundlesPath: 监控的目录路径（如 "./plugins"）
// depth: 监控深度（层数），默认为3层
func (w *BundleWatcher) Watch(bundlesPath string, depth int) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return ErrBundleAlreadyExists
	}

	w.watchPath = bundlesPath
	if depth > 0 {
		w.depth = depth
	}
	w.running = true
	w.mu.Unlock()

	log.Printf("[bundle] watcher starting: path=%s, depth=%d", bundlesPath, w.depth)

	// 首次扫描，通知已存在的bundle
	w.scanAndNotify()

	// 启动文件监控
	w.watcher = make(chan notify.EventInfo, 100)
	watchPath := filepath.Join(bundlesPath, "...")
	if err := notify.Watch(watchPath, w.watcher, notify.All); err != nil {
		w.Stop()
		return err
	}

	// 启动文件变化监听循环
	go w.watchLoop()

	// 启动定时扫描（检测unload）
	go w.startPeriodicScan()

	log.Printf("[bundle] watcher started: path=%s", bundlesPath)
	return nil
}

// Stop 停止监控
func (w *BundleWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.running {
		return
	}

	w.running = false

	if w.watcher != nil {
		notify.Stop(w.watcher)
		w.watcher = nil
	}

	close(w.stopCh)
	w.stopCh = make(chan struct{})

	log.Printf("[bundle] watcher stopped")
}

// Close 关闭监控器（支持defer延迟调用）
func (w *BundleWatcher) Close() error {
	w.Stop()
	return nil
}

// AddCallback 添加回调函数
func (w *BundleWatcher) AddCallback(callback BundleWatchCallback) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.callbacks = append(w.callbacks, callback)
}

// RemoveCallback 移除回调函数
func (w *BundleWatcher) RemoveCallback(callback BundleWatchCallback) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for i, cb := range w.callbacks {
		if &cb == &callback {
			w.callbacks = append(w.callbacks[:i], w.callbacks[i+1:]...)
			break
		}
	}
}

// emit 触发回调通知
func (w *BundleWatcher) emit(event BundleWatchEvent) {
	w.mu.RLock()
	callbacks := w.callbacks
	w.mu.RUnlock()

	log.Printf("[bundle] watcher emit: type=%s, bundle=%s, path=%s", event.EventType, event.BundleName, event.BundlePath)

	for _, cb := range callbacks {
		cb(event)
	}
}

// watchLoop 文件变化监听循环
func (w *BundleWatcher) watchLoop() {
	for {
		select {
		case <-w.stopCh:
			return
		case event, ok := <-w.watcher:
			if !ok {
				return
			}
			w.handleFileEvent(event)
		}
	}
}

// handleFileEvent 处理文件变化事件
// 只处理.js文件变化，触发load事件
func (w *BundleWatcher) handleFileEvent(event notify.EventInfo) {
	// 只处理.js文件
	if !strings.HasSuffix(event.Path(), ".js") {
		log.Printf("[bundle] file changed: skipped (not .js), path=%s", event.Path())
		return
	}

	bundlePath := w.findBundlePath(event.Path())
	if bundlePath == "" {
		log.Printf("[bundle] file changed: skipped (no bundle), path=%s", event.Path())
		return
	}

	bundleName := filepath.Base(bundlePath)

	// 记录已知的bundle
	w.mu.Lock()
	if _, known := w.knownBundles[bundleName]; !known {
		w.knownBundles[bundleName] = bundlePath
	}
	w.mu.Unlock()

	log.Printf("[bundle] file changed: path=%s, bundle=%s", event.Path(), bundleName)

	// 触发load事件
	w.emit(BundleWatchEvent{
		EventType:  BundleEventLoad,
		BundleName: bundleName,
		BundlePath: bundlePath,
	})
}

// scanAndNotify 首次扫描并通知已存在的bundle
func (w *BundleWatcher) scanAndNotify() {
	w.mu.Lock()

	entries, err := os.ReadDir(w.watchPath)
	if err != nil {
		log.Printf("[bundle] watcher scan dir failed: %v", err)
		w.mu.Unlock()
		return
	}

	newKnown := make(map[string]string)
	discoveredCount := 0

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		bundlePath := filepath.Join(w.watchPath, entry.Name())
		indexFile := filepath.Join(bundlePath, "index.js")

		// 跳过没有index.js的目录
		if _, err := os.Stat(indexFile); os.IsNotExist(err) {
			log.Printf("[bundle] skipped: no index.js, path=%s", bundlePath)
			continue
		}

		bundleName := entry.Name()
		newKnown[bundleName] = bundlePath

		// 如果是新发现的bundle，触发load事件
		if _, known := w.knownBundles[bundleName]; !known {
			discoveredCount++
			log.Printf("[bundle] discovered: bundle=%s, path=%s", bundleName, bundlePath)
		}
	}

	totalCount := len(newKnown)
	w.mu.Unlock()

	// 释放锁后再触发事件，避免死锁
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		bundleName := entry.Name()
		bundlePath, ok := newKnown[bundleName]
		if !ok {
			continue
		}

		// 检查是否是新发现的（使用上一次的 knownBundles 比较）
		w.mu.RLock()
		_, known := w.knownBundles[bundleName]
		w.mu.RUnlock()

		if !known {
			w.emit(BundleWatchEvent{
				EventType:  BundleEventLoad,
				BundleName: bundleName,
				BundlePath: bundlePath,
			})
		}
	}

	// 最后更新 knownBundles
	w.mu.Lock()
	w.knownBundles = newKnown
	w.mu.Unlock()

	log.Printf("[bundle] watcher scanned: total=%d, discovered=%d", totalCount, discoveredCount)
}

// findBundlePath 根据文件路径查找所属的bundle路径
func (w *BundleWatcher) findBundlePath(filePath string) string {
	// 先从已知bundle中查找
	w.mu.RLock()
	defer w.mu.RUnlock()

	for _, path := range w.knownBundles {
		if strings.HasPrefix(filePath, path) {
			return path
		}
	}

	// 向上遍历目录查找index.js
	dir := filepath.Dir(filePath)
	for i := 0; i < w.depth; i++ {
		indexFile := filepath.Join(dir, "index.js")
		if _, err := os.Stat(indexFile); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}

	return ""
}

// startPeriodicScan 启动定时扫描
// 定期检查bundle是否被删除（目录删除或index.js删除），触发unload事件
func (w *BundleWatcher) startPeriodicScan() {
	ticker := time.NewTicker(defaultScanInterval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.checkUnloadedBundles()
		}
	}
}

// checkUnloadedBundles 检查被卸载的bundle
// 如果已知bundle的index.js不存在了，触发unload事件
func (w *BundleWatcher) checkUnloadedBundles() {
	w.mu.Lock()
	defer w.mu.Unlock()

	for name, path := range w.knownBundles {
		indexFile := filepath.Join(path, "index.js")
		if _, err := os.Stat(indexFile); os.IsNotExist(err) {
			delete(w.knownBundles, name)
			log.Printf("[bundle] disappeared: bundle=%s, path=%s", name, path)
			w.emit(BundleWatchEvent{
				EventType:  BundleEventUnload,
				BundleName: name,
				BundlePath: path,
			})
		}
	}
}
