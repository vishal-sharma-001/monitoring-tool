package collector

import (
	"context"
	"fmt"

	"github.com/monitoring-engine/monitoring-tool/internal/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

// MetricsClient wraps Kubernetes metrics API client
type MetricsClient struct {
	clientset        *kubernetes.Clientset
	metricsClientset *metricsclientset.Clientset
}

// PodMetrics represents CPU and memory usage for a pod
type PodMetrics struct {
	Namespace          string
	PodName            string
	CPUUsageMillicores int64
	CPUUsagePercent    float64
	MemoryUsageBytes   int64
	MemoryUsagePercent float64
	CPURequestMillis   int64
	MemoryRequestBytes int64
}

// NodeMetrics represents CPU and memory usage for a node
type NodeMetrics struct {
	NodeName           string
	CPUUsageMillicores int64
	CPUUsagePercent    float64
	MemoryUsageBytes   int64
	MemoryUsagePercent float64
	CPUCapacityMillis  int64
	MemoryCapacityBytes int64
}

// NewMetricsClient creates a new metrics client
func NewMetricsClient(clientset *kubernetes.Clientset, metricsClientset *metricsclientset.Clientset) *MetricsClient {
	return &MetricsClient{
		clientset:        clientset,
		metricsClientset: metricsClientset,
	}
}

// GetPodMetrics retrieves metrics for a specific pod
func (mc *MetricsClient) GetPodMetrics(ctx context.Context, namespace, podName string) (*PodMetrics, error) {
	// Get metrics from metrics-server
	podMetrics, err := mc.metricsClientset.MetricsV1beta1().PodMetricses(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod metrics: %w", err)
	}

	// Get pod details for resource requests
	pod, err := mc.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod details: %w", err)
	}

	var totalCPUUsage int64
	var totalMemoryUsage int64
	var totalCPURequest int64
	var totalMemoryRequest int64

	// Sum usage across all containers
	for _, container := range podMetrics.Containers {
		cpuQuantity := container.Usage.Cpu()
		memQuantity := container.Usage.Memory()
		totalCPUUsage += cpuQuantity.MilliValue()
		totalMemoryUsage += memQuantity.Value()
	}

	// Sum requests across all containers
	for _, container := range pod.Spec.Containers {
		if cpuRequest := container.Resources.Requests.Cpu(); cpuRequest != nil {
			totalCPURequest += cpuRequest.MilliValue()
		}
		if memRequest := container.Resources.Requests.Memory(); memRequest != nil {
			totalMemoryRequest += memRequest.Value()
		}
	}

	// Calculate percentages based on requests
	var cpuPercent float64
	var memPercent float64

	if totalCPURequest > 0 {
		cpuPercent = (float64(totalCPUUsage) / float64(totalCPURequest)) * 100
	}

	if totalMemoryRequest > 0 {
		memPercent = (float64(totalMemoryUsage) / float64(totalMemoryRequest)) * 100
	}

	return &PodMetrics{
		Namespace:          namespace,
		PodName:            podName,
		CPUUsageMillicores: totalCPUUsage,
		CPUUsagePercent:    cpuPercent,
		MemoryUsageBytes:   totalMemoryUsage,
		MemoryUsagePercent: memPercent,
		CPURequestMillis:   totalCPURequest,
		MemoryRequestBytes: totalMemoryRequest,
	}, nil
}

// GetNodeMetrics retrieves metrics for a specific node
func (mc *MetricsClient) GetNodeMetrics(ctx context.Context, nodeName string) (*NodeMetrics, error) {
	// Get metrics from metrics-server
	nodeMetrics, err := mc.metricsClientset.MetricsV1beta1().NodeMetricses().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get node metrics: %w", err)
	}

	// Get node details for capacity
	node, err := mc.clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get node details: %w", err)
	}

	cpuUsage := nodeMetrics.Usage.Cpu().MilliValue()
	memUsage := nodeMetrics.Usage.Memory().Value()
	cpuCapacity := node.Status.Capacity.Cpu().MilliValue()
	memCapacity := node.Status.Capacity.Memory().Value()

	cpuPercent := (float64(cpuUsage) / float64(cpuCapacity)) * 100
	memPercent := (float64(memUsage) / float64(memCapacity)) * 100

	return &NodeMetrics{
		NodeName:            nodeName,
		CPUUsageMillicores:  cpuUsage,
		CPUUsagePercent:     cpuPercent,
		MemoryUsageBytes:    memUsage,
		MemoryUsagePercent:  memPercent,
		CPUCapacityMillis:   cpuCapacity,
		MemoryCapacityBytes: memCapacity,
	}, nil
}

// GetAllPodsMetrics retrieves metrics for all pods in all namespaces
func (mc *MetricsClient) GetAllPodsMetrics(ctx context.Context) ([]*PodMetrics, error) {
	podMetricsList, err := mc.metricsClientset.MetricsV1beta1().PodMetricses("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list pod metrics: %w", err)
	}

	var allMetrics []*PodMetrics

	for _, podMetrics := range podMetricsList.Items {
		metrics, err := mc.GetPodMetrics(ctx, podMetrics.Namespace, podMetrics.Name)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("namespace", podMetrics.Namespace).
				Str("pod", podMetrics.Name).
				Msg("Failed to get pod metrics")
			continue
		}
		allMetrics = append(allMetrics, metrics)
	}

	return allMetrics, nil
}

// GetAllNodesMetrics retrieves metrics for all nodes
func (mc *MetricsClient) GetAllNodesMetrics(ctx context.Context) ([]*NodeMetrics, error) {
	nodeMetricsList, err := mc.metricsClientset.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list node metrics: %w", err)
	}

	var allMetrics []*NodeMetrics

	for _, nodeMetrics := range nodeMetricsList.Items {
		metrics, err := mc.GetNodeMetrics(ctx, nodeMetrics.Name)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("node", nodeMetrics.Name).
				Msg("Failed to get node metrics")
			continue
		}
		allMetrics = append(allMetrics, metrics)
	}

	return allMetrics, nil
}
