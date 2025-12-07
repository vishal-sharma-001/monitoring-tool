package collector_test

import (
	"testing"

	"github.com/monitoring-engine/monitoring-tool/internal/collector"
	"github.com/monitoring-engine/monitoring-tool/internal/models"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestBuildPodAlert(t *testing.T) {
	t.Run("should build pod failed alert", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "test-namespace",
			},
			Status: corev1.PodStatus{
				Phase:   corev1.PodFailed,
				Reason:  "Error",
				Message: "Container failed",
			},
		}

		alert := collector.BuildPodAlert(pod, collector.AlertTypePodFailed, 1.0)

		assert.NotNil(t, alert)
		assert.Equal(t, "critical", alert.Severity)
		assert.Contains(t, alert.Message, "test-pod")
		assert.Contains(t, alert.Message, "FAILED")
		assert.Equal(t, "k8s_pod", alert.Source)
		assert.Equal(t, 1.0, alert.Value)
	})

	t.Run("should build pod OOM killed alert", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "oom-pod",
				Namespace: "production",
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name: "app-container",
						LastTerminationState: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								Reason: "OOMKilled",
							},
						},
					},
				},
			},
		}

		alert := collector.BuildPodAlert(pod, collector.AlertTypePodOOMKilled, 1.0)

		assert.NotNil(t, alert)
		assert.Equal(t, "critical", alert.Severity)
		assert.Contains(t, alert.Message, "OOM KILLED")
		assert.Contains(t, alert.Message, "app-container")
	})

	t.Run("should build pod crash loop alert", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "crash-pod",
				Namespace: "staging",
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name: "main-container",
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{
								Reason:  "CrashLoopBackOff",
								Message: "Container exited with code 1",
							},
						},
					},
				},
			},
		}

		alert := collector.BuildPodAlert(pod, collector.AlertTypePodCrashLoop, 5.0)

		assert.NotNil(t, alert)
		assert.Equal(t, "high", alert.Severity)
		assert.Contains(t, alert.Message, "CRASH LOOP")
		assert.Contains(t, alert.Message, "main-container")
	})

	t.Run("should build pod restart threshold alert", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "restart-pod",
				Namespace: "default",
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name:         "container-1",
						RestartCount: 10,
					},
					{
						Name:         "container-2",
						RestartCount: 5,
					},
				},
			},
		}

		alert := collector.BuildPodAlert(pod, collector.AlertTypePodRestartThreshold, 10.0)

		assert.NotNil(t, alert)
		assert.Equal(t, "high", alert.Severity)
		assert.Contains(t, alert.Message, "EXCESSIVE RESTARTS")
		assert.Contains(t, alert.Message, "container-1") // Highest restart count
	})

	t.Run("should build pod image pull error alert", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "image-pod",
				Namespace: "dev",
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name: "app",
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{
								Reason:  "ImagePullBackOff",
								Message: "Failed to pull image",
							},
						},
					},
				},
			},
		}

		alert := collector.BuildPodAlert(pod, collector.AlertTypePodImagePullError, 1.0)

		assert.NotNil(t, alert)
		assert.Equal(t, "high", alert.Severity)
		assert.Contains(t, alert.Message, "cannot pull image")
	})

	t.Run("should build pod pending alert", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "pending-pod",
				Namespace: "test",
			},
			Status: corev1.PodStatus{
				Phase:   corev1.PodPending,
				Reason:  "Unschedulable",
				Message: "Insufficient CPU",
			},
		}

		alert := collector.BuildPodAlert(pod, collector.AlertTypePodPending, 1.0)

		assert.NotNil(t, alert)
		assert.Equal(t, "medium", alert.Severity)
		assert.Contains(t, alert.Message, "PENDING")
		assert.Contains(t, alert.Message, "Unschedulable")
	})

	t.Run("should build pod unknown alert", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "unknown-pod",
				Namespace: "test",
			},
			Status: corev1.PodStatus{
				Phase: corev1.PodUnknown,
			},
		}

		alert := collector.BuildPodAlert(pod, collector.AlertTypePodUnknown, 1.0)

		assert.NotNil(t, alert)
		assert.Equal(t, "critical", alert.Severity)
		assert.Contains(t, alert.Message, "UNKNOWN state")
	})

	t.Run("should handle pod with no container statuses", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "empty-pod",
				Namespace: "test",
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{},
			},
		}

		alert := collector.BuildPodAlert(pod, collector.AlertTypePodOOMKilled, 1.0)

		assert.NotNil(t, alert)
		// Should handle gracefully with "unknown" container
	})
}

