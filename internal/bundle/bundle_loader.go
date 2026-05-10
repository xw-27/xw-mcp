package bundle

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/dop251/goja"
)

// BundleLoader Bundle加载器，负责解析和加载Bundle中的Action
// 提供三种加载模式：FullLoad、MetaLoad、Unload
type BundleLoader struct{}

// NewBundleLoader 创建新的BundleLoader实例
func NewBundleLoader() *BundleLoader {
	return &BundleLoader{}
}

// FullLoad 全量加载：解析所有Action元数据 + 绑定函数
// 执行index.js，提取完整的Action信息，包括execute/read函数绑定
func (l *BundleLoader) FullLoad(b *Bundle) error {
	l.Unload(b) // 先完整卸载，重置Runtime和状态
	err := l.loadBundle(b, true)
	if err != nil {
		log.Printf("[bundle] full load failed: %s, error: %v", b.Name(), err)
		return err
	}
	log.Printf("[bundle] full loaded: %s", b.Name())
	return nil
}

// MetaLoad 半加载：只解析Action元数据
// 执行index.js，只提取name/description/type/schema等信息，不绑定函数
func (l *BundleLoader) MetaLoad(b *Bundle) error {
	l.Unload(b) // 先完整卸载，重置Runtime和状态
	err := l.loadBundle(b, false)
	if err != nil {
		log.Printf("[bundle] meta load failed: %s, error: %v", b.Name(), err)
		return err
	}
	log.Printf("[bundle] meta loaded: %s", b.Name())
	return nil
}

// Unload 卸载：清空Bundle的Registry，重置Runtime，重置加载状态
func (l *BundleLoader) Unload(b *Bundle) {
	b.Registry().Clear()
	b.ResetRuntime() // 重置Runtime，避免模块缓存累积
	b.SetLoadState(LoadStateNone)
	log.Printf("[bundle] unloaded: %s", b.Name())
}

// loadBundle 内部加载方法
// fullLoad为true时绑定execute/read函数，为false时只解析元数据
func (l *BundleLoader) loadBundle(b *Bundle, fullLoad bool) error {
	// 获取Bundle的Runtime
	vm := b.GetRuntime()

	// 读取index.js内容
	indexPath := filepath.Join(b.Path(), b.IndexFile())
	content, err := os.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("read index.js failed: %w", err)
	}

	// 在Runtime中执行JS
	_, err = vm.RunString(string(content))
	if err != nil {
		return fmt.Errorf("execute js failed: %w", err)
	}

	// 获取module.exports
	module := vm.Get("module").ToObject(vm)
	exports := module.Get("exports")
	if goja.IsUndefined(exports) || goja.IsNull(exports) {
		return fmt.Errorf("module.exports is empty")
	}

	// 转换为对象
	exportsObj := exports.ToObject(vm)
	var items []goja.Value

	// 判断是数组还是单个对象
	if exportsObj.ClassName() == "Array" {
		length := exportsObj.Get("length").ToInteger()
		for i := int64(0); i < length; i++ {
			items = append(items, exportsObj.Get(fmt.Sprintf("%d", i)))
		}
	} else {
		items = []goja.Value{exports}
	}

	// 遍历并解析每个条目
	for _, item := range items {
		if goja.IsUndefined(item) || goja.IsNull(item) {
			continue
		}

		obj := item.ToObject(vm)
		if obj == nil {
			continue
		}

		// 提取type字段
		typeVal := obj.Get("type")
		if goja.IsUndefined(typeVal) || goja.IsNull(typeVal) {
			continue
		}
		actionType := typeVal.String()

		// 根据type分发解析
		switch actionType {
		case ActionTypeTool:
			l.parseToolAction(vm, obj, b, fullLoad)
		case ActionTypePrompt:
			l.parsePromptAction(vm, obj, b)
		case ActionTypeResource:
			l.parseResourceAction(vm, obj, b, fullLoad)
		case ActionTypeResourceTemplate:
			l.parseResourceTemplateAction(vm, obj, b, fullLoad)
		}
	}

	// 设置加载状态
	if fullLoad {
		b.SetLoadState(LoadStateFull)
	} else {
		b.SetLoadState(LoadStateMeta)
	}

	return nil
}

// parseToolAction 解析ToolAction
func (l *BundleLoader) parseToolAction(vm *goja.Runtime, obj *goja.Object, b *Bundle, fullLoad bool) {
	name := obj.Get("name").String()
	description := obj.Get("description").String()

	// 解析inputSchema
	var inputSchema map[string]interface{}
	schemaVal := obj.Get("inputSchema")
	if !goja.IsUndefined(schemaVal) && !goja.IsNull(schemaVal) {
		if m, ok := schemaVal.Export().(map[string]interface{}); ok {
			inputSchema = m
		}
	}

	tool := &ToolAction{
		BaseAction: BaseAction{
			Type:        ActionTypeTool,
			Name:        name,
			Description: description,
		},
		InputSchema: inputSchema,
	}

	// 如果是全加载，绑定execute函数
	if fullLoad {
		executeVal := obj.Get("execute")
		if !goja.IsUndefined(executeVal) && !goja.IsNull(executeVal) {
			if fn, ok := goja.AssertFunction(executeVal); ok {
				tool.Execute = func(params interface{}) (interface{}, error) {
					result, err := fn(goja.Undefined(), vm.ToValue(params))
					if err != nil {
						return nil, err
					}
					return result.Export(), nil
				}
			}
		}
	}

	b.RegisterTool(tool)
}

