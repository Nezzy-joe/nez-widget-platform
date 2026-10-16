package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
)

func TestJobRepositoryLifecycle(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is required for PostgreSQL integration tests")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		20*time.Second,
	)
	defer cancel()

	pool, err := NewPostgresPool(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer pool.Close()

	tenantID := createIntegrationTenant(t, ctx, pool, "jobs")

	widgetRepo := NewWidgetRepository(pool)
	submissionRepo := NewSubmissionRepository(pool)
	jobRepo := NewJobRepository(pool)

	widget, err := widgetRepo.Create(ctx, tenantID, &models.Widget{
		PublicID:       fmt.Sprintf("integration-job-%d", time.Now().UnixNano()),
		Type:           "cta",
		Title:          "Background Job Test",
		ButtonText:     "Continue",
		DisplayOptions: json.RawMessage(`{}`),
	})
	if err != nil {
		t.Fatalf("create test widget: %v", err)
	}

	// Enqueue a notification for a persisted submission.
	submission, err := submissionRepo.Create(ctx, tenantID, &models.Submission{
		WidgetID: widget.ID,
		Data:     json.RawMessage(`{"email":"job@example.com"}`),
	})
	if err != nil {
		t.Fatalf("create test submission: %v", err)
	}

	job, err := jobRepo.Enqueue(
		ctx,
		submission.ID,
		JobTypeConfirmationEmail,
	)
	if err != nil {
		t.Fatalf("enqueue job: %v", err)
	}
	// Make this test job the earliest eligible pending job.
	_, err = pool.Exec(
		ctx,
		`UPDATE background_jobs
     SET run_after = NOW() - INTERVAL '365 days'
     WHERE id = $1`,
		job.ID,
	)
	if err != nil {
		t.Fatalf("prioritize test job: %v", err)
	}

	if job.Status != "pending" {
		t.Fatalf("expected pending job, got %q", job.Status)
	}
	if job.Attempts != 0 {
		t.Fatalf("expected zero attempts before claim, got %d", job.Attempts)
	}

	// Enqueuing the same logical job must not create a duplicate.
	duplicate, err := jobRepo.Enqueue(
		ctx,
		submission.ID,
		JobTypeConfirmationEmail,
	)
	if err != nil {
		t.Fatalf("enqueue duplicate job: %v", err)
	}
	if duplicate.ID != job.ID {
		t.Fatalf("duplicate enqueue created another job: %q != %q",
			duplicate.ID, job.ID)
	}

	// ClaimNext should increment attempts and load submission data.
	claimed, err := jobRepo.ClaimNext(ctx)
	if err != nil {
		t.Fatalf("claim job: %v", err)
	}
	if claimed == nil {
		t.Fatal("expected a job to be claimed")
	}
	if claimed.ID != job.ID {
		t.Fatalf("claimed the wrong job: %q", claimed.ID)
	}
	if claimed.Status != "running" || claimed.Attempts != 1 {
		t.Fatalf("unexpected claimed job state: %+v", claimed)
	}

	var submittedData map[string]string
	if err := json.Unmarshal(claimed.SubmissionData, &submittedData); err != nil {
		t.Fatalf("decode claimed submission data: %v", err)
	}
	if submittedData["email"] != "job@example.com" {
		t.Fatalf("unexpected submission data: %+v", submittedData)
	}

	if err := jobRepo.Complete(ctx, claimed.ID); err != nil {
		t.Fatalf("complete job: %v", err)
	}

	var status string
	err = pool.QueryRow(
		ctx,
		`SELECT status FROM background_jobs WHERE id = $1`,
		job.ID,
	).Scan(&status)
	if err != nil {
		t.Fatalf("read completed job status: %v", err)
	}
	if status != "succeeded" {
		t.Fatalf("expected succeeded status, got %q", status)
	}

	// Create a separate job to test retry and terminal failure behavior.
	retrySubmission, err := submissionRepo.Create(ctx, tenantID, &models.Submission{
		WidgetID: widget.ID,
		Data:     json.RawMessage(`{"email":"retry@example.com"}`),
	})
	if err != nil {
		t.Fatalf("create retry test submission: %v", err)
	}

	retryJob, err := jobRepo.Enqueue(
		ctx,
		retrySubmission.ID,
		JobTypeConfirmationEmail,
	)
	if err != nil {
		t.Fatalf("enqueue retry test job: %v", err)
	}
	// Make this test's job the earliest eligible job in the shared queue.
	_, err = pool.Exec(
		ctx,
		`UPDATE background_jobs
     SET run_after = TIMESTAMPTZ '1970-01-01 00:00:00+00'
     WHERE id = $1`,
		retryJob.ID,
	)
	if err != nil {
		t.Fatalf("prioritize retry test job: %v", err)
	}

	// Lower the limit for this test so exhaustion is deterministic and fast.
	_, err = pool.Exec(
		ctx,
		`UPDATE background_jobs SET max_attempts = 2 WHERE id = $1`,
		retryJob.ID,
	)
	if err != nil {
		t.Fatalf("set retry limit: %v", err)
	}

	firstAttempt, err := jobRepo.ClaimNext(ctx)
	if err != nil {
		t.Fatalf("claim first retry attempt: %v", err)
	}
	if firstAttempt == nil || firstAttempt.ID != retryJob.ID {
		t.Fatal("expected to claim the retry test job")
	}
	if firstAttempt.Attempts != 1 {
		t.Fatalf("expected attempt 1, got %d", firstAttempt.Attempts)
	}

	permanentlyFailed, err := jobRepo.RetryOrFail(
		ctx,
		*firstAttempt,
		errors.New("temporary delivery failure"),
		0,
	)
	if err != nil {
		t.Fatalf("schedule retry: %v", err)
	}
	if permanentlyFailed {
		t.Fatal("job should be retried before exhausting its attempts")
	}

	// RetryOrFail changes run_after, so prioritize the job again.
	_, err = pool.Exec(
		ctx,
		`UPDATE background_jobs
     SET run_after = TIMESTAMPTZ '1970-01-01 00:00:00+00'
     WHERE id = $1`,
		retryJob.ID,
	)
	if err != nil {
		t.Fatalf("reprioritize retry test job: %v", err)
	}
	// RetryOrFail updates run_after, so prioritize this job again
	// before claiming its second attempt.

	secondAttempt, err := jobRepo.ClaimNext(ctx)
	if err != nil {
		t.Fatalf("claim second retry attempt: %v", err)
	}
	if secondAttempt == nil || secondAttempt.ID != retryJob.ID {
		t.Fatal("expected to reclaim the retry test job")
	}
	if secondAttempt.Attempts != 2 {
		t.Fatalf("expected attempt 2, got %d", secondAttempt.Attempts)
	}

	permanentlyFailed, err = jobRepo.RetryOrFail(
		ctx,
		*secondAttempt,
		errors.New("delivery failed again"),
		0,
	)
	if err != nil {
		t.Fatalf("mark exhausted job as failed: %v", err)
	}
	if !permanentlyFailed {
		t.Fatal("expected job to exhaust its retry limit")
	}

	var finalStatus string
	var attempts int
	err = pool.QueryRow(
		ctx,
		`SELECT status, attempts FROM background_jobs WHERE id = $1`,
		retryJob.ID,
	).Scan(&finalStatus, &attempts)
	if err != nil {
		t.Fatalf("read final retry status: %v", err)
	}
	if finalStatus != "failed" || attempts != 2 {
		t.Fatalf("unexpected final state: status=%q attempts=%d",
			finalStatus, attempts)
	}
}