func TestBuildNodeAlert(t *testing.T) {
	t.Run("should build node not ready alert", func(t *testing.T) {
		node := &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "worker-node-1",
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{
						Type:   corev1.NodeReady,
						Status: corev1.ConditionFalse,
						Reason: "KubeletNotReady",
					},
				},
			},
		}

		alert := collector.BuildNodeAlert(node, collector.AlertTypeNodeNotReady, 1.0)

		assert.NotNil(t, alert)
		assert.Equal(t, "critical", alert.Severity)
		assert.Contains(t, alert.Message, "NOT READY")
		assert.Contains(t, alert.Message, "worker-node-1")
		assert.Equal(t, "k8s_node", alert.Source)
	})

	t.Run("should build node memory pressure alert", func(t *testing.T) {
		node := &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-2",
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{
						Type:   corev1.NodeMemoryPressure,
						Status: corev1.ConditionTrue,
					},
				},
			},
		}

		alert := collector.BuildNodeAlert(node, collector.AlertTypeNodeMemoryPressure, 1.0)

		assert.NotNil(t, alert)
		assert.Equal(t, "high", alert.Severity)
		assert.Contains(t, alert.Message, "MEMORY PRESSURE")
	})

	t.Run("should build node disk pressure alert", func(t *testing.T) {
		node := &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-3",
			},
		}

		alert := collector.BuildNodeAlert(node, collector.AlertTypeNodeDiskPressure, 1.0)

		assert.NotNil(t, alert)
		assert.Equal(t, "high", alert.Severity)
		assert.Contains(t, alert.Message, "DISK PRESSURE")
	})

	t.Run("should build node PID pressure alert", func(t *testing.T) {
		node := &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "node-4",
			},
		}

		alert := collector.BuildNodeAlert(node, collector.AlertTypeNodePIDPressure, 1.0)

		assert.NotNil(t, alert)
		assert.Equal(t, "medium", alert.Severity)
		assert.Contains(t, alert.Message, "PID PRESSURE")
	})
}

func TestBuildPodMetricAlert(t *testing.T) {
	t.Run("should build pod CPU high alert", func(t *testing.T) {
		alert := collector.BuildPodMetricAlert("production", "api-pod", collector.AlertTypePodCPUHigh, 85.5, 80.0)

		assert.NotNil(t, alert)
		assert.Equal(t, "high", alert.Severity)
		assert.Contains(t, alert.Message, "CPU usage is HIGH")
		assert.Contains(t, alert.Message, "85.5")
		assert.Contains(t, alert.Message, "80.0")
		assert.Equal(t, "k8s_pod_metrics", alert.Source)
		assert.Equal(t, 85.5, alert.Value)
	})

	t.Run("should build pod memory high alert", func(t *testing.T) {
		alert := collector.BuildPodMetricAlert("staging", "db-pod", collector.AlertTypePodMemoryHigh, 92.3, 85.0)

		assert.NotNil(t, alert)
		assert.Equal(t, "high", alert.Severity)
		assert.Contains(t, alert.Message, "Memory usage is HIGH")
		assert.Contains(t, alert.Message, "92.3")
		assert.Contains(t, alert.Message, "85.0")
	})

	t.Run("should handle unknown pod metric alert type", func(t *testing.T) {
		alert := collector.BuildPodMetricAlert("test", "pod-1", "unknown", 50.0, 60.0)

		assert.NotNil(t, alert)
		assert.Equal(t, "medium", alert.Severity)
	})
}

func TestBuildNodeMetricAlert(t *testing.T) {
	t.Run("should build node CPU high alert", func(t *testing.T) {
		alert := collector.BuildNodeMetricAlert("master-node", collector.AlertTypeNodeCPUHigh, 88.7, 70.0)

		assert.NotNil(t, alert)
		assert.Equal(t, "critical", alert.Severity)
		assert.Contains(t, alert.Message, "CPU usage is CRITICAL")
		assert.Contains(t, alert.Message, "88.7")
		assert.Contains(t, alert.Message, "70.0")
		assert.Equal(t, "k8s_node_metrics", alert.Source)
		assert.Equal(t, 88.7, alert.Value)
	})

	t.Run("should build node memory high alert", func(t *testing.T) {
		alert := collector.BuildNodeMetricAlert("worker-node", collector.AlertTypeNodeMemoryHigh, 91.2, 75.0)

		assert.NotNil(t, alert)
		assert.Equal(t, "critical", alert.Severity)
		assert.Contains(t, alert.Message, "Memory usage is CRITICAL")
		assert.Contains(t, alert.Message, "91.2")
		assert.Contains(t, alert.Message, "75.0")
	})

	t.Run("should handle unknown node metric alert type", func(t *testing.T) {
		alert := collector.BuildNodeMetricAlert("node-1", "unknown", 50.0, 60.0)

		assert.NotNil(t, alert)
		assert.Equal(t, "medium", alert.Severity)
	})
}

func TestAlertLabels(t *testing.T) {
	t.Run("pod alert should include correct labels", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "test-ns",
			},
		}

		alert := collector.BuildPodAlert(pod, collector.AlertTypePodFailed, 1.0)

		labels := alert.GetLabelsMap()
		assert.Equal(t, "test-ns", labels["namespace"])
		assert.Equal(t, "test-pod", labels["pod"])
		assert.Equal(t, "pod_failed", labels["alert_type"])
	})

	t.Run("node alert should include correct labels", func(t *testing.T) {
		node := &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-node",
			},
		}

		alert := collector.BuildNodeAlert(node, collector.AlertTypeNodeNotReady, 1.0)

		labels := alert.GetLabelsMap()
		assert.Equal(t, "test-node", labels["node"])
		assert.Equal(t, "node_not_ready", labels["alert_type"])
	})
}

