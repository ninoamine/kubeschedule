package main

import (
	"context"

	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func ListDeployments(ctx context.Context, client kubernetes.Interface, namespace string) ([]string, error) {
	deployments, err := client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(deployments.Items))
	for _, d := range deployments.Items {
		names = append(names, d.Name)
	}
	return names, nil
}

func main() {
	config, err := clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	if err != nil {
		panic(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	names, err := ListDeployments(context.Background(), clientset, "default")
	if err != nil {
		panic(err)
	}

	fmt.Printf("Found %d deployments\n", len(names))
	for _, name := range names {
		fmt.Printf("Deployment: %s\n", name)
	}
}
