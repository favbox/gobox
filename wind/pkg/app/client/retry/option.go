package retry

import "time"

// Option 用于设置重试选项的唯一结构体。
type Option struct {
	F func(o *Options)
}

// Options 重试选项
type Options struct {
	// 调用尝试的最大次数，包括初始调用
	MaxAttemptTimes uint

	// 初始重试的延迟时间
	Delay time.Duration

	// 重试的最大延迟时间
	MaxDelay time.Duration

	// 随机延迟的最大抖动时间，当延迟策略为随机延迟时生效
	MaxJitter time.Duration

	// 延迟策略，可组合使用多种策略。
	// 例如 CombineDelay(BackOffDelayPolicy, RandomDelayPolicy) 或 BackOffDelayPolicy 等。
	DelayPolicy DelayPolicyFunc
}

func (o *Options) Apply(opts []Option) {
	for _, opt := range opts {
		opt.F(o)
	}
}

// WithMaxAttemptTimes 设置重试的最大尝试次数，包括初始调用。
func WithMaxAttemptTimes(maxAttemptTimes uint) Option {
	return Option{
		F: func(o *Options) {
			o.MaxAttemptTimes = maxAttemptTimes
		},
	}
}

// WithInitDelay 设置初始重试的延迟时间。
func WithInitDelay(delay time.Duration) Option {
	return Option{
		F: func(o *Options) {
			o.Delay = delay
		},
	}
}

// WithMaxDelay 设置重试的最大延迟时间。
func WithMaxDelay(maxDelay time.Duration) Option {
	return Option{F: func(o *Options) {
		o.MaxDelay = maxDelay
	}}
}

// WithMaxJitter 设置随机延迟的最大抖动时间。
func WithMaxJitter(maxJitter time.Duration) Option {
	return Option{F: func(o *Options) {
		o.MaxJitter = maxJitter
	}}
}

// WithDelayPolicy 设置重试的延迟策略。
func WithDelayPolicy(delayPolicy DelayPolicyFunc) Option {
	return Option{F: func(o *Options) {
		o.DelayPolicy = delayPolicy
	}}
}