func TestAlertSeverityMapping(t *testing.T) {
	tests := []struct {
		name         string
		alertType    collector.AlertType
		wantSeverity string
	}{
		{"pod failed is critical", collector.AlertTypePodFailed, "critical"},
		{"pod OOM is critical", collector.AlertTypePodOOMKilled, "critical"},
		{"pod crash loop is high", collector.AlertTypePodCrashLoop, "high"},
		{"pod restart threshold is high", collector.AlertTypePodRestartThreshold, "high"},
		{"pod pending is medium", collector.AlertTypePodPending, "medium"},
		{"node not ready is critical", collector.AlertTypeNodeNotReady, "critical"},
		{"node memory pressure is high", collector.AlertTypeNodeMemoryPressure, "high"},
		{"node disk pressure is high", collector.AlertTypeNodeDiskPressure, "high"},
		{"node PID pressure is medium", collector.AlertTypeNodePIDPressure, "medium"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var alert *models.Alert

			// Create appropriate test object based on alert type
			if tt.alertType == collector.AlertTypeNodeNotReady ||
				tt.alertType == collector.AlertTypeNodeMemoryPressure ||
				tt.alertType == collector.AlertTypeNodeDiskPressure ||
				tt.alertType == collector.AlertTypeNodePIDPressure {
				node := &corev1.Node{
					ObjectMeta: metav1.ObjectMeta{Name: "test-node"},
				}
				alert = collector.BuildNodeAlert(node, tt.alertType, 1.0)
			} else {
				pod := &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "test-ns",
					},
				}
				alert = collector.BuildPodAlert(pod, tt.alertType, 1.0)
			}

			assert.Equal(t, tt.wantSeverity, alert.Severity)
		})
	}
}

func TestBuildPodAlert_EdgeCases(t *testing.T) {
	t.Run("should handle pod with multiple image pull errors", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "multi-error-pod",
				Namespace: "test",
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name: "container-1",
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{
								Reason:  "ErrImagePull",
								Message: "Failed to pull image",
							},
						},
					},
					{
						Name: "container-2",
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{},
						},
					},
				},
			},
		}

		alert := collector.BuildPodAlert(pod, collector.AlertTypePodImagePullError, 1.0)

		assert.NotNil(t, alert)
		assert.Equal(t, "high", alert.Severity)
		assert.Contains(t, alert.Message, "container-1")
	})

	t.Run("should handle pod with no waiting reason", func(t *testing.T) {
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "no-reason-pod",
				Namespace: "test",
			},
			Status: corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{
					{
						Name: "container",
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{},
						},
					},
				},
			},
		}

		alert := collector.BuildPodAlert(pod, collector.AlertTypePodImagePullError, 1.0)

		assert.NotNil(t, alert)
		assert.Contains(t, alert.Message, "unknown")
	})

	t.Run("should handle node with no matching condition", func(t *testing.T) {
		node := &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-node",
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{},
			},
		}

		alert := collector.BuildNodeAlert(node, collector.AlertTypeNodeMemoryPressure, 1.0)

		assert.NotNil(t, alert)
		assert.Equal(t, "high", alert.Severity)
	})

	t.Run("should handle node condition with empty reason", func(t *testing.T) {
		node := &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-node",
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{
						Type:   corev1.NodeReady,
						Status: corev1.ConditionFalse,
						Reason: "",
					},
				},
			},
		}

		alert := collector.BuildNodeAlert(node, collector.AlertTypeNodeNotReady, 1.0)

		assert.NotNil(t, alert)
		assert.Contains(t, alert.Message, "False")
	})
}

func TestBuildNodeAlert_EdgeCases(t *testing.T) {
	t.Run("should handle unknown alert type", func(t *testing.T) {
		node := &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-node",
			},
		}

		alert := collector.BuildNodeAlert(node, "unknown_type", 1.0)

		assert.NotNil(t, alert)
		assert.Equal(t, "medium", alert.Severity)
	})

	t.Run("should handle node with multiple conditions", func(t *testing.T) {
		node := &corev1.Node{
			ObjectMeta: metav1.ObjectMeta{
				Name: "multi-condition-node",
			},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{
						Type:   corev1.NodeReady,
						Status: corev1.ConditionTrue,
						Reason: "KubeletReady",
					},
					{
						Type:   corev1.NodeMemoryPressure,
						Status: corev1.ConditionTrue,
						Reason: "MemoryPressure",
					},
				},
			},
		}

		alert := collector.BuildNodeAlert(node, collector.AlertTypeNodeMemoryPressure, 1.0)

		assert.NotNil(t, alert)
		assert.Equal(t, "high", alert.Severity)
		assert.Contains(t, alert.Message, "MEMORY PRESSURE")
	})
}
