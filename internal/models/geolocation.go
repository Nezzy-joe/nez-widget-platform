package models

type GeoLocation struct {
	Country string `json:"country,omitempty"`
	City    string `json:"city,omitempty"`
}
