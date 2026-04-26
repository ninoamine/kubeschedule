package action

import "time"

type ActionConfig struct {
	Timeout time.Duration
	DryRun  bool
	Retries int
}

type Option func(*ActionConfig)

func WithTimeout(d time.Duration) Option {
	return func(c *ActionConfig) {
		c.Timeout = d
	}
}

func WithDryRun(enable bool) Option {
	return func(c *ActionConfig) {
		c.DryRun = enable
	}
}

func WithRetries(n int) Option {
	return func(c *ActionConfig) {
		c.Retries = n
	}
}

func NewActionConfig(opts ...Option) *ActionConfig {
	config := &ActionConfig{
		Timeout: 30 * time.Second,
		DryRun:  false,
		Retries: 1,
	}
	for _, opt := range opts {
		opt(config)
	}
	return config
}
