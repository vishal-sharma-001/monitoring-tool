package collector

import (
	"context"
	"sync"
	"time"

	"github.com/monitoring-engine/monitoring-tool/internal/config"
	"github.com/monitoring-engine/monitoring-tool/internal/processor"
	
	"github.com/monitoring-engine/monitoring-tool/internal/logger"
	"github.com/monitoring-engine/monitoring-tool/internal/pool"
)

// MetricsWatcher watches pod and node metrics and generates alerts
type MetricsWatcher struct {
	client        *K8sClient
	stateManager  *processor.AlertStateManager
	workerPool    *pool.WorkerPool
	interval      time.Duration
	thresholds    *config.AlertRulesConfig
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

// NewMetricsWatcher creates a new metrics watcher
func NewMetricsWatcher(
	client *K8sClient,
	stateManager *processor.AlertStateManager,
	workerPool *pool.WorkerPool,
) *MetricsWatcher {
	cfg := config.Get()
	interval := time.Duration(cfg.Kubernetes.MetricsInterval) * time.Second
	if interval == 0 {
		interval = 60 * time.Second
	}

	return &MetricsWatcher{
		client:       client,
		stateManager: stateManager,
		workerPool:   workerPool,
		interval:     interval,
		thresholds:   &cfg.AlertRules,
		stopCh:       make(chan struct{}),
	}
}

// Start begins metrics monitoring
func (mw *MetricsWatcher) Start(ctx context.Context) {
	logger.Info().
		Str("interval", mw.interval.String()).
		Msg("Starting Metrics Watcher")

	mw.wg.Add(1)
	go mw.metricsLoop(ctx)
}

// metricsLoop periodically checks metrics
func (mw *MetricsWatcher) metricsLoop(ctx context.Context) {
	defer mw.wg.Done()

	ticker := time.NewTicker(mw.interval)
	defer ticker.Stop()

	// Run immediately on start
	mw.checkAllMetrics(ctx)

	for {
		select {
		case <-ticker.C:
			mw.checkAllMetrics(ctx)
		case <-mw.stopCh:
			logger.Info().Msg("Metrics watcher stopped")
			return
		case <-ctx.Done():
			logger.Info().Msg("Metrics watcher context cancelled")
			return
		}
	}
}

// checkAllMetrics checks both pod and node metrics
func (mw *MetricsWatcher) checkAllMetrics(ctx context.Context) {
	logger.Info().Msg("Checking all metrics")

	// Check pod metrics
	if err := mw.workerPool.SubmitWithContext(ctx, func(ctx context.Context) error {
		return mw.checkPodMetrics(ctx)
	}); err != nil {
		logger.Warn().Err(err).Msg("Failed to submit pod metrics check (worker pool queue full)")
	}

	// Check node metrics
	if err := mw.workerPool.SubmitWithContext(ctx, func(ctx context.Context) error {
		return mw.checkNodeMetrics(ctx)
	}); err != nil {
		logger.Warn().Err(err).Msg("Failed to submit node metrics check (worker pool queue full)")
	}
}

// checkPodMetrics checks all pod metrics
func (mw *MetricsWatcher) checkPodMetrics(ctx context.Context) error {
	metricsClient := mw.client.GetMetricsClient()
	podMetrics, err := metricsClient.GetAllPodsMetrics(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get pod metrics")
		return err
	}

	logger.Info().Int("pod_count", len(podMetrics)).Msg("Checking pod metrics")

	for _, metrics := range podMetrics {
		// Skip if no resource requests (can't calculate percentage)
		if metrics.CPURequestMillis == 0 && metrics.MemoryRequestBytes == 0 {
			continue
		}

		// Check CPU threshold
		if metrics.CPURequestMillis > 0 && metrics.CPUUsagePercent > mw.thresholds.PodCPUPercent {
			alert := BuildPodMetricAlert(
				metrics.Namespace,
				metrics.PodName,
				AlertTypePodCPUHigh,
				metrics.CPUUsagePercent,
				mw.thresholds.PodCPUPercent,
			)

			if _, err := mw.stateManager.ProcessAlert(ctx, alert); err != nil {
				logger.Error().Err(err).Str("pod", metrics.PodName).Msg("Failed to create pod CPU alert")
			} else {
				logger.Info().
					Str("pod", metrics.PodName).
					Float64("cpu_percent", metrics.CPUUsagePercent).
					Msg("Pod CPU alert created")
			}
		}

		// Check Memory threshold
		if metrics.MemoryRequestBytes > 0 && metrics.MemoryUsagePercent > mw.thresholds.PodMemoryPercent {
			alert := BuildPodMetricAlert(
				metrics.Namespace,
				metrics.PodName,
				AlertTypePodMemoryHigh,
				metrics.MemoryUsagePercent,
				mw.thresholds.PodMemoryPercent,
			)

			if _, err := mw.stateManager.ProcessAlert(ctx, alert); err != nil {
				logger.Error().Err(err).Str("pod", metrics.PodName).Msg("Failed to create pod memory alert")
			} else {
				logger.Info().
					Str("pod", metrics.PodName).
					Float64("memory_percent", metrics.MemoryUsagePercent).
					Msg("Pod memory alert created")
			}
		}
	}

	return nil
}

// checkNodeMetrics checks all node metrics
func (mw *MetricsWatcher) checkNodeMetrics(ctx context.Context) error {
	metricsClient := mw.client.GetMetricsClient()
	nodeMetrics, err := metricsClient.GetAllNodesMetrics(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get node metrics")
		return err
	}

	logger.Info().Int("node_count", len(nodeMetrics)).Msg("Checking node metrics")

	for _, metrics := range nodeMetrics {
		// Check CPU threshold
		if metrics.CPUUsagePercent > mw.thresholds.NodeCPUPercent {
			alert := BuildNodeMetricAlert(
				metrics.NodeName,
				AlertTypeNodeCPUHigh,
				metrics.CPUUsagePercent,
				mw.thresholds.NodeCPUPercent,
			)

			if _, err := mw.stateManager.ProcessAlert(ctx, alert); err != nil {
				logger.Error().Err(err).Str("node", metrics.NodeName).Msg("Failed to create node CPU alert")
			} else {
				logger.Info().
					Str("node", metrics.NodeName).
					Float64("cpu_percent", metrics.CPUUsagePercent).
					Msg("Node CPU alert created")
			}
		}

		// Check Memory threshold
		if metrics.MemoryUsagePercent > mw.thresholds.NodeMemoryPercent {
			alert := BuildNodeMetricAlert(
				metrics.NodeName,
				AlertTypeNodeMemoryHigh,
				metrics.MemoryUsagePercent,
				mw.thresholds.NodeMemoryPercent,
			)

			if _, err := mw.stateManager.ProcessAlert(ctx, alert); err != nil {
				logger.Error().Err(err).Str("node", metrics.NodeName).Msg("Failed to create node memory alert")
			} else {
				logger.Info().
					Str("node", metrics.NodeName).
					Float64("memory_percent", metrics.MemoryUsagePercent).
					Msg("Node memory alert created")
			}
		}
	}

	return nil
}

// Stop gracefully stops the metrics watcher
func (mw *MetricsWatcher) Stop() {
	close(mw.stopCh)
	mw.wg.Wait()
	logger.Info().Msg("Metrics watcher fully stopped")
}
