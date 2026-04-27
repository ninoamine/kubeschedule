package action_test

import (
	"testing"
	"time"

	"github.com/ninoamine/kubeschedule/pkg/action"
)

func TestNewActionConfig(t *testing.T) {
	tests := []struct {
		name        string
		opts        []action.Option
		wantTimeout time.Duration
		wantDryRun  bool
		wantRetries int
	}{
		{
			name:        "default with no options",
			opts:        nil,
			wantTimeout: 30 * time.Second,
			wantDryRun:  false,
			wantRetries: 1,
		},
		{
			name:        "custom timeout",
			opts:        []action.Option{action.WithTimeout(10 * time.Second)},
			wantTimeout: 10 * time.Second,
			wantDryRun:  false,
			wantRetries: 1,
		},
		{
			name:        "custom dry run",
			opts:        []action.Option{action.WithDryRun(true)},
			wantTimeout: 30 * time.Second,
			wantDryRun:  true,
			wantRetries: 1,
		},
		{
			name:        "custom retries",
			opts:        []action.Option{action.WithRetries(3)},
			wantTimeout: 30 * time.Second,
			wantDryRun:  false,
			wantRetries: 3,
		},
		{
			name:        "all options",
			opts:        []action.Option{action.WithTimeout(10 * time.Second), action.WithDryRun(true), action.WithRetries(3)},
			wantTimeout: 10 * time.Second,
			wantDryRun:  true,
			wantRetries: 3,
		},
		{
			name:        "mixed options",
			opts:        []action.Option{action.WithTimeout(10 * time.Second), action.WithRetries(3)},
			wantTimeout: 10 * time.Second,
			wantDryRun:  false,
			wantRetries: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := action.NewActionConfig(tt.opts...)
			if cfg.Timeout != tt.wantTimeout {
				t.Errorf("NewActionConfig(%v) timeout = %v, want %v", tt.opts, cfg.Timeout, tt.wantTimeout)
			}
			if cfg.DryRun != tt.wantDryRun {
				t.Errorf("NewActionConfig(%v) dryRun = %v, want %v", tt.opts, cfg.DryRun, tt.wantDryRun)
			}
			if cfg.Retries != tt.wantRetries {
				t.Errorf("NewActionConfig(%v) retries = %v, want %v", tt.opts, cfg.Retries, tt.wantRetries)
			}
		})
	}
}
