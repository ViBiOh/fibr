package provider

import "time"

// Aggregate contains aggregated data for a folder
type Aggregate struct {
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Location string    `json:"location"`
}
