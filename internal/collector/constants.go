package collector

// Alert Severities
const (
	SeverityCritical = "critical"
	SeverityHigh     = "high"
	SeverityMedium   = "medium"
	SeverityLow      = "low"
)

// Alert States
const (
	AlertStateTriggered    = "triggered"
	AlertStateAcknowledged = "acknowledged"
	AlertStateResolved     = "resolved"
)

// Notification States
const (
	NotificationStatePending = "pending"
	NotificationStateSuccess = "success"
	NotificationStateFailed  = "failed"
)

// Notification Channel Types
const (
	ChannelTypeEmail = "email"
)

// Target Types
const (
	TargetTypeServer      = "server"
	TargetTypeApplication = "application"
	TargetTypeService     = "service"
	TargetTypeContainer   = "container"
	TargetTypeK8sCluster  = "k8s_cluster"
)

// Target Status
const (
	TargetStatusActive      = "active"
	TargetStatusInactive    = "inactive"
	TargetStatusMaintenance = "maintenance"
)

// WebSocket Message Types
const (
	WSMessageTypeMetric    = "metric"
	WSMessageTypeAlert     = "alert"
	WSMessageTypeK8sEvent  = "k8s_event"
	WSMessageTypeSubscribe = "subscribe"
	WSMessageTypePing      = "ping"
	WSMessageTypePong      = "pong"
)

// Kubernetes Resource Types
const (
	K8sResourceTypePod        = "Pod"
	K8sResourceTypeNode       = "Node"
	K8sResourceTypeDeployment = "Deployment"
	K8sResourceTypeService    = "Service"
	K8sResourceTypePVC        = "PersistentVolumeClaim"
	K8sResourceTypeNamespace  = "Namespace"
)

// Kubernetes Event Types
const (
	K8sEventTypeAdded   = "Added"
	K8sEventTypeUpdated = "Updated"
	K8sEventTypeDeleted = "Deleted"
)

// Metric Names
const (
	MetricK8sPodPhase              = "k8s.pod.phase"
	MetricK8sContainerCPUUsage     = "k8s.container.cpu.usage"
	MetricK8sContainerMemoryUsage  = "k8s.container.memory.usage"
	MetricK8sContainerRestarts     = "k8s.container.restarts"
	MetricK8sContainerReady        = "k8s.container.ready"
	MetricK8sNodeCondition         = "k8s.node.condition"
	MetricK8sNodeCPUCapacity       = "k8s.node.cpu.capacity"
	MetricK8sNodeCPUAllocatable    = "k8s.node.cpu.allocatable"
	MetricK8sNodeCPUUsage          = "k8s.node.cpu.usage"
	MetricK8sNodeMemoryCapacity    = "k8s.node.memory.capacity"
	MetricK8sNodeMemoryAllocatable = "k8s.node.memory.allocatable"
	MetricK8sNodeMemoryUsage       = "k8s.node.memory.usage"
	MetricK8sNodePodsCapacity      = "k8s.node.pods.capacity"
	MetricK8sNodePodsAllocatable   = "k8s.node.pods.allocatable"
)
