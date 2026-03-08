// Package k8s provides a shared Kubernetes clientset factory for all k8s-dependent
// components. It supports in-cluster configuration and optional kubeconfig file paths.
package k8s

import (
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// NewClientset creates a Kubernetes clientset. If inCluster is true, it uses
// in-cluster service account credentials. Otherwise it reads the kubeconfig file.
func NewClientset(inCluster bool, kubeconfig string) (kubernetes.Interface, error) {
	var cfg *rest.Config
	var err error

	if inCluster {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("in-cluster config: %w", err)
		}
	} else {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("kubeconfig %q: %w", kubeconfig, err)
		}
	}

	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create clientset: %w", err)
	}
	return cs, nil
}
