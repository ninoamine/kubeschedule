package main

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestListDeployments(t *testing.T) {
	tests := []struct {
		name        string
		namespace   string
		deployments []appsv1.Deployment
		want        []string
	}{
		{
			name:        "no deployments",
			namespace:   "default",
			deployments: nil,
			want:        []string{},
		},
		{
			name:      "single deployment",
			namespace: "default",
			deployments: []appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "nginx", Namespace: "default",
					},
				},
			},
			want: []string{"nginx"},
		},
		{
			name:      "multiple deployments",
			namespace: "default",
			deployments: []appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "nginx", Namespace: "default",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "redis", Namespace: "default",
					},
				},
			},
			want: []string{"nginx", "redis"},
		},
		{
			name:      "filtre by namespace",
			namespace: "production",
			deployments: []appsv1.Deployment{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "nginx", Namespace: "production",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "redis", Namespace: "qa",
					},
				},
			},
			want: []string{"nginx"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objs := make([]runtime.Object, len(tt.deployments))
			for i := range tt.deployments {
				objs[i] = &tt.deployments[i]
			}
			client := fake.NewSimpleClientset(objs...)

			got, err := ListDeployments(context.Background(), client, tt.namespace)
			if err != nil {
				t.Fatalf("ListDeployments(%q) error = %v", tt.namespace, err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d deployments, want %d", len(got), len(tt.want))
			}
			for i, name := range got {
				if name != tt.want[i] {
					t.Errorf("got[%d] = %q, want %q", i, name, tt.want[i])
				}
			}
		})
	}

}
