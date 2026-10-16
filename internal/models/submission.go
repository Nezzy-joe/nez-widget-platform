package models

import (
	"encoding/json"
	"time"
)

type Submission struct {
	ID        string          `json:"id"`
	TenantID  string          `json:"-"`
	WidgetID  string          `json:"widget_id"`
	Data      json.RawMessage `json:"data"`
	IPAddress string          `json:"-"`
	Country   string          `json:"country,omitempty"`
	City      string          `json:"city,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}
