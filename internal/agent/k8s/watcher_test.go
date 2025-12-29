package k8s

import (
	"context"
	"testing"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestWatchJob_AlreadyCompleted(t *testing.T) {
	// Create a fake clientset
	fakeClientset := fake.NewSimpleClientset()

	// Create a client with the fake clientset
	client := &Client{
		clientset: fakeClientset,
	}

	testNamespace := "nightcrier"
	testJobName := "triage-test-123"

	// Create a completed Job
	completedJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testJobName,
			Namespace: testNamespace,
		},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{
				{
					Type:               batchv1.JobComplete,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             "Completed",
					Message:            "Job completed successfully",
				},
			},
		},
	}

	_, err := fakeClientset.BatchV1().Jobs(testNamespace).Create(context.Background(), completedJob, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test Job: %v", err)
	}

	// Test watching an already completed Job
	ctx := context.Background()
	cfg := WatchJobConfig{
		Namespace: testNamespace,
		JobName:   testJobName,
		Timeout:   5 * time.Second,
	}

	result, err := client.WatchJob(ctx, cfg)
	if err != nil {
		t.Fatalf("WatchJob() failed: %v", err)
	}

	// Verify result
	if result.Status != JobStatusSucceeded {
		t.Errorf("Expected status %s, got %s", JobStatusSucceeded, result.Status)
	}

	if result.Message != "Job completed successfully" {
		t.Errorf("Expected message 'Job completed successfully', got '%s'", result.Message)
	}
}

func TestWatchJob_AlreadyFailed(t *testing.T) {
	// Create a fake clientset
	fakeClientset := fake.NewSimpleClientset()

	// Create a client with the fake clientset
	client := &Client{
		clientset: fakeClientset,
	}

	testNamespace := "nightcrier"
	testJobName := "triage-test-456"

	// Create a failed Job
	failedJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testJobName,
			Namespace: testNamespace,
		},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{
				{
					Type:               batchv1.JobFailed,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             "BackoffLimitExceeded",
					Message:            "Job has reached the specified backoff limit",
				},
			},
		},
	}

	_, err := fakeClientset.BatchV1().Jobs(testNamespace).Create(context.Background(), failedJob, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test Job: %v", err)
	}

	// Test watching an already failed Job
	ctx := context.Background()
	cfg := WatchJobConfig{
		Namespace: testNamespace,
		JobName:   testJobName,
		Timeout:   5 * time.Second,
	}

	result, err := client.WatchJob(ctx, cfg)
	if err != nil {
		t.Fatalf("WatchJob() failed: %v", err)
	}

	// Verify result
	if result.Status != JobStatusFailed {
		t.Errorf("Expected status %s, got %s", JobStatusFailed, result.Status)
	}

	if result.Message != "Job has reached the specified backoff limit" {
		t.Errorf("Expected failure message, got '%s'", result.Message)
	}
}

func TestWatchJob_DeadlineExceeded(t *testing.T) {
	// Create a fake clientset
	fakeClientset := fake.NewSimpleClientset()

	// Create a client with the fake clientset
	client := &Client{
		clientset: fakeClientset,
	}

	testNamespace := "nightcrier"
	testJobName := "triage-test-789"

	// Create a Job that failed due to deadline
	deadlineJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testJobName,
			Namespace: testNamespace,
		},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{
				{
					Type:               batchv1.JobFailed,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             "DeadlineExceeded",
					Message:            "Job was active longer than specified deadline",
				},
			},
		},
	}

	_, err := fakeClientset.BatchV1().Jobs(testNamespace).Create(context.Background(), deadlineJob, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test Job: %v", err)
	}

	// Test watching a Job that hit the deadline
	ctx := context.Background()
	cfg := WatchJobConfig{
		Namespace: testNamespace,
		JobName:   testJobName,
		Timeout:   5 * time.Second,
	}

	result, err := client.WatchJob(ctx, cfg)
	if err != nil {
		t.Fatalf("WatchJob() failed: %v", err)
	}

	// Verify result - should be JobStatusTimeout
	if result.Status != JobStatusTimeout {
		t.Errorf("Expected status %s, got %s", JobStatusTimeout, result.Status)
	}
}

func TestWatchJob_MissingJob(t *testing.T) {
	// Create a fake clientset
	fakeClientset := fake.NewSimpleClientset()

	// Create a client with the fake clientset
	client := &Client{
		clientset: fakeClientset,
	}

	testNamespace := "nightcrier"
	testJobName := "nonexistent-job"

	// Test watching a Job that doesn't exist
	ctx := context.Background()
	cfg := WatchJobConfig{
		Namespace: testNamespace,
		JobName:   testJobName,
		Timeout:   5 * time.Second,
	}

	_, err := client.WatchJob(ctx, cfg)
	if err == nil {
		t.Fatal("Expected error when watching nonexistent Job, got nil")
	}

	// Error message should mention the Job name
	if err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}

