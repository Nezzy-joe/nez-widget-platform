package models

// DailySubmissionCount represents submission activity for one calendar day.
type DailySubmissionCount struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// SubmissionAnalytics contains tenant-scoped dashboard metrics.
// The 7-day and 30-day counts use calendar days, including today.
type SubmissionAnalytics struct {
	TotalSubmissions int                    `json:"total_submissions"`
	Last7Days        int                    `json:"submissions_last_7_days"`
	Last30Days       int                    `json:"submissions_last_30_days"`
	Daily            []DailySubmissionCount `json:"daily"`
}
