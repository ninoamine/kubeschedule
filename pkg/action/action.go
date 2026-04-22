// Package action defines the core abstractions for scheduled Kubernetes
// operations. Using interfaces here lets the controller depend on behaviour
// contracts rather than concrete implementations, so new action types
// (Scale, Patch, Delete, …) can be added without touching the reconciler.
package action

import "context"

// Executor carries out a scheduled action against the Kubernetes API.
type Executor interface {
	Execute(ctx context.Context) error
}

// Validator checks whether an action's configuration is valid before
// it is accepted or executed.
type Validator interface {
	Validate() error
}

// StatusReporter returns human-readable status information about
// the last execution of an action.
type StatusReporter interface {
	ReportStatus() (string, error)
}
