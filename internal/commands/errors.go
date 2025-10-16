package commands

import "errors"

// reportedError 表示错误已在界面上输出，无需再次渲染
type reportedError struct {
	err error
}

func (r reportedError) Error() string {
	if r.err == nil {
		return ""
	}
	return r.err.Error()
}

func (r reportedError) Unwrap() error {
	return r.err
}

// wrapReportedError 包装已处理错误，避免重复输出
func wrapReportedError(err error) error {
	if err == nil {
		return nil
	}
	return reportedError{err: err}
}

// isReportedError 判断错误是否已经在界面展示
func isReportedError(err error) bool {
	var target reportedError
	return errors.As(err, &target)
}

// WrapReportedError 对外导出包装函数
func WrapReportedError(err error) error {
	return wrapReportedError(err)
}

// IsReportedError 对外导出判断函数
func IsReportedError(err error) bool {
	return isReportedError(err)
}