// parsePromptAction 解析PromptAction
func (l *BundleLoader) parsePromptAction(vm *goja.Runtime, obj *goja.Object, b *Bundle) {
	name := obj.Get("name").String()
	description := obj.Get("description").String()

	// 解析arguments
	var arguments []*PromptArgument
	argsVal := obj.Get("arguments")
	if !goja.IsUndefined(argsVal) && !goja.IsNull(argsVal) {
		argsObj := argsVal.ToObject(vm)
		if argsObj.ClassName() == "Array" {
			length := argsObj.Get("length").ToInteger()
			for i := int64(0); i < length; i++ {
				argObj := argsObj.Get(fmt.Sprintf("%d", i)).ToObject(vm)
				arg := &PromptArgument{
					Name:        argObj.Get("name").String(),
					Title:       argObj.Get("title").String(),
					Description: argObj.Get("description").String(),
					Required:    argObj.Get("required").ToBoolean(),
				}
				arguments = append(arguments, arg)
			}
		}
	}

	// 解析messages
	var messages []*PromptMessage
	msgVal := obj.Get("messages")
	if !goja.IsUndefined(msgVal) && !goja.IsNull(msgVal) {
		msgObj := msgVal.ToObject(vm)
		if msgObj.ClassName() == "Array" {
			length := msgObj.Get("length").ToInteger()
			for i := int64(0); i < length; i++ {
				msgItem := msgObj.Get(fmt.Sprintf("%d", i)).ToObject(vm)
				msg := &PromptMessage{
					Role:    msgItem.Get("role").String(),
					Content: msgItem.Get("content").String(),
				}
				messages = append(messages, msg)
			}
		}
	}

	prompt := &PromptAction{
		BaseAction: BaseAction{
			Type:        ActionTypePrompt,
			Name:        name,
			Description: description,
		},
		Arguments: arguments,
		Messages:  messages,
	}

	b.RegisterPrompt(prompt)
}

// parseResourceAction 解析ResourceAction
func (l *BundleLoader) parseResourceAction(vm *goja.Runtime, obj *goja.Object, b *Bundle, fullLoad bool) {
	name := obj.Get("name").String()
	description := obj.Get("description").String()
	uri := obj.Get("uri").String()
	mimeType := obj.Get("mimeType").String()

	resource := &ResourceAction{
		BaseAction: BaseAction{
			Type:        ActionTypeResource,
			Name:        name,
			Description: description,
		},
		URI:      uri,
		MIMEType: mimeType,
	}

	// 如果是全加载，绑定read函数
	if fullLoad {
		readVal := obj.Get("read")
		if !goja.IsUndefined(readVal) && !goja.IsNull(readVal) {
			if fn, ok := goja.AssertFunction(readVal); ok {
				resource.Read = func(uri string) (*ResourceContents, error) {
					result, err := fn(goja.Undefined(), vm.ToValue(uri))
					if err != nil {
						return nil, err
					}
					if obj, ok := result.Export().(map[string]interface{}); ok {
						rc := &ResourceContents{}
						if uri, ok := obj["uri"].(string); ok {
							rc.URI = uri
						}
						if mime, ok := obj["mimeType"].(string); ok {
							rc.MIMEType = mime
						}
						if text, ok := obj["text"].(string); ok {
							rc.Text = text
						}
						if blob, ok := obj["blob"].([]byte); ok {
							rc.Blob = blob
						}
						return rc, nil
					}
					return nil, fmt.Errorf("invalid resource contents")
				}
			}
		}
	}

	b.RegisterResource(resource)
}

// parseResourceTemplateAction 解析ResourceTemplateAction
func (l *BundleLoader) parseResourceTemplateAction(vm *goja.Runtime, obj *goja.Object, b *Bundle, fullLoad bool) {
	name := obj.Get("name").String()
	description := obj.Get("description").String()
	uriTemplate := obj.Get("uriTemplate").String()
	mimeType := obj.Get("mimeType").String()

	template := &ResourceTemplateAction{
		BaseAction: BaseAction{
			Type:        ActionTypeResourceTemplate,
			Name:        name,
			Description: description,
		},
		URITemplate: uriTemplate,
		MIMEType:    mimeType,
	}

	// 如果是全加载，绑定read函数
	if fullLoad {
		readVal := obj.Get("read")
		if !goja.IsUndefined(readVal) && !goja.IsNull(readVal) {
			if fn, ok := goja.AssertFunction(readVal); ok {
				template.Read = func(uri string) (*ResourceContents, error) {
					result, err := fn(goja.Undefined(), vm.ToValue(uri))
					if err != nil {
						return nil, err
					}
					if obj, ok := result.Export().(map[string]interface{}); ok {
						rc := &ResourceContents{}
						if uri, ok := obj["uri"].(string); ok {
							rc.URI = uri
						}
						if mime, ok := obj["mimeType"].(string); ok {
							rc.MIMEType = mime
						}
						if text, ok := obj["text"].(string); ok {
							rc.Text = text
						}
						if blob, ok := obj["blob"].([]byte); ok {
							rc.Blob = blob
						}
						return rc, nil
					}
					return nil, fmt.Errorf("invalid resource contents")
				}
			}
		}
	}

	b.RegisterTemplate(template)
}
