package jobs

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
)

type fakeJobStore struct {
	job *models.BackgroundJob

	claimErr    error
	completeErr error
	retryErr    error
	staleErr    error

	staleFailed   int64
	claimCalls    int
	completeCalls int
	completedID   string

	retryCalls             int
	retryPermanentlyFailed bool
	retryCause             error
	retryDelay             time.Duration
	retryJob               models.BackgroundJob
}

func (s *fakeJobStore) ClaimNext(
	_ context.Context,
) (*models.BackgroundJob, error) {
	s.claimCalls++
	if s.claimErr != nil {
		return nil, s.claimErr
	}
	if s.job == nil {
		return nil, nil
	}

	job := *s.job
	s.job = nil
	return &job, nil
}

func (s *fakeJobStore) Complete(
	_ context.Context,
	jobID string,
) error {
	s.completeCalls++
	s.completedID = jobID
	return s.completeErr
}

func (s *fakeJobStore) RetryOrFail(
	_ context.Context,
	job models.BackgroundJob,
	cause error,
	delay time.Duration,
) (bool, error) {
	s.retryCalls++
	s.retryJob = job
	s.retryCause = cause
	s.retryDelay = delay
	return s.retryPermanentlyFailed, s.retryErr
}

func (s *fakeJobStore) FailExhaustedStale(
	_ context.Context,
) (int64, error) {
	return s.staleFailed, s.staleErr
}

type senderFunc func(
	context.Context,
	models.BackgroundJob,
) error

func (f senderFunc) SendConfirmation(
	ctx context.Context,
	job models.BackgroundJob,
) error {
	return f(ctx, job)
}

func newTestJob(attempts, maxAttempts int) *models.BackgroundJob {
	return &models.BackgroundJob{
		ID:           "job-123",
		SubmissionID: "submission-123",
		JobType:      "confirmation_email",
		Status:       "running",
		Attempts:     attempts,
		MaxAttempts:  maxAttempts,
	}
}

func TestWorkerProcessesAndCompletesJob(t *testing.T) {
	store := &fakeJobStore{job: newTestJob(1, 5)}
	var senderCalls int

	worker := NewWorker(store, senderFunc(
		func(_ context.Context, job models.BackgroundJob) error {
			senderCalls++
			if job.ID != "job-123" {
				t.Fatalf("unexpected job: %q", job.ID)
			}
			return nil
		},
	))

	if err := worker.ProcessNext(context.Background()); err != nil {
		t.Fatalf("ProcessNext failed: %v", err)
	}

	if senderCalls != 1 {
		t.Fatalf("expected one sender call, got %d", senderCalls)
	}
	if store.completeCalls != 1 || store.completedID != "job-123" {
		t.Fatalf("job was not completed correctly: %+v", store)
	}
	if store.retryCalls != 0 {
		t.Fatal("successful job must not be retried")
	}
}

func TestWorkerSchedulesRetryOnFailure(t *testing.T) {
	store := &fakeJobStore{job: newTestJob(1, 5)}
	deliveryErr := errors.New("temporary delivery failure")

	worker := NewWorker(store, senderFunc(
		func(context.Context, models.BackgroundJob) error {
			return deliveryErr
		},
	))

	if err := worker.ProcessNext(context.Background()); err != nil {
		t.Fatalf("ProcessNext should handle a retryable failure: %v", err)
	}

	if store.retryCalls != 1 {
		t.Fatalf("expected one retry call, got %d", store.retryCalls)
	}
	if !errors.Is(store.retryCause, deliveryErr) {
		t.Fatalf("unexpected retry cause: %v", store.retryCause)
	}
	if store.retryDelay != defaultRetryDelay {
		t.Fatalf("unexpected first retry delay: %v", store.retryDelay)
	}
	if store.completeCalls != 0 {
		t.Fatal("failed job must not be marked complete")
	}
}

func TestWorkerAlertsWhenJobPermanentlyFails(t *testing.T) {
	store := &fakeJobStore{
		job:                    newTestJob(5, 5),
		retryPermanentlyFailed: true,
	}

	var logs bytes.Buffer
	worker := NewWorker(store, senderFunc(
		func(context.Context, models.BackgroundJob) error {
			return errors.New("delivery failed")
		},
	))
	worker.logger = log.New(&logs, "", 0)

	if err := worker.ProcessNext(context.Background()); err != nil {
		t.Fatalf("ProcessNext failed: %v", err)
	}

	if store.retryCalls != 1 || !store.retryPermanentlyFailed {
		t.Fatal("expected exhausted job to be marked permanently failed")
	}
	if !strings.Contains(logs.String(), "ALERT: background job permanently failed") {
		t.Fatalf("expected terminal failure alert, got %q", logs.String())
	}
	if store.completeCalls != 0 {
		t.Fatal("permanently failed job must not be completed")
	}
}

func TestWorkerRetriesUnsupportedJobType(t *testing.T) {
	job := newTestJob(1, 5)
	job.JobType = "unsupported"

	store := &fakeJobStore{job: job}
	var senderCalls int

	worker := NewWorker(store, senderFunc(
		func(context.Context, models.BackgroundJob) error {
			senderCalls++
			return nil
		},
	))

	if err := worker.ProcessNext(context.Background()); err != nil {
		t.Fatalf("ProcessNext failed: %v", err)
	}

	if senderCalls != 0 {
		t.Fatal("unsupported job type reached the sender")
	}
	if store.retryCalls != 1 {
		t.Fatal("unsupported job type should be handled as a failure")
	}
}

func TestWorkerRejectsMissingStore(t *testing.T) {
	worker := NewWorker(nil, nil)

	if err := worker.ProcessNext(context.Background()); err == nil {
		t.Fatal("expected an error when the job store is missing")
	}
}

func TestRetryBackoffIsBounded(t *testing.T) {
	if got := retryBackoff(1, defaultRetryDelay); got != 2*time.Second {
		t.Fatalf("first retry should be 2 seconds, got %v", got)
	}

	if got := retryBackoff(2, defaultRetryDelay); got != 4*time.Second {
		t.Fatalf("second retry should be 4 seconds, got %v", got)
	}

	if got := retryBackoff(20, defaultRetryDelay); got != maxRetryDelay {
		t.Fatalf("expected retry delay capped at %v, got %v", maxRetryDelay, got)
	}
}
