package bundle

import "errors"

// 错误定义
var (
	ErrBundleNotFound      = errors.New("bundle not found")       // Bundle未找到
	ErrBundleAlreadyExists = errors.New("bundle already exists")  // Bundle已存在
	ErrActionNotFound      = errors.New("action not found")        // Action未找到
	ErrActionAlreadyExists = errors.New("action already exists")  // Action已存在
	ErrInvalidActionType   = errors.New("invalid action type")     // 无效的Action类型
	ErrBundlePathNotFound  = errors.New("bundle path not found")   // Bundle路径未找到
	ErrIndexFileNotFound   = errors.New("index file not found")   // 入口文件未找到
	ErrLoadFailed          = errors.New("load failed")            // 加载失败
	ErrExecuteFailed       = errors.New("execute failed")         // 执行失败
)

// BundleError Bundle相关错误
type BundleError struct {
	BundleName string  // Bundle名称
	Err        error   // 错误原因
}

// Error 返回错误描述
func (e *BundleError) Error() string {
	return e.BundleName + ": " + e.Err.Error()
}

// Unwrap 解包错误
func (e *BundleError) Unwrap() error {
	return e.Err
}

// NewBundleError 创建BundleError
func NewBundleError(bundleName string, err error) *BundleError {
	return &BundleError{
		BundleName: bundleName,
		Err:        err,
	}
}

// ActionError Action相关错误
type ActionError struct {
	ActionName string  // Action名称
	BundleName string  // 所属Bundle名称
	Err        error   // 错误原因
}

// Error 返回错误描述
func (e *ActionError) Error() string {
	return e.ActionName + " (in " + e.BundleName + "): " + e.Err.Error()
}

// Unwrap 解包错误
func (e *ActionError) Unwrap() error {
	return e.Err
}

// NewActionError 创建ActionError
func NewActionError(actionName, bundleName string, err error) *ActionError {
	return &ActionError{
		ActionName: actionName,
		BundleName: bundleName,
		Err:        err,
	}
}