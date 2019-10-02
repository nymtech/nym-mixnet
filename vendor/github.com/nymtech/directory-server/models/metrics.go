package models

// MixMetric is a report from each mixnode detailing recent traffic.
// Useful for creating visualisations.
type MixMetric struct {
	PubKey   string          `json:"pubKey" binding:"required"`
	Sent     map[string]uint `json:"sent" binding:"required"`
	Received uint            `json:"received" binding:"required"`
}
