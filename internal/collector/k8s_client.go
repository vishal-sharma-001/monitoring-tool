package collector

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

// K8sClient wraps Kubernetes client
type K8sClient struct {
	clientset        *kubernetes.Clientset
	metricsClientset *metricsclientset.Clientset
	metricsClient    *MetricsClient
	stopCh           chan struct{}
	mu               sync.RWMutex
}

// NewK8sClient creates a new K8s client using kubeconfig
func NewK8sClient() (*K8sClient, error) {
	config, err := getKubeConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	metricsClientset, err := metricsclientset.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create metrics clientset: %w", err)
	}

	metricsClient := NewMetricsClient(clientset, metricsClientset)

	return &K8sClient{
		clientset:        clientset,
		metricsClientset: metricsClientset,
		metricsClient:    metricsClient,
		stopCh:           make(chan struct{}),
	}, nil
}

// getKubeConfig returns Kubernetes REST config
func getKubeConfig() (*rest.Config, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Fall back to kubeconfig file
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
	}

	return config, nil
}

// GetClientset returns the Kubernetes clientset
func (kc *K8sClient) GetClientset() *kubernetes.Clientset {
	return kc.clientset
}

// GetMetricsClient returns the metrics client
func (kc *K8sClient) GetMetricsClient() *MetricsClient {
	return kc.metricsClient
}

// Start initializes the client
func (kc *K8sClient) Start(ctx context.Context) {
	go func() {
		<-ctx.Done()
		kc.Stop()
	}()
}

// Stop gracefully stops the client
func (kc *K8sClient) Stop() {
	kc.mu.Lock()
	defer kc.mu.Unlock()
	select {
	case <-kc.stopCh:
	default:
		close(kc.stopCh)
	}
}

// GetStopChannel returns the stop channel
func (kc *K8sClient) GetStopChannel() <-chan struct{} {
	return kc.stopCh
}