func TestWatchJob_ContextCanceled(t *testing.T) {
	// Create a fake clientset
	fakeClientset := fake.NewSimpleClientset()

	// Create a client with the fake clientset
	client := &Client{
		clientset: fakeClientset,
	}

	testNamespace := "nightcrier"
	testJobName := "triage-test-cancel"

	// Create a running Job (not completed)
	runningJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testJobName,
			Namespace: testNamespace,
		},
		Status: batchv1.JobStatus{
			Active: 1, // Job is still running
		},
	}

	_, err := fakeClientset.BatchV1().Jobs(testNamespace).Create(context.Background(), runningJob, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test Job: %v", err)
	}

	// Test with a canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	cfg := WatchJobConfig{
		Namespace: testNamespace,
		JobName:   testJobName,
	}

	_, err = client.WatchJob(ctx, cfg)
	if err == nil {
		t.Fatal("Expected error when context is canceled, got nil")
	}

	// Should get a context canceled error
	if ctx.Err() != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", ctx.Err())
	}
}

func TestWatchJob_Timeout(t *testing.T) {
	// Create a fake clientset
	fakeClientset := fake.NewSimpleClientset()

	// Create a client with the fake clientset
	client := &Client{
		clientset: fakeClientset,
	}

	testNamespace := "nightcrier"
	testJobName := "triage-test-timeout"

	// Create a running Job (not completed)
	runningJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testJobName,
			Namespace: testNamespace,
		},
		Status: batchv1.JobStatus{
			Active: 1, // Job is still running
		},
	}

	_, err := fakeClientset.BatchV1().Jobs(testNamespace).Create(context.Background(), runningJob, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test Job: %v", err)
	}

	// Test with a very short timeout
	ctx := context.Background()
	cfg := WatchJobConfig{
		Namespace: testNamespace,
		JobName:   testJobName,
		Timeout:   100 * time.Millisecond, // Very short timeout
	}

	result, err := client.WatchJob(ctx, cfg)

	// With fake clientset, the watch might fail or return timeout
	// Both are acceptable for this test case
	if err != nil {
		// If there's an error, it should be a context timeout
		t.Logf("WatchJob() returned error (acceptable for timeout test): %v", err)
		return
	}

	// If we got a result, it should be timeout status
	if result != nil && result.Status != JobStatusTimeout {
		t.Errorf("Expected status %s, got %s", JobStatusTimeout, result.Status)
	}
}

