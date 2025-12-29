// Package k8s provides Kubernetes client initialization and utilities for the Nightcrier agent.
package k8s

import (
	"context"
	"fmt"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// JobStatus represents the final status of a Job.
type JobStatus string

const (
	// JobStatusSucceeded indicates the Job completed successfully
	JobStatusSucceeded JobStatus = "Succeeded"
	// JobStatusFailed indicates the Job failed
	JobStatusFailed JobStatus = "Failed"
	// JobStatusTimeout indicates the Job exceeded its deadline
	JobStatusTimeout JobStatus = "Timeout"
	// JobStatusUnknown indicates the Job status couldn't be determined
	JobStatusUnknown JobStatus = "Unknown"
)

// JobWatchResult contains the result of watching a Job to completion.
type JobWatchResult struct {
	// Status is the final status of the Job
	Status JobStatus
	// Message provides additional context about the Job status
	Message string
	// CompletionTime is when the Job completed
	CompletionTime time.Time
	// Job is the final Job object
	Job *batchv1.Job
}

// WatchJobConfig holds configuration for watching a Job.
type WatchJobConfig struct {
	// Namespace is the Kubernetes namespace where the Job is running
	Namespace string
	// JobName is the name of the Job to watch
	JobName string
	// Timeout is how long to wait for Job completion
	// If zero, no timeout is applied
	Timeout time.Duration
	// LogFunc is an optional function to call for logging key events
	// It receives event messages like "Job started", "Pod running", etc.
	LogFunc func(message string)
}

// WatchJob watches a Job until it completes (Succeeded/Failed) or times out.
// It handles watch connection drops and automatically reconnects.
// If LogFunc is provided, it logs key events during Job execution.
//
// The function returns when:
// - The Job completes successfully (Succeeded)
// - The Job fails (Failed condition or deadline exceeded)
// - The watch timeout is reached
// - The context is canceled
func (c *Client) WatchJob(ctx context.Context, cfg WatchJobConfig) (*JobWatchResult, error) {
	if cfg.Namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	if cfg.JobName == "" {
		return nil, fmt.Errorf("job name is required")
	}

	// Apply context timeout if specified
	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	// Log helper
	logEvent := func(message string) {
		if cfg.LogFunc != nil {
			cfg.LogFunc(message)
		}
	}

	// Initial check: verify Job exists
	job, err := c.clientset.BatchV1().Jobs(cfg.Namespace).Get(ctx, cfg.JobName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get Job %s: %w", cfg.JobName, err)
	}

	// Check if Job already completed
	if isJobComplete(job) {
		result := buildJobResult(job)
		logEvent(fmt.Sprintf("Job %s already completed with status: %s", cfg.JobName, result.Status))
		return result, nil
	}

	logEvent(fmt.Sprintf("Job %s started, waiting for completion", cfg.JobName))

	// Watch the Job for changes
	// We use a loop to handle reconnections
	for {
		select {
		case <-ctx.Done():
			// Context canceled or timed out
			if ctx.Err() == context.DeadlineExceeded {
				return &JobWatchResult{
					Status:  JobStatusTimeout,
					Message: fmt.Sprintf("Watch timeout exceeded (%s)", cfg.Timeout),
					Job:     job,
				}, nil
			}
			return nil, fmt.Errorf("watch canceled: %w", ctx.Err())
		default:
			// Continue to watch
		}

		// Start watching from the current resource version
		listOpts := metav1.ListOptions{
			FieldSelector:   fmt.Sprintf("metadata.name=%s", cfg.JobName),
			ResourceVersion: job.ResourceVersion,
		}

		watcher, err := c.clientset.BatchV1().Jobs(cfg.Namespace).Watch(ctx, listOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to start watch for Job %s: %w", cfg.JobName, err)
		}

		// Process watch events
		result, shouldReturn := c.processWatchEvents(ctx, watcher, cfg, &job, logEvent)
		if shouldReturn {
			return result, nil
		}

		// Watch disconnected, will retry after a brief delay
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				return &JobWatchResult{
					Status:  JobStatusTimeout,
					Message: fmt.Sprintf("Watch timeout exceeded (%s)", cfg.Timeout),
					Job:     job,
				}, nil
			}
			return nil, fmt.Errorf("watch canceled: %w", ctx.Err())
		case <-time.After(1 * time.Second):
			// Brief delay before reconnecting
			logEvent(fmt.Sprintf("Watch disconnected for Job %s, reconnecting...", cfg.JobName))
		}
	}
}

