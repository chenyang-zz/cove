package tool

// RunnerOption 配置 Runner 的调用行为。
type RunnerOption func(*Runner)

// WithErrorAsOutput 设置 Runner 是否把调用错误转换成 Output。
//
// enabled 为 true 时，Invoke 对未知工具或工具执行错误返回 nil error，并把错误写入
// Output.Text 和 Output.Metadata。enabled 为 false 时，错误会直接返回给调用方。
func WithErrorAsOutput(enabled bool) RunnerOption {
	return func(r *Runner) {
		r.errorAsOutput = enabled
	}
}
