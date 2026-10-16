package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrBackgroundJobNotFound = errors.New("background job not found")

type JobRepository struct {
	db *pgxpool.Pool
}

func NewJobRepository(db *pgxpool.Pool) *JobRepository {
	return &JobRepository{db: db}
}

const JobTypeConfirmationEmail = "confirmation_email"

// Enqueue creates one logical job per submission and job type.
// Repeated calls return the existing job rather than creating duplicates.
func (r *JobRepository) Enqueue(
	ctx context.Context,
	submissionID string,
	jobType string,
) (*models.BackgroundJob, error) {
	submissionID = strings.TrimSpace(submissionID)
	jobType = strings.TrimSpace(jobType)

	if submissionID == "" {
		return nil, errors.New("submission ID is required")
	}
	if jobType != JobTypeConfirmationEmail {
		return nil, fmt.Errorf("unsupported background job type: %q", jobType)
	}

	_, err := r.db.Exec(ctx, `
		INSERT INTO background_jobs (submission_id, job_type)
		VALUES ($1, $2)
		ON CONFLICT (submission_id, job_type) DO NOTHING
	`, submissionID, jobType)
	if err != nil {
		return nil, fmt.Errorf("enqueue background job: %w", err)
	}

	var job models.BackgroundJob
	var lockedAt, completedAt sql.NullTime
	var lastError sql.NullString

	err = r.db.QueryRow(ctx, `
		SELECT
			id,
			submission_id,
			job_type,
			status,
			attempts,
			max_attempts,
			run_after,
			locked_at,
			last_error,
			created_at,
			updated_at,
			completed_at
		FROM background_jobs
		WHERE submission_id = $1 AND job_type = $2
	`, submissionID, jobType).Scan(
		&job.ID,
		&job.SubmissionID,
		&job.JobType,
		&job.Status,
		&job.Attempts,
		&job.MaxAttempts,
		&job.RunAfter,
		&lockedAt,
		&lastError,
		&job.CreatedAt,
		&job.UpdatedAt,
		&completedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("load enqueued background job: %w", err)
	}

	if lockedAt.Valid {
		job.LockedAt = &lockedAt.Time
	}
	if lastError.Valid {
		job.LastError = lastError.String
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}

	return &job, nil
}

// ClaimNext atomically claims one available job.
// SKIP LOCKED allows multiple workers to claim different jobs concurrently.
func (r *JobRepository) ClaimNext(
	ctx context.Context,
) (*models.BackgroundJob, error) {
	const query = `
		WITH candidate AS (
			SELECT id
			FROM background_jobs
			WHERE
				(status = 'pending' AND run_after <= NOW())
				OR
				(
					status = 'running'
					AND locked_at < NOW() - INTERVAL '2 minutes'
					AND attempts < max_attempts
				)
			ORDER BY run_after, created_at
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		UPDATE background_jobs AS job
		SET
			status = 'running',
			attempts = job.attempts + 1,
			locked_at = NOW(),
			updated_at = NOW()
		FROM candidate
		WHERE job.id = candidate.id
		RETURNING
			job.id,
			job.submission_id,
			job.job_type,
			job.status,
			job.attempts,
			job.max_attempts,
			job.run_after,
			job.locked_at,
			job.last_error,
			job.created_at,
			job.updated_at,
			job.completed_at
	`

	var job models.BackgroundJob
	var lockedAt, completedAt sql.NullTime
	var lastError sql.NullString

	err := r.db.QueryRow(ctx, query).Scan(
		&job.ID,
		&job.SubmissionID,
		&job.JobType,
		&job.Status,
		&job.Attempts,
		&job.MaxAttempts,
		&job.RunAfter,
		&lockedAt,
		&lastError,
		&job.CreatedAt,
		&job.UpdatedAt,
		&completedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("claim background job: %w", err)
	}

	if lockedAt.Valid {
		job.LockedAt = &lockedAt.Time
	}
	if completedAt.Valid {
		job.CompletedAt = &completedAt.Time
	}
	if lastError.Valid {
		job.LastError = lastError.String
	}

	var submissionData []byte
	err = r.db.QueryRow(
		ctx,
		`SELECT data FROM submissions WHERE id = $1`,
		job.SubmissionID,
	).Scan(&submissionData)
	if err != nil {
		return nil, fmt.Errorf("load submission for job: %w", err)
	}

	job.SubmissionData = json.RawMessage(submissionData)
	return &job, nil
}

// Complete marks a claimed job as successfully processed.
func (r *JobRepository) Complete(
	ctx context.Context,
	jobID string,
) error {
	tag, err := r.db.Exec(
		ctx,
		`UPDATE background_jobs
		 SET status = 'succeeded',
		     locked_at = NULL,
		     last_error = NULL,
		     updated_at = NOW(),
		     completed_at = NOW()
		 WHERE id = $1 AND status = 'running'`,
		jobID,
	)
	if err != nil {
		return fmt.Errorf("complete background job: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrBackgroundJobNotFound
	}

	return nil
}

// RetryOrFail schedules another attempt or marks the job permanently failed.
// ClaimNext increments Attempts when it claims a job.
func (r *JobRepository) RetryOrFail(
	ctx context.Context,
	job models.BackgroundJob,
	cause error,
	delay time.Duration,
) (bool, error) {
	if cause == nil {
		cause = errors.New("unspecified background job failure")
	}
	if delay < 0 {
		delay = 0
	}

	message := strings.TrimSpace(cause.Error())
	if len(message) > 2000 {
		message = message[:2000]
	}

	var query string
	var args []any
	var permanentlyFailed bool

	if job.Attempts >= job.MaxAttempts {
		permanentlyFailed = true
		query = `
			UPDATE background_jobs
			SET status = 'failed',
			    last_error = $2,
			    locked_at = NULL,
			    updated_at = NOW(),
			    completed_at = NOW()
			WHERE id = $1 AND status = 'running'
		`
		args = []any{job.ID, message}
	} else {
		query = `
			UPDATE background_jobs
			SET status = 'pending',
			    run_after = NOW() + ($3 * INTERVAL '1 second'),
			    last_error = $2,
			    locked_at = NULL,
			    updated_at = NOW()
			WHERE id = $1 AND status = 'running'
		`
		args = []any{job.ID, message, delay.Seconds()}
	}

	tag, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("reschedule background job: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return false, ErrBackgroundJobNotFound
	}

	return permanentlyFailed, nil
}

// FailExhaustedStale marks jobs failed when a worker crashed during its final
// allowed attempt and the job's lock has expired.
func (r *JobRepository) FailExhaustedStale(
	ctx context.Context,
) (int64, error) {
	tag, err := r.db.Exec(ctx, `
		UPDATE background_jobs
		SET status = 'failed',
		    last_error = COALESCE(
			    last_error,
			    'worker lease expired after final attempt'
		    ),
		    locked_at = NULL,
		    updated_at = NOW(),
		    completed_at = NOW()
		WHERE status = 'running'
		  AND locked_at < NOW() - INTERVAL '2 minutes'
		  AND attempts >= max_attempts
	`)
	if err != nil {
		return 0, fmt.Errorf("fail exhausted stale jobs: %w", err)
	}

	return tag.RowsAffected(), nil
}