// processWatchEvents processes events from a watch channel.
// Returns (result, true) if the Job completed, (nil, false) if the watch should reconnect.
func (c *Client) processWatchEvents(
	ctx context.Context,
	watcher watch.Interface,
	cfg WatchJobConfig,
	currentJob **batchv1.Job,
	logEvent func(string),
) (*JobWatchResult, bool) {
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			// Context canceled
			return nil, true
		case event, ok := <-watcher.ResultChan():
			if !ok {
				// Watch channel closed, need to reconnect
				return nil, false
			}

			switch event.Type {
			case watch.Added, watch.Modified:
				job, ok := event.Object.(*batchv1.Job)
				if !ok {
					logEvent(fmt.Sprintf("Warning: Unexpected object type in watch event: %T", event.Object))
					continue
				}

				// Update current Job reference
				*currentJob = job

				// Log Pod status changes
				c.logPodStatusIfChanged(ctx, cfg.Namespace, job, logEvent)

				// Check if Job completed
				if isJobComplete(job) {
					result := buildJobResult(job)
					logEvent(fmt.Sprintf("Job %s completed with status: %s", cfg.JobName, result.Status))
					return result, true
				}

			case watch.Deleted:
				// Job was deleted
				return &JobWatchResult{
					Status:  JobStatusUnknown,
					Message: "Job was deleted during execution",
				}, true

			case watch.Error:
				// Watch error, will reconnect
				logEvent(fmt.Sprintf("Watch error for Job %s, will reconnect", cfg.JobName))
				return nil, false
			}
		}
	}
}

// logPodStatusIfChanged logs when a Job's Pod starts running.
// This provides visibility into Job lifecycle: scheduled -> running -> complete.
func (c *Client) logPodStatusIfChanged(ctx context.Context, namespace string, job *batchv1.Job, logEvent func(string)) {
	// Only log if we have active pods
	if job.Status.Active == 0 {
		return
	}

	// Get the Pod(s) for this Job
	labelSelector := fmt.Sprintf("job-name=%s", job.Name)
	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return
	}

	// Log if any pod is running
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			logEvent(fmt.Sprintf("Pod %s is running", pod.Name))
			return
		}
	}
}

// isJobComplete checks if a Job has reached a terminal state.
func isJobComplete(job *batchv1.Job) bool {
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobComplete && condition.Status == corev1.ConditionTrue {
			return true
		}
		if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// buildJobResult constructs a JobWatchResult from a completed Job.
func buildJobResult(job *batchv1.Job) *JobWatchResult {
	result := &JobWatchResult{
		Job: job,
	}

	// Check completion conditions
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobComplete && condition.Status == corev1.ConditionTrue {
			result.Status = JobStatusSucceeded
			result.Message = condition.Message
			result.CompletionTime = condition.LastTransitionTime.Time
			return result
		}
		if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			result.Status = JobStatusFailed
			result.Message = condition.Message
			result.CompletionTime = condition.LastTransitionTime.Time

			// Check if failure was due to deadline
			if condition.Reason == "DeadlineExceeded" {
				result.Status = JobStatusTimeout
			}
			return result
		}
	}

	// Job is complete but no condition found
	result.Status = JobStatusUnknown
	result.Message = "Job completed but no success/failure condition found"
	return result
}

// GetPodLogs retrieves logs from the first Pod of a Job.
// This is useful for debugging or providing real-time log streaming.
// For full log streaming, use kubectl logs -f or the K8s API directly.
func (c *Client) GetPodLogs(ctx context.Context, namespace, jobName string) (string, error) {
	// Get the Pod(s) for this Job
	labelSelector := fmt.Sprintf("job-name=%s", jobName)
	pods, err := c.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return "", fmt.Errorf("failed to list pods for Job %s: %w", jobName, err)
	}

	if len(pods.Items) == 0 {
		return "", fmt.Errorf("no pods found for Job %s", jobName)
	}

	// Get logs from the first pod
	pod := pods.Items[0]
	podLogOpts := corev1.PodLogOptions{}
	req := c.clientset.CoreV1().Pods(namespace).GetLogs(pod.Name, &podLogOpts)
	logs, err := req.DoRaw(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get logs for Pod %s: %w", pod.Name, err)
	}

	return string(logs), nil
}
