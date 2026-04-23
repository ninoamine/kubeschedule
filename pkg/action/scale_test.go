package action_test

import (
	"context"
	"testing"

	"github.com/ninoamine/kubeschedule/pkg/action"
)

var _ action.Executor = (*action.ScaleExecutor)(nil)

func TestScaleExecutor_Execute(t *testing.T) {
	exec := &action.ScaleExecutor{}
	if err := exec.Execute(context.Background()); err != nil {
		t.Fatalf("expected no error from no-op executor, got: %v", err)
	}
}
