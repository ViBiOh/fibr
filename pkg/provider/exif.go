package provider

import "time"

// Aggregate contains aggregated data for a folder
type Aggregate struct {
	Location string    `json:"location"`
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
}
