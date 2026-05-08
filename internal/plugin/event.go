package plugin

// PluginEvent 插件事件类型
type PluginEvent uint32

const (
    EventPluginAdd PluginEvent = iota // 0: 插件新增
    EventPluginUpdate                 // 1: 插件更新
    EventPluginDelete                 // 2: 插件删除
)

// PluginEventData 事件携带的数据
type PluginEventData struct {
    Event     PluginEvent // 事件类型
    Plugin    *Plugin    // 新增/更新后的插件
    OldPlugin *Plugin    // 删除前的插件
}

// EventCallback 事件回调函数类型
type EventCallback func(data PluginEventData)

// eventEntry 事件条目
type eventEntry struct {
    callback EventCallback
    id       uintptr
}