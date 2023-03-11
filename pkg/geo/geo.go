package geo

import (
	"fmt"
)

type Type string

const (
	// TypeFeature as defined in https://datatracker.ietf.org/doc/html/rfc7946#section-1.4
	TypeFeature Type = "Feature"
	// TypePoint as defined in https://datatracker.ietf.org/doc/html/rfc7946#section-1.4
	TypePoint Type = "Point"
)

type Feature struct {
	Properties map[string]any `json:"properties"`
	Geometry   any            `json:"geometry"`
	Type       Type           `json:"type"`
}

func NewFeature(geometry any, properties map[string]any) Feature {
	return Feature{
		Type:       TypeFeature,
		Geometry:   geometry,
		Properties: properties,
	}
}

type Point struct {
	Type        Type     `json:"type"`
	Coordinates Position `json:"coordinates"`
}

func NewPoint(position Position) Point {
	return Point{
		Type:        TypePoint,
		Coordinates: position,
	}
}

type Position struct {
	Longitude float64
	Latitude  float64
}

func NewPosition(lon, lat float64) Position {
	return Position{
		Longitude: lon,
		Latitude:  lat,
	}
}

// MarshalJSON marshals the position as an array.
func (p Position) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("[%f,%f]", p.Longitude, p.Latitude)), nil
}
