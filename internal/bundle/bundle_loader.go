package bundle

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/dop251/goja"
)

// BundleLoader Bundle加载器，负责解析和加载Bundle中的Action
// 提供三种加载模式：FullLoad、MetaLoad、Unload
type BundleLoader struct{}

// NewBundleLoader 创建新的BundleLoader实例
func NewBundleLoader() *BundleLoader {
	return &BundleLoader{}
}

// validateBundleName 验证bundle名称是否合法
func (l *BundleLoader) validateBundleName(name string) error {
	if strings.Contains(name, " ") {
		return fmt.Errorf("bundle name '%s' contains spaces", name)
	}
	return nil
}

// FullLoad 全量加载：解析所有Action元数据 + 绑定函数
// 执行index.js，提取完整的Action信息，包括execute/read函数绑定
func (l *BundleLoader) FullLoad(b *Bundle) error {
	if err := l.validateBundleName(b.Name()); err != nil {
		b.SetLoadState(LoadStateError)
		b.SetLoadError(err)
		log.Printf("[bundle] full load failed: %s, error: %v", b.Name(), err)
		return err
	}
	l.Unload(b)
	err := l.loadBundle(b, true)
	if err != nil {
		b.SetLoadState(LoadStateError)
		b.SetLoadError(err)
		log.Printf("[bundle] full load failed: %s, error: %v", b.Name(), err)
		return err
	}
	log.Printf("[bundle] full loaded: %s", b.Name())
	return nil
}

// MetaLoad 半加载：只解析Action元数据
// 执行index.js，只提取name/description/type/schema等信息，不绑定函数
func (l *BundleLoader) MetaLoad(b *Bundle) error {
	if err := l.validateBundleName(b.Name()); err != nil {
		b.SetLoadState(LoadStateError)
		b.SetLoadError(err)
		log.Printf("[bundle] meta load failed: %s, error: %v", b.Name(), err)
		return err
	}
	l.Unload(b)
	err := l.loadBundle(b, false)
	if err != nil {
		b.SetLoadState(LoadStateError)
		b.SetLoadError(err)
		log.Printf("[bundle] meta load failed: %s, error: %v", b.Name(), err)
		return err
	}
	log.Printf("[bundle] meta loaded: %s", b.Name())
	return nil
}

// Unload 卸载：清空Bundle的Registry，重置Runtime，重置加载状态
func (l *BundleLoader) Unload(b *Bundle) {
	b.Close()
	log.Printf("[bundle] unloaded: %s", b.Name())
}

// loadBundle 内部加载方法
// fullLoad为true时绑定execute/read函数，为false时只解析元数据
func (l *BundleLoader) loadBundle(b *Bundle, fullLoad bool) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic during load: %v", r)
		}
	}()

	// 获取Bundle的Runtime
	vm := b.GetRuntime()

	// 读取index.js内容
	indexPath := filepath.Join(b.Path(), b.IndexFile())
	content, err := os.ReadFile(indexPath)
	if err != nil {
		return fmt.Errorf("read index.js failed: %w", err)
	}

	// 设置CommonJS模块环境（module.exports等）
	moduleObj := vm.NewObject()
	exportsObj := vm.NewObject()
	moduleObj.Set("exports", exportsObj)
	vm.Set("module", moduleObj)
	vm.Set("exports", exportsObj)

	// 在Runtime中执行JS
	_, err = vm.RunString(string(content))
	if err != nil {
		return fmt.Errorf("execute js failed: %w", err)
	}

	// 获取module.exports
	moduleVal := vm.Get("module").ToObject(vm)
	exportsVal := moduleVal.Get("exports")
	if goja.IsUndefined(exportsVal) || goja.IsNull(exportsVal) {
		return fmt.Errorf("module.exports is empty")
	}

	// 转换为对象
	exportsObj = exportsVal.ToObject(vm)
	var items []goja.Value

	// 判断是数组还是单个对象
	if exportsObj.ClassName() == "Array" {
		length := exportsObj.Get("length").ToInteger()
		for i := int64(0); i < length; i++ {
			items = append(items, exportsObj.Get(fmt.Sprintf("%d", i)))
		}
	} else {
		items = []goja.Value{exportsVal}
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
		actionType := typeVal.ToString().String()

		// 根据type分发解析
		switch actionType {
		case ActionTypeTool:
			l.parseToolAction(vm, obj, b, fullLoad)
		case ActionTypePrompt:
			l.parsePromptAction(vm, obj, b, fullLoad)
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
				log.Printf("[bundle] binding execute function for tool: %s", name)
				tool.Execute = func(params interface{}) (interface{}, error) {
					log.Printf("[bundle] execute tool: %s, params type=%T, params=%v", name, params, params)
					jsParams := vm.ToValue(params)
					log.Printf("[bundle] execute tool: %s, jsParams type=%T, jsParams=%v", name, jsParams, jsParams)
					result, err := fn(goja.Undefined(), jsParams)
					if err != nil {
						log.Printf("[bundle] execute tool error: %s, error=%v", name, err)
						return nil, err
					}
					log.Printf("[bundle] execute tool result: %s, result=%v", name, result.Export())
					return result.Export(), nil
				}
			}
		}
	}

	b.RegisterTool(tool)
}

// parsePromptAction 解析PromptAction
func (l *BundleLoader) parsePromptAction(vm *goja.Runtime, obj *goja.Object, b *Bundle, fullLoad bool) {
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

	// 如果是全加载，绑定execute函数
	if fullLoad {
		executeVal := obj.Get("execute")
		if !goja.IsUndefined(executeVal) && !goja.IsNull(executeVal) {
			if fn, ok := goja.AssertFunction(executeVal); ok {
				prompt.Execute = func(params interface{}) (interface{}, error) {
					result, err := fn(goja.Undefined(), vm.ToValue(params))
					if err != nil {
						return nil, err
					}
					return result.Export(), nil
				}
			}
		}
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

	if fullLoad {
		executeVal := obj.Get("execute")
		if !goja.IsUndefined(executeVal) && !goja.IsNull(executeVal) {
			if fn, ok := goja.AssertFunction(executeVal); ok {
				resource.Execute = func(params interface{}) (interface{}, error) {
					result, err := fn(goja.Undefined(), vm.ToValue(params))
					if err != nil {
						return nil, err
					}
					return result.Export(), nil
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

	if fullLoad {
		executeVal := obj.Get("execute")
		if !goja.IsUndefined(executeVal) && !goja.IsNull(executeVal) {
			if fn, ok := goja.AssertFunction(executeVal); ok {
				template.Execute = func(params interface{}) (interface{}, error) {
					result, err := fn(goja.Undefined(), vm.ToValue(params))
					if err != nil {
						return nil, err
					}
					return result.Export(), nil
				}
			}
		}
	}

	b.RegisterTemplate(template)
}
