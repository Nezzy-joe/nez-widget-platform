package models

import (
	"encoding/json"
	"time"
)

type Widget struct {
	ID             string          `json:"id"`
	TenantID       string          `json:"-"`
	PublicID       string          `json:"public_id"`
	Type           string          `json:"type"`
	Title          string          `json:"title"`
	Description    *string         `json:"description,omitempty"`
	ButtonText     string          `json:"button_text"`
	DisplayOptions json.RawMessage `json:"display_options"`
	Fields         []WidgetField   `json:"fields,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}

type WidgetField struct {
	ID        string    `json:"id"`
	WidgetID  string    `json:"-"`
	Name      string    `json:"name"`
	Label     string    `json:"label"`
	FieldType string    `json:"field_type"`
	Required  bool      `json:"required"`
	Position  int       `json:"position"`
	CreatedAt time.Time `json:"created_at"`
}
