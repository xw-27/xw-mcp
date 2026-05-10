package bundle

import (
	"log"
	"strings"
)

// DoAction 执行 action（统一入口）
// actionType: tool / prompt / resource / template
// actionID: bundleName_actionName 格式，如 math_add
// params: 调用参数，类型根据 actionType 而定
//   - tool:      params 是工具输入参数对象
//   - prompt:    params 是模板变量 map
//   - resource:  params 是资源参数（由 JS 自己解析）
//   - template:  params 是模板参数（由 JS 自己解析）
//
// 执行流程：
// 1. 解析 actionID 获取 bundleName 和 actionName
// 2. 获取 bundle 实例
// 3. 如果 bundle 未 FullLoad，先进行 FullLoad（绑定 execute 函数）
// 4. 根据 actionType 调用对应的 action Execute 方法
//
// 返回值：
//   - 成功: (interface{}, nil)
//   - 失败: (nil, error)
func (c *BundleContext) DoAction(actionType, actionID string, params interface{}) (interface{}, error) {
	log.Printf("[bundle] DoAction: actionType=%s, actionID=%s, params=%v", actionType, actionID, params)

	bundleName, actionName, err := parseActionID(actionID)
	if err != nil {
		log.Printf("[bundle] DoAction: parseActionID failed, actionID=%s, error=%v", actionID, err)
		return nil, err
	}

	bundle, ok := c.Get(bundleName)
	if !ok {
		log.Printf("[bundle] DoAction: bundle not found, bundleName=%s", bundleName)
		return nil, ErrBundleNotFound
	}

	// 如果 bundle 未 FullLoad，先进行 FullLoad
	// 这里使用双重检查锁定（double-checked locking）保证线程安全
	if bundle.LoadState() != LoadStateFull {
		log.Printf("[bundle] DoAction: bundle not full loaded, bundleName=%s, currentState=%d", bundleName, bundle.LoadState())
		bundle.LockLoad()
		if bundle.LoadState() != LoadStateFull {
			log.Printf("[bundle] DoAction: performing FullLoad, bundleName=%s", bundleName)
			if err := c.BundleLoader.FullLoad(bundle); err != nil {
				bundle.UnlockLoad()
				log.Printf("[bundle] DoAction: FullLoad failed, bundleName=%s, error=%v", bundleName, err)
				return nil, err
			}
		}
		bundle.UnlockLoad()
	}

	switch actionType {
	case ActionTypeTool:
		tool, ok := bundle.GetTool(actionName)
		if !ok {
			log.Printf("[bundle] DoAction: tool not found, actionName=%s", actionName)
			return nil, ErrActionNotFound
		}
		log.Printf("[bundle] DoAction: executing tool, actionName=%s, params=%v", actionName, params)
		result, err := tool.Execute(params)
		if err != nil {
			log.Printf("[bundle] DoAction: tool execution failed, actionName=%s, error=%v", actionName, err)
		} else {
			log.Printf("[bundle] DoAction: tool execution success, actionName=%s, result=%v", actionName, result)
		}
		return result, err
	case ActionTypePrompt:
		prompt, ok := bundle.GetPrompt(actionName)
		if !ok {
			log.Printf("[bundle] DoAction: prompt not found, actionName=%s", actionName)
			return nil, ErrActionNotFound
		}
		return prompt.Execute(params)
	case ActionTypeResource:
		resource, ok := bundle.GetResource(actionName)
		if !ok {
			log.Printf("[bundle] DoAction: resource not found, actionName=%s", actionName)
			return nil, ErrActionNotFound
		}
		return resource.Execute(params)
	case ActionTypeResourceTemplate:
		template, ok := bundle.GetTemplate(actionName)
		if !ok {
			log.Printf("[bundle] DoAction: template not found, actionName=%s", actionName)
			return nil, ErrActionNotFound
		}
		return template.Execute(params)
	default:
		log.Printf("[bundle] DoAction: invalid action type, actionType=%s", actionType)
		return nil, ErrInvalidActionType
	}
}

// parseActionID 解析 action ID
// 输入格式：bundleName_actionName
// 输出：bundleName, actionName
// 例如：
//   - "math_add" -> bundleName="math", actionName="add"
//   - "test_hello" -> bundleName="test", actionName="hello"
func parseActionID(actionID string) (bundleName, actionName string, err error) {
	parts := strings.Split(actionID, "_")
	if len(parts) != 2 {
		return "", "", ErrInvalidActionID
	}
	return parts[0], parts[1], nil
}