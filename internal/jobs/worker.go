package jobs

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
)

const (
	defaultPollInterval = 2 * time.Second
	defaultRetryDelay   = 2 * time.Second
	maxRetryDelay       = 5 * time.Minute
)

type JobStore interface {
	ClaimNext(context.Context) (*models.BackgroundJob, error)
	Complete(context.Context, string) error
	RetryOrFail(
		context.Context,
		models.BackgroundJob,
		error,
		time.Duration,
	) (bool, error)
	FailExhaustedStale(context.Context) (int64, error)
}

type ConfirmationSender interface {
	SendConfirmation(
		context.Context,
		models.BackgroundJob,
	) error
}

type Worker struct {
	store        JobStore
	sender       ConfirmationSender
	pollInterval time.Duration
	retryDelay   time.Duration
	logger       *log.Logger
}

func NewWorker(
	store JobStore,
	sender ConfirmationSender,
) *Worker {
	return &Worker{
		store:        store,
		sender:       sender,
		pollInterval: defaultPollInterval,
		retryDelay:   defaultRetryDelay,
		logger:       log.Default(),
	}
}

// ProcessNext claims and processes at most one available job.
func (w *Worker) ProcessNext(ctx context.Context) error {
	if w.store == nil {
		return errors.New("job store is required")
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	failedCount, err := w.store.FailExhaustedStale(ctx)
	if err != nil {
		return fmt.Errorf("recover exhausted jobs: %w", err)
	}
	if failedCount > 0 {
		w.logger.Printf(
			"ALERT: %d abandoned background jobs exhausted their retries",
			failedCount,
		)
	}

	job, err := w.store.ClaimNext(ctx)
	if err != nil {
		return fmt.Errorf("claim background job: %w", err)
	}
	if job == nil {
		return nil
	}

	var processingErr error

	switch job.JobType {
	case "confirmation_email":
		if w.sender == nil {
			processingErr = errors.New("confirmation sender is not configured")
		} else {
			processingErr = w.sender.SendConfirmation(ctx, *job)
		}

	default:
		processingErr = fmt.Errorf(
			"unsupported background job type %q",
			job.JobType,
		)
	}

	if processingErr != nil {
		delay := retryBackoff(job.Attempts, w.retryDelay)

		permanentlyFailed, retryErr := w.store.RetryOrFail(
			ctx,
			*job,
			processingErr,
			delay,
		)
		if retryErr != nil {
			return fmt.Errorf(
				"schedule retry for job %s: %w",
				job.ID,
				retryErr,
			)
		}

		if permanentlyFailed {
			w.logger.Printf(
				"ALERT: background job permanently failed id=%s attempts=%d error=%v",
				job.ID,
				job.Attempts,
				processingErr,
			)
		} else {
			w.logger.Printf(
				"background job failed; retry scheduled id=%s attempt=%d",
				job.ID,
				job.Attempts,
			)
		}

		// A failed side effect is handled by the retry policy. It must not
		// affect the original submission that was already persisted.
		return nil
	}

	if err := ctx.Err(); err != nil {
		// Leave the job running so the repository's stale-lock recovery
		// can recover it if cancellation interrupts completion.
		return err
	}

	if err := w.store.Complete(ctx, job.ID); err != nil {
		return fmt.Errorf("complete background job %s: %w", job.ID, err)
	}

	w.logger.Printf("background job succeeded id=%s", job.ID)
	return nil
}

func (w *Worker) Run(ctx context.Context) error {
	if w.store == nil {
		return errors.New("job store is required")
	}

	interval := w.pollInterval
	if interval <= 0 {
		interval = defaultPollInterval
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := w.ProcessNext(ctx); err != nil {
			if ctx.Err() != nil {
				return nil
			}

			w.logger.Printf("background worker error: %v", err)
		}

		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}

func retryBackoff(attempts int, base time.Duration) time.Duration {
	if base <= 0 {
		base = defaultRetryDelay
	}

	if attempts < 1 {
		attempts = 1
	}

	// Cap the exponent to prevent duration overflow.
	exponent := int(math.Min(float64(attempts-1), 8))
	delay := base * time.Duration(1<<exponent)

	if delay > maxRetryDelay {
		return maxRetryDelay
	}
	return delay
}