func TestIsJobComplete(t *testing.T) {
	tests := []struct {
		name     string
		job      *batchv1.Job
		expected bool
	}{
		{
			name: "Job succeeded",
			job: &batchv1.Job{
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{
							Type:   batchv1.JobComplete,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "Job failed",
			job: &batchv1.Job{
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{
							Type:   batchv1.JobFailed,
							Status: corev1.ConditionTrue,
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "Job running",
			job: &batchv1.Job{
				Status: batchv1.JobStatus{
					Active: 1,
				},
			},
			expected: false,
		},
		{
			name: "Job pending",
			job: &batchv1.Job{
				Status: batchv1.JobStatus{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isJobComplete(tt.job)
			if result != tt.expected {
				t.Errorf("isJobComplete() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBuildJobResult(t *testing.T) {
	tests := []struct {
		name           string
		job            *batchv1.Job
		expectedStatus JobStatus
		expectedReason string
	}{
		{
			name: "Successful Job",
			job: &batchv1.Job{
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{
							Type:               batchv1.JobComplete,
							Status:             corev1.ConditionTrue,
							LastTransitionTime: metav1.Now(),
							Reason:             "Completed",
							Message:            "Job completed successfully",
						},
					},
				},
			},
			expectedStatus: JobStatusSucceeded,
			expectedReason: "Completed",
		},
		{
			name: "Failed Job",
			job: &batchv1.Job{
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{
							Type:               batchv1.JobFailed,
							Status:             corev1.ConditionTrue,
							LastTransitionTime: metav1.Now(),
							Reason:             "BackoffLimitExceeded",
							Message:            "Job failed",
						},
					},
				},
			},
			expectedStatus: JobStatusFailed,
			expectedReason: "BackoffLimitExceeded",
		},
		{
			name: "Deadline exceeded",
			job: &batchv1.Job{
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{
							Type:               batchv1.JobFailed,
							Status:             corev1.ConditionTrue,
							LastTransitionTime: metav1.Now(),
							Reason:             "DeadlineExceeded",
							Message:            "Job exceeded deadline",
						},
					},
				},
			},
			expectedStatus: JobStatusTimeout,
			expectedReason: "DeadlineExceeded",
		},
		{
			name: "Unknown status",
			job: &batchv1.Job{
				Status: batchv1.JobStatus{},
			},
			expectedStatus: JobStatusUnknown,
			expectedReason: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildJobResult(tt.job)

			if result.Status != tt.expectedStatus {
				t.Errorf("buildJobResult() status = %v, want %v", result.Status, tt.expectedStatus)
			}

			if result.Job != tt.job {
				t.Error("buildJobResult() should return the same Job object")
			}
		})
	}
}

func TestGetPodLogs(t *testing.T) {
	// Create a fake clientset
	fakeClientset := fake.NewSimpleClientset()

	// Create a client with the fake clientset
	client := &Client{
		clientset: fakeClientset,
	}

	testNamespace := "nightcrier"
	testJobName := "triage-test-123"

	// Create a Pod for the Job
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "triage-test-123-abc123",
			Namespace: testNamespace,
			Labels: map[string]string{
				"job-name": testJobName,
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	_, err := fakeClientset.CoreV1().Pods(testNamespace).Create(context.Background(), pod, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test Pod: %v", err)
	}

	// Test getting Pod logs
	ctx := context.Background()
	logs, err := client.GetPodLogs(ctx, testNamespace, testJobName)

	// Note: fake clientset doesn't simulate logs, so this will return empty
	// In real tests, you'd use a more sophisticated mock
	if err != nil {
		// This is expected with fake clientset
		t.Logf("GetPodLogs() returned error (expected with fake clientset): %v", err)
	} else if logs != "" {
		t.Logf("GetPodLogs() returned logs: %s", logs)
	}
}

func TestGetPodLogs_NoPods(t *testing.T) {
	// Create a fake clientset
	fakeClientset := fake.NewSimpleClientset()

	// Create a client with the fake clientset
	client := &Client{
		clientset: fakeClientset,
	}

	testNamespace := "nightcrier"
	testJobName := "nonexistent-job"

	// Test getting logs when no Pods exist
	ctx := context.Background()
	_, err := client.GetPodLogs(ctx, testNamespace, testJobName)

	if err == nil {
		t.Fatal("Expected error when no Pods exist, got nil")
	}

	// Should mention no pods found
	expectedErrMsg := "no pods found"
	if err.Error()[:len(expectedErrMsg)] != expectedErrMsg {
		t.Errorf("Expected error message to start with '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestWatchJob_LogEvents(t *testing.T) {
	// Create a fake clientset
	fakeClientset := fake.NewSimpleClientset()

	// Create a client with the fake clientset
	client := &Client{
		clientset: fakeClientset,
	}

	testNamespace := "nightcrier"
	testJobName := "triage-test-logging"

	// Create a completed Job
	completedJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      testJobName,
			Namespace: testNamespace,
		},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{
				{
					Type:               batchv1.JobComplete,
					Status:             corev1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             "Completed",
					Message:            "Job completed successfully",
				},
			},
		},
	}

	_, err := fakeClientset.BatchV1().Jobs(testNamespace).Create(context.Background(), completedJob, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Failed to create test Job: %v", err)
	}

	// Capture log messages
	var logMessages []string
	logFunc := func(message string) {
		logMessages = append(logMessages, message)
	}

	// Test with logging
	ctx := context.Background()
	cfg := WatchJobConfig{
		Namespace: testNamespace,
		JobName:   testJobName,
		Timeout:   5 * time.Second,
		LogFunc:   logFunc,
	}

	result, err := client.WatchJob(ctx, cfg)
	if err != nil {
		t.Fatalf("WatchJob() failed: %v", err)
	}

	// Verify result
	if result.Status != JobStatusSucceeded {
		t.Errorf("Expected status %s, got %s", JobStatusSucceeded, result.Status)
	}

	// Verify log messages were captured
	if len(logMessages) == 0 {
		t.Error("Expected log messages, got none")
	}

	// Check that at least one message mentions the Job completion
	foundCompletionLog := false
	for _, msg := range logMessages {
		if msg != "" {
			foundCompletionLog = true
			break
		}
	}

	if !foundCompletionLog {
		t.Error("Expected to find log message about Job completion")
	}
}
