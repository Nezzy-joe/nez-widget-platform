CREATE TABLE background_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    submission_id UUID NOT NULL
        REFERENCES submissions(id) ON DELETE CASCADE,

    job_type TEXT NOT NULL
        CHECK (job_type IN ('confirmation_email')),

    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'running', 'succeeded', 'failed')),

    attempts INTEGER NOT NULL DEFAULT 0
        CHECK (attempts >= 0),

    max_attempts INTEGER NOT NULL DEFAULT 5
        CHECK (max_attempts BETWEEN 1 AND 20),

    run_after TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    locked_at TIMESTAMPTZ,
    last_error TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,

    -- Prevent duplicate logical jobs for one submission.
    CONSTRAINT uq_background_jobs_submission_type
        UNIQUE (submission_id, job_type)
);

-- Efficiently find jobs that are ready to run.
CREATE INDEX idx_background_jobs_pending
    ON background_jobs(run_after, created_at)
    WHERE status = 'pending';

-- Find jobs whose workers may have crashed while processing them.
CREATE INDEX idx_background_jobs_stale_lock
    ON background_jobs(locked_at)
    WHERE status = 'running';
