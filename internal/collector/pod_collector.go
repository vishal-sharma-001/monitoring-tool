package collector

import (
	"context"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/monitoring-engine/monitoring-tool/internal/config"
	"github.com/monitoring-engine/monitoring-tool/internal/processor"
	"github.com/monitoring-engine/monitoring-tool/internal/models"
	"github.com/monitoring-engine/monitoring-tool/internal/logger"
	"github.com/monitoring-engine/monitoring-tool/internal/pool"
)

// PodEvent represents a pod event
type PodEvent struct {
	Type      watch.EventType
	Pod       *corev1.Pod
	Timestamp time.Time
}

// PodWatcher watches pod events and processes them with goroutines
type PodWatcher struct {
	client              *K8sClient
	eventChan           chan *PodEvent // Buffered channel
	stateManager        *processor.AlertStateManager
	workerPool          *pool.WorkerPool
	stopCh              chan struct{}
	wg                  sync.WaitGroup
	restartThreshold    int32
	pendingTimeout      time.Duration
}

// NewPodWatcher creates a new pod watcher
func NewPodWatcher(k8sClient *K8sClient, stateManager *processor.AlertStateManager, workerPool *pool.WorkerPool) *PodWatcher {
	cfg := config.Get()
	return &PodWatcher{
		client:           k8sClient,
		eventChan:        make(chan *PodEvent, 500), // Buffered
		stateManager:     stateManager,
		workerPool:       workerPool,
		stopCh:           make(chan struct{}),
		restartThreshold: int32(cfg.AlertRules.PodRestartThreshold),
		pendingTimeout:   5 * time.Minute, // Default 5 minutes, can be made configurable
	}
}

// Start begins watching pods using worker pool
func (pw *PodWatcher) Start(ctx context.Context) {
	logger.Info().Msg("Starting Pod Watcher with worker pool")

	// Start event dispatcher that submits to worker pool
	pw.wg.Add(1)
	go pw.eventDispatcher(ctx)

	// Start real K8s pod watcher
	pw.wg.Add(1)
	go pw.watchPods(ctx)
}

// watchPods watches for pod events from Kubernetes API
func (pw *PodWatcher) watchPods(ctx context.Context) {
	defer pw.wg.Done()

	clientset := pw.client.GetClientset()

	for {
		select {
		case <-pw.stopCh:
			return
		case <-ctx.Done():
			return
		default:
		}

		// Watch all pods in all namespaces
		watcher, err := clientset.CoreV1().Pods("").Watch(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Error().Err(err).Msg("Failed to create pod watcher, retrying in 5s")
			time.Sleep(5 * time.Second)
			continue
		}

		logger.Info().Msg("Pod watcher connected to Kubernetes API")

		// Process watch events
		func() {
			defer watcher.Stop()

			for {
				select {
				case event, ok := <-watcher.ResultChan():
					if !ok {
						logger.Warn().Msg("Pod watch channel closed, reconnecting...")
						return
					}

					pod, ok := event.Object.(*corev1.Pod)
					if !ok {
						logger.Warn().Msg("Received non-pod object from watch")
						continue
					}

					podEvent := &PodEvent{
						Type:      event.Type,
						Pod:       pod,
						Timestamp: time.Now(),
					}

					select {
					case pw.eventChan <- podEvent:
						logger.Debug().
							Str("type", string(event.Type)).
							Str("pod", pod.Name).
							Str("namespace", pod.Namespace).
							Msg("Received pod event")
					default:
						logger.Warn().Msg("Pod event channel full, dropping event")
					}

				case <-pw.stopCh:
					return
				case <-ctx.Done():
					return
				}
			}
		}()
	}
}

