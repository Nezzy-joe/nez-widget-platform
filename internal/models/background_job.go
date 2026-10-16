package models

import (
	"encoding/json"
	"time"
)

// BackgroundJob represents a durable task queued for asynchronous processing.
type BackgroundJob struct {
	ID           string
	SubmissionID string
	JobType      string
	Status       string
	Attempts     int
	MaxAttempts  int
	RunAfter     time.Time
	LockedAt     *time.Time
	LastError    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CompletedAt  *time.Time

	// SubmissionData is loaded from the related submission when a job is claimed.
	// It is not a column in background_jobs.
	SubmissionData json.RawMessage
}
