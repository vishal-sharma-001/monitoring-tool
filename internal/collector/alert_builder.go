package collector

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/monitoring-engine/monitoring-tool/internal/models"
)

// AlertType represents different categories of alerts
type AlertType string

const (
	// Event-based alerts
	AlertTypePodFailed           AlertType = "pod_failed"
	AlertTypePodOOMKilled        AlertType = "pod_oom_killed"
	AlertTypePodCrashLoop        AlertType = "pod_crash_loop"
	AlertTypePodRestartThreshold AlertType = "pod_restart_threshold"
	AlertTypePodImagePullError   AlertType = "pod_image_pull"
	AlertTypePodPending          AlertType = "pod_pending"
	AlertTypePodUnknown          AlertType = "pod_unknown"
	AlertTypeNodeNotReady        AlertType = "node_not_ready"
	AlertTypeNodeMemoryPressure  AlertType = "node_memory_pressure"
	AlertTypeNodeDiskPressure    AlertType = "node_disk_pressure"
	AlertTypeNodePIDPressure     AlertType = "node_pid_pressure"

	// Metric-based alerts
	AlertTypePodCPUHigh     AlertType = "pod_cpu_high"
	AlertTypePodMemoryHigh  AlertType = "pod_memory_high"
	AlertTypeNodeCPUHigh    AlertType = "node_cpu_high"
	AlertTypeNodeMemoryHigh AlertType = "node_memory_high"
)

// BuildPodAlert creates a detailed alert for pod issues
func BuildPodAlert(pod *corev1.Pod, alertType AlertType, value float64) *models.Alert {
	var severity string
	var message string

	labels := map[string]string{
		"namespace":  pod.Namespace,
		"pod":        pod.Name,
		"alert_type": string(alertType),
	}

	switch alertType {
	case AlertTypePodFailed:
		severity = SeverityCritical
		message = fmt.Sprintf("Pod %s/%s has FAILED - Phase: %s, Reason: %s",
			pod.Namespace, pod.Name, pod.Status.Phase, pod.Status.Reason)
		if pod.Status.Message != "" {
			message += fmt.Sprintf(", Message: %s", pod.Status.Message)
		}

	case AlertTypePodOOMKilled:
		severity = SeverityCritical
		containerName := getOOMKilledContainer(pod)
		message = fmt.Sprintf("Pod %s/%s container '%s' was OOM KILLED - Out of memory",
			pod.Namespace, pod.Name, containerName)
		labels["container"] = containerName

	case AlertTypePodCrashLoop:
		severity = SeverityHigh
		containerName, reason := getCrashLoopContainer(pod)
		message = fmt.Sprintf("Pod %s/%s container '%s' is in CRASH LOOP BACKOFF - Reason: %s",
			pod.Namespace, pod.Name, containerName, reason)
		labels["container"] = containerName
		labels["reason"] = reason

	case AlertTypePodRestartThreshold:
		severity = SeverityHigh
		restartCount := int32(value)
		containerName := getHighestRestartContainer(pod)
		message = fmt.Sprintf("Pod %s/%s has EXCESSIVE RESTARTS - Total restarts: %d, Container: %s",
			pod.Namespace, pod.Name, restartCount, containerName)
		labels["container"] = containerName

	case AlertTypePodImagePullError:
		severity = SeverityHigh
		containerName, imageError := getImagePullError(pod)
		message = fmt.Sprintf("Pod %s/%s container '%s' cannot pull image - Error: %s",
			pod.Namespace, pod.Name, containerName, imageError)
		labels["container"] = containerName

	case AlertTypePodPending:
		severity = SeverityMedium
		message = fmt.Sprintf("Pod %s/%s is PENDING for extended period - Reason: %s",
			pod.Namespace, pod.Name, pod.Status.Reason)
		if pod.Status.Message != "" {
			message += fmt.Sprintf(", Details: %s", pod.Status.Message)
		}

	case AlertTypePodUnknown:
		severity = SeverityCritical
		message = fmt.Sprintf("Pod %s/%s is in UNKNOWN state - Last known phase: %s",
			pod.Namespace, pod.Name, pod.Status.Phase)

	default:
		severity = SeverityMedium
		message = fmt.Sprintf("Pod %s/%s issue detected - Type: %s",
			pod.Namespace, pod.Name, alertType)
	}

	return models.NewAlert(severity, message, "k8s_pod", value, labels)
}

