package provider

import (
	"time"

	"github.com/ViBiOh/exas/pkg/model"
)

// Aggregate contains aggregated data for a folder
type Aggregate struct {
	Start    time.Time `json:"start"`
	End      time.Time `json:"end"`
	Location string    `json:"location"`
}

// ExifResponse from AMQP
type ExifResponse struct {
	Item StorageItem `json:"item"`
	Exif model.Exif  `json:"exif"`
}
