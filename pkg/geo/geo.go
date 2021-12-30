package geo

import (
	"bytes"
	"fmt"
)

// Type is a type of GeoJSON Object
type Type string

const (
	// TypeFeatureCollection as defined in https://datatracker.ietf.org/doc/html/rfc7946#section-1.4
	TypeFeatureCollection Type = "FeatureCollection"
	// TypeFeature as defined in https://datatracker.ietf.org/doc/html/rfc7946#section-1.4
	TypeFeature Type = "Feature"
	// TypePoint as defined in https://datatracker.ietf.org/doc/html/rfc7946#section-1.4
	TypePoint Type = "Point"
)

// FeatureCollection description
type FeatureCollection struct {
	Type     Type      `json:"type"`
	Features []Feature `json:"features"`
}

// NewFeatureCollection creates a FeatureCollection from given features
func NewFeatureCollection(features []Feature) FeatureCollection {
	return FeatureCollection{
		Type:     TypeFeatureCollection,
		Features: features,
	}
}

// Feature description
type Feature struct {
	Properties map[string]interface{} `json:"properties"`
	Geometry   interface{}            `json:"geometry"`
	Type       Type                   `json:"type"`
}

// NewFeature creates a Feature from given geometry and properties
func NewFeature(geometry interface{}, properties map[string]interface{}) Feature {
	return Feature{
		Type:       TypeFeature,
		Geometry:   geometry,
		Properties: properties,
	}
}

// Point description
type Point struct {
	Type        Type     `json:"type"`
	Coordinates Position `json:"coordinates"`
}

// NewPoint creates a Point from given position
func NewPoint(position Position) Point {
	return Point{
		Type:        TypePoint,
		Coordinates: position,
	}
}

// Position description
type Position struct {
	Longitude float64
	Latitude  float64
	Altitude  float64
}

// NewPosition creates a new position
func NewPosition(lon, lat, alt float64) Position {
	return Position{
		Longitude: lon,
		Latitude:  lat,
		Altitude:  alt,
	}
}

// MarshalJSON marshals the position as an array
func (p Position) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`[`)

	fmt.Fprintf(buffer, "%f,%f", p.Longitude, p.Latitude)
	if p.Altitude != 0 {
		fmt.Fprintf(buffer, ",%f", p.Altitude)
	}

	buffer.WriteString(`]`)

	return buffer.Bytes(), nil
}