// BuildNodeAlert creates a detailed alert for node issues
func BuildNodeAlert(node *corev1.Node, alertType AlertType, value float64) *models.Alert {
	var severity string
	var message string

	labels := map[string]string{
		"node":       node.Name,
		"alert_type": string(alertType),
	}

	switch alertType {
	case AlertTypeNodeNotReady:
		severity = SeverityCritical
		message = fmt.Sprintf("Node %s is NOT READY - Status: %s",
			node.Name, getNodeConditionReason(node, corev1.NodeReady))

	case AlertTypeNodeMemoryPressure:
		severity = SeverityHigh
		message = fmt.Sprintf("Node %s has MEMORY PRESSURE - Available memory is low",
			node.Name)

	case AlertTypeNodeDiskPressure:
		severity = SeverityHigh
		message = fmt.Sprintf("Node %s has DISK PRESSURE - Disk space is running low",
			node.Name)

	case AlertTypeNodePIDPressure:
		severity = SeverityMedium
		message = fmt.Sprintf("Node %s has PID PRESSURE - Too many processes running",
			node.Name)

	default:
		severity = SeverityMedium
		message = fmt.Sprintf("Node %s issue detected - Type: %s",
			node.Name, alertType)
	}

	return models.NewAlert(severity, message, "k8s_node", value, labels)
}

// Helper functions to extract container-specific information

func getOOMKilledContainer(pod *corev1.Pod) string {
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.LastTerminationState.Terminated != nil && cs.LastTerminationState.Terminated.Reason == "OOMKilled" {
			return cs.Name
		}
	}
	return "unknown"
}

func getCrashLoopContainer(pod *corev1.Pod) (string, string) {
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason == "CrashLoopBackOff" {
			return cs.Name, cs.State.Waiting.Message
		}
	}
	return "unknown", "unknown"
}

func getHighestRestartContainer(pod *corev1.Pod) string {
	maxRestarts := int32(0)
	containerName := "unknown"
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.RestartCount > maxRestarts {
			maxRestarts = cs.RestartCount
			containerName = cs.Name
		}
	}
	return containerName
}

func getImagePullError(pod *corev1.Pod) (string, string) {
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil &&
			(cs.State.Waiting.Reason == "ImagePullBackOff" || cs.State.Waiting.Reason == "ErrImagePull") {
			return cs.Name, cs.State.Waiting.Message
		}
	}
	return "unknown", "unknown"
}

func getNodeConditionReason(node *corev1.Node, conditionType corev1.NodeConditionType) string {
	for _, condition := range node.Status.Conditions {
		if condition.Type == conditionType {
			if condition.Reason != "" {
				return condition.Reason
			}
			return string(condition.Status)
		}
	}
	return "Unknown"
}
// BuildPodMetricAlert creates an alert for pod metric threshold violations
func BuildPodMetricAlert(namespace, podName string, alertType AlertType, value float64, threshold float64) *models.Alert {
	var severity string
	var message string
	

	labels := map[string]string{
		"namespace":  namespace,
		"pod":        podName,
		"alert_type": string(alertType),
		"metric":     "",
	}

	switch alertType {
	case AlertTypePodCPUHigh:
		severity = SeverityHigh
		
		labels["metric"] = "cpu"
		message = fmt.Sprintf("Pod %s/%s CPU usage is HIGH: %.1f%% (threshold: %.1f%%)",
			namespace, podName, value, threshold)

	case AlertTypePodMemoryHigh:
		severity = SeverityHigh
		
		labels["metric"] = "memory"
		message = fmt.Sprintf("Pod %s/%s Memory usage is HIGH: %.1f%% (threshold: %.1f%%)",
			namespace, podName, value, threshold)

	default:
		severity = SeverityMedium
		message = fmt.Sprintf("Pod %s/%s metric alert", namespace, podName)
	}

	return models.NewAlert(severity, message, "k8s_pod_metrics", value, labels)
}

// BuildNodeMetricAlert creates an alert for node metric threshold violations
func BuildNodeMetricAlert(nodeName string, alertType AlertType, value float64, threshold float64) *models.Alert {
	var severity string
	var message string
	

	labels := map[string]string{
		"node":       nodeName,
		"alert_type": string(alertType),
		"metric":     "",
	}

	switch alertType {
	case AlertTypeNodeCPUHigh:
		severity = SeverityCritical
		
		labels["metric"] = "cpu"
		message = fmt.Sprintf("Node %s CPU usage is CRITICAL: %.1f%% (threshold: %.1f%%)",
			nodeName, value, threshold)

	case AlertTypeNodeMemoryHigh:
		severity = SeverityCritical
		
		labels["metric"] = "memory"
		message = fmt.Sprintf("Node %s Memory usage is CRITICAL: %.1f%% (threshold: %.1f%%)",
			nodeName, value, threshold)

	default:
		severity = SeverityMedium
		message = fmt.Sprintf("Node %s metric alert", nodeName)
	}

	return models.NewAlert(severity, message, "k8s_node_metrics", value, labels)
}
