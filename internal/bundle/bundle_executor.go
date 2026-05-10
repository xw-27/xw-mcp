package bundle

import "strings"

// DoAction 执行 action（统一入口）
// actionType: tool / prompt / resource / template
// actionID: bundleName_actionName 格式，如 math_add
// params: 调用参数，类型根据 actionType 而定
func (c *BundleContext) DoAction(actionType, actionID string, params interface{}) (interface{}, error) {
	bundleName, actionName, err := parseActionID(actionID)
	if err != nil {
		return nil, err
	}

	bundle, ok := c.Get(bundleName)
	if !ok {
		return nil, ErrBundleNotFound
	}

	if bundle.LoadState() != LoadStateFull {
		bundle.LockLoad()
		if bundle.LoadState() != LoadStateFull {
			if err := c.BundleLoader.FullLoad(bundle); err != nil {
				bundle.UnlockLoad()
				return nil, err
			}
		}
		bundle.UnlockLoad()
	}

	switch actionType {
	case ActionTypeTool:
		tool, ok := bundle.GetTool(actionName)
		if !ok {
			return nil, ErrActionNotFound
		}
		return tool.Execute(params)
	case ActionTypePrompt:
		prompt, ok := bundle.GetPrompt(actionName)
		if !ok {
			return nil, ErrActionNotFound
		}
		return prompt.Execute(params)
	case ActionTypeResource:
		resource, ok := bundle.GetResource(actionName)
		if !ok {
			return nil, ErrActionNotFound
		}
		return resource.Execute(params)
	case ActionTypeResourceTemplate:
		template, ok := bundle.GetTemplate(actionName)
		if !ok {
			return nil, ErrActionNotFound
		}
		return template.Execute(params)
	default:
		return nil, ErrInvalidActionType
	}
}

// parseActionID 解析 action ID
// 输入格式：bundleName_actionName
// 输出：bundleName, actionName
func parseActionID(actionID string) (bundleName, actionName string, err error) {
	parts := strings.Split(actionID, "_")
	if len(parts) != 2 {
		return "", "", ErrInvalidActionID
	}
	return parts[0], parts[1], nil
}