// eventDispatcher reads events and submits them to worker pool
func (pw *PodWatcher) eventDispatcher(ctx context.Context) {
	defer pw.wg.Done()

	for {
		select {
		case event := <-pw.eventChan:
			// Submit event processing to worker pool
			eventCopy := event // Capture for closure
			if err := pw.workerPool.SubmitWithContext(ctx, func(ctx context.Context) error {
				return pw.processEvent(ctx, eventCopy)
			}); err != nil {
				logger.Warn().Err(err).
					Str("pod", event.Pod.Name).
					Str("namespace", event.Pod.Namespace).
					Msg("Failed to submit pod event to worker pool (queue full)")
			}

		case <-pw.stopCh:
			logger.Info().Msg("Pod event dispatcher stopped")
			return

		case <-ctx.Done():
			return
		}
	}
}

// processEvent handles a single pod event with detailed alert categorization
func (pw *PodWatcher) processEvent(ctx context.Context, event *PodEvent) error {
	pod := event.Pod

	logger.Debug().
		Str("type", string(event.Type)).
		Str("pod", pod.Name).
		Str("namespace", pod.Namespace).
		Str("phase", string(pod.Status.Phase)).
		Msg("Processing pod event")

	// Check for different types of critical conditions
	alerts := pw.evaluatePodConditions(pod)

	// Process each alert through the state manager
	for _, alert := range alerts {
		created, err := pw.stateManager.ProcessAlert(ctx, alert)
		if err != nil {
			logger.Error().Err(err).
				Str("pod", pod.Name).
				Str("alert_type", alert.GetLabelsMap()["alert_type"]).
				Msg("Failed to process alert")
			continue
		}

		if created {
			logger.Warn().
				Str("pod", pod.Name).
				Str("namespace", pod.Namespace).
				Str("severity", alert.Severity).
				Str("message", alert.Message).
				Msg("New pod alert created")
		}
	}

	return nil
}

// evaluatePodConditions checks pod for various critical conditions and returns alerts
func (pw *PodWatcher) evaluatePodConditions(pod *corev1.Pod) []*models.Alert {
	var alerts []*models.Alert

	// 1. Check for pod failure
	if pod.Status.Phase == corev1.PodFailed {
		alert := BuildPodAlert(pod, AlertTypePodFailed, 1.0)
		alerts = append(alerts, alert)
	}

	// 2. Check for unknown state
	if pod.Status.Phase == corev1.PodUnknown {
		alert := BuildPodAlert(pod, AlertTypePodUnknown, 1.0)
		alerts = append(alerts, alert)
	}

	// 3. Check container statuses for various issues
	var totalRestarts int32
	for _, cs := range pod.Status.ContainerStatuses {
		totalRestarts += cs.RestartCount

		// Check for OOMKilled
		if cs.LastTerminationState.Terminated != nil && cs.LastTerminationState.Terminated.Reason == "OOMKilled" {
			alert := BuildPodAlert(pod, AlertTypePodOOMKilled, float64(cs.RestartCount))
			alerts = append(alerts, alert)
		}

		// Check for CrashLoopBackOff
		if cs.State.Waiting != nil && cs.State.Waiting.Reason == "CrashLoopBackOff" {
			alert := BuildPodAlert(pod, AlertTypePodCrashLoop, float64(cs.RestartCount))
			alerts = append(alerts, alert)
		}

		// Check for Image Pull errors
		if cs.State.Waiting != nil &&
			(cs.State.Waiting.Reason == "ImagePullBackOff" || cs.State.Waiting.Reason == "ErrImagePull") {
			alert := BuildPodAlert(pod, AlertTypePodImagePullError, 1.0)
			alerts = append(alerts, alert)
		}
	}

	// 4. Check for excessive restarts (using configured threshold)
	if totalRestarts > pw.restartThreshold {
		alert := BuildPodAlert(pod, AlertTypePodRestartThreshold, float64(totalRestarts))
		alerts = append(alerts, alert)
	}

	// 5. Check for pending state (could indicate scheduling issues)
	if pod.Status.Phase == corev1.PodPending && time.Since(pod.CreationTimestamp.Time) > pw.pendingTimeout {
		alert := BuildPodAlert(pod, AlertTypePodPending, 1.0)
		alerts = append(alerts, alert)
	}

	return alerts
}

// Stop gracefully stops the pod watcher
func (pw *PodWatcher) Stop() {
	close(pw.stopCh)
	pw.wg.Wait()
}
