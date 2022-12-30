package provider

import (
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	exas "github.com/ViBiOh/exas/pkg/model"
)

type Metadata struct {
	Description string `json:"description,omitempty"`
	exas.Exif
}

type Aggregate struct {
	Start    time.Time `json:"start,omitempty"`
	End      time.Time `json:"end,omitempty"`
	Location string    `json:"location,omitempty"`
	Cover    string    `json:"cover,omitempty"`
}

type ExifResponse struct {
	Exif exas.Exif  `json:"exif"`
	Item absto.Item `json:"item"`
}
