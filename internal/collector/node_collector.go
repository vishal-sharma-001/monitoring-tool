package collector

import (
	"context"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/monitoring-engine/monitoring-tool/internal/processor"
	"github.com/monitoring-engine/monitoring-tool/internal/models"
	"github.com/monitoring-engine/monitoring-tool/internal/logger"
	"github.com/monitoring-engine/monitoring-tool/internal/pool"
)

// NodeEvent represents a node event
type NodeEvent struct {
	Type      watch.EventType
	Node      *corev1.Node
	Timestamp time.Time
}

// NodeWatcher watches node events using worker pool
type NodeWatcher struct {
	client       *K8sClient
	eventChan    chan *NodeEvent
	stateManager *processor.AlertStateManager
	workerPool   *pool.WorkerPool
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

func NewNodeWatcher(k8sClient *K8sClient, stateManager *processor.AlertStateManager, workerPool *pool.WorkerPool) *NodeWatcher {
	return &NodeWatcher{
		client:       k8sClient,
		eventChan:    make(chan *NodeEvent, 300),
		stateManager: stateManager,
		workerPool:   workerPool,
		stopCh:       make(chan struct{}),
	}
}

func (nw *NodeWatcher) Start(ctx context.Context) {
	logger.Info().Msg("Starting Node Watcher with worker pool")

	// Start event dispatcher that submits to worker pool
	nw.wg.Add(1)
	go nw.eventDispatcher(ctx)

	// Start real K8s node watcher
	nw.wg.Add(1)
	go nw.watchNodes(ctx)
}

// watchNodes watches for node events from Kubernetes API
func (nw *NodeWatcher) watchNodes(ctx context.Context) {
	defer nw.wg.Done()

	clientset := nw.client.GetClientset()

	for {
		select {
		case <-nw.stopCh:
			return
		case <-ctx.Done():
			return
		default:
		}

		// Watch all nodes
		watcher, err := clientset.CoreV1().Nodes().Watch(ctx, metav1.ListOptions{})
		if err != nil {
			logger.Error().Err(err).Msg("Failed to create node watcher, retrying in 5s")
			time.Sleep(5 * time.Second)
			continue
		}

		logger.Info().Msg("Node watcher connected to Kubernetes API")

		// Process watch events
		func() {
			defer watcher.Stop()

			for {
				select {
				case event, ok := <-watcher.ResultChan():
					if !ok {
						logger.Warn().Msg("Node watch channel closed, reconnecting...")
						return
					}

					node, ok := event.Object.(*corev1.Node)
					if !ok {
						logger.Warn().Msg("Received non-node object from watch")
						continue
					}

					nodeEvent := &NodeEvent{
						Type:      event.Type,
						Node:      node,
						Timestamp: time.Now(),
					}

					select {
					case nw.eventChan <- nodeEvent:
						logger.Debug().
							Str("type", string(event.Type)).
							Str("node", node.Name).
							Msg("Received node event")
					default:
						logger.Warn().Msg("Node event channel full, dropping event")
					}

				case <-nw.stopCh:
					return
				case <-ctx.Done():
					return
				}
			}
		}()
	}
}

// eventDispatcher reads events and submits them to worker pool
func (nw *NodeWatcher) eventDispatcher(ctx context.Context) {
	defer nw.wg.Done()

	for {
		select {
		case event := <-nw.eventChan:
			// Submit event processing to worker pool
			eventCopy := event // Capture for closure
			if err := nw.workerPool.SubmitWithContext(ctx, func(ctx context.Context) error {
				return nw.processEvent(ctx, eventCopy)
			}); err != nil {
				logger.Warn().Err(err).
					Str("node", event.Node.Name).
					Msg("Failed to submit node event to worker pool (queue full)")
			}

		case <-nw.stopCh:
			logger.Info().Msg("Node event dispatcher stopped")
			return

		case <-ctx.Done():
			return
		}
	}
}

// processEvent handles a single node event with detailed alert categorization
func (nw *NodeWatcher) processEvent(ctx context.Context, event *NodeEvent) error {
	node := event.Node

	logger.Debug().
		Str("type", string(event.Type)).
		Str("node", node.Name).
		Msg("Processing node event")

	// Check for different types of critical conditions
	alerts := nw.evaluateNodeConditions(node)

	// Process each alert through the state manager
	for _, alert := range alerts {
		created, err := nw.stateManager.ProcessAlert(ctx, alert)
		if err != nil {
			logger.Error().Err(err).
				Str("node", node.Name).
				Str("alert_type", alert.GetLabelsMap()["alert_type"]).
				Msg("Failed to process alert")
			continue
		}

		if created {
			logger.Warn().
				Str("node", node.Name).
				Str("severity", alert.Severity).
				Str("message", alert.Message).
				Msg("New node alert created")
		}
	}

	return nil
}

// evaluateNodeConditions checks node for various critical conditions and returns alerts
func (nw *NodeWatcher) evaluateNodeConditions(node *corev1.Node) []*models.Alert {
	var alerts []*models.Alert

	for _, condition := range node.Status.Conditions {
		switch condition.Type {
		case corev1.NodeReady:
			// Node is NOT ready
			if condition.Status != corev1.ConditionTrue {
				alert := BuildNodeAlert(node, AlertTypeNodeNotReady, 1.0)
				alerts = append(alerts, alert)
			}

		case corev1.NodeMemoryPressure:
			// Node has memory pressure
			if condition.Status == corev1.ConditionTrue {
				alert := BuildNodeAlert(node, AlertTypeNodeMemoryPressure, 1.0)
				alerts = append(alerts, alert)
			}

		case corev1.NodeDiskPressure:
			// Node has disk pressure
			if condition.Status == corev1.ConditionTrue {
				alert := BuildNodeAlert(node, AlertTypeNodeDiskPressure, 1.0)
				alerts = append(alerts, alert)
			}

		case corev1.NodePIDPressure:
			// Node has PID pressure
			if condition.Status == corev1.ConditionTrue {
				alert := BuildNodeAlert(node, AlertTypeNodePIDPressure, 1.0)
				alerts = append(alerts, alert)
			}
		}
	}

	return alerts
}

func (nw *NodeWatcher) Stop() {
	close(nw.stopCh)
	nw.wg.Wait()
}
