package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ActionSpec struct {
	//+kubebuilder:validation:Required
	//+kubebuilder:validation:Enum=Scale;Patch;Delete;RolloutRestart;RotateSecret;Cordon;Exec;Apply
	Type ActionType `json:"type"`
	//+kubebuilder:validation:Required
	TargetRef string            `json:"targetRef"`
	Params    map[string]string `json:"params,omitempty"`
}

type ScheduledActionSpec struct {
	//+kubebuilder:validation:Required
	Schedule string  `json:"schedule"`
	TimeZone *string `json:"timeZone,omitempty"`
	//+kubebuilder:validation:Required
	Action ActionSpec `json:"action"`
	//+kubebuilder:default:false
	Suspend *bool `json:"suspend,omitempty"`
}

type ScheduledActionStatus struct {
	LastRun *metav1.Time `json:"lastRun,omitempty"`
	NextRun *metav1.Time `json:"nextRun,omitempty"`
	History []string     `json:"history,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Schedule",type="string",JSONPath=".spec.schedule"
// +kubebuilder:printcolumn:name="Last Run",type="string",JSONPath=".status.lastRun"

type ScheduledAction struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScheduledActionSpec   `json:"spec,omitempty"`
	Status ScheduledActionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type ScheduledActionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ScheduledAction `json:"items"`
}

type ActionType string

const (
	ActionTypeScale          ActionType = "Scale"
	ActionTypePatch          ActionType = "Patch"
	ActionTypeDelete         ActionType = "Delete"
	ActionTypeRolloutRestart ActionType = "RolloutRestart"
	ActionTypeRotateSecret   ActionType = "RotateSecret"
	ActionTypeCordon         ActionType = "Cordon"
	ActionTypeExec           ActionType = "Exec"
	ActionTypeApply          ActionType = "Apply"
)
