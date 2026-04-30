package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ActionSpec struct {
	Type      string            `json:"type"`
	TargetRef string            `json:"targetRef"`
	Params    map[string]string `json:"params,omitempty"`
}

type ScheduledActionSpec struct {
	Schedule string     `json:"schedule"`
	TimeZone *string    `json:"timeZone,omitempty"`
	Action   ActionSpec `json:"action"`
	Suspend  *bool      `json:"suspend,omitempty"`
}

type ScheduledActionStatus struct {
	LastRun *metav1.Time `json:"lastRun,omitempty"`
	NextRun *metav1.Time `json:"nextRun,omitempty"`
	History []string     `json:"history,omitempty"`
}

type ScheduledAction struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScheduledActionSpec   `json:"spec,omitempty"`
	Status ScheduledActionStatus `json:"status,omitempty"`
}

type ScheduledActionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ScheduledAction `json:"items"`
}
