package jobs

import (
	"context"
	"errors"
	"log"

	"github.com/Nezzy-joe/nez-widget-platform/internal/models"
)

// ConsoleConfirmationSender simulates notification delivery for local
// development. It never logs submitted form values or email addresses.
type ConsoleConfirmationSender struct {
	logger *log.Logger
}

func NewConsoleConfirmationSender(logger *log.Logger) *ConsoleConfirmationSender {
	if logger == nil {
		logger = log.Default()
	}
	return &ConsoleConfirmationSender{logger: logger}
}

func (s *ConsoleConfirmationSender) SendConfirmation(
	ctx context.Context,
	job models.BackgroundJob,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if job.ID == "" || job.SubmissionID == "" {
		return errors.New("job and submission IDs are required")
	}

	s.logger.Printf(
		"SIMULATED_CONFIRMATION_EMAIL job_id=%s submission_id=%s",
		job.ID,
		job.SubmissionID,
	)

	return nil
}
