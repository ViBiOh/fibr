package geo

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
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
	Type       Type                   `json:"type"`
	Geometry   interface{}            `json:"geometry"`
	Properties map[string]interface{} `json:"properties"`
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

// UnmarshalJSON unmarshal Position
func (p *Position) UnmarshalJSON(b []byte) (err error) {
	if len(b) == 0 {
		return errors.New("invalid empty Position")
	}

	strValue := string(b)
	if !strings.HasPrefix(strValue, "[") || !strings.HasSuffix(strValue, "]") {
		err = errors.New("Position is not an array of float")
		return
	}

	parts := strings.Split(strValue[1:len(strValue)-1], ",")
	if len(parts) < 2 {
		err = errors.New("Position must contains at least 2 float")
		return
	}

	p.Longitude, err = strconv.ParseFloat(parts[0], 64)
	if err != nil {
		err = fmt.Errorf("unable to parse Position's longitude: %s", err)
		return
	}

	p.Longitude, err = strconv.ParseFloat(parts[1], 64)
	if err != nil {
		err = fmt.Errorf("unable to parse Position's latitude: %s", err)
		return
	}

	if len(parts) == 3 {
		p.Altitude, err = strconv.ParseFloat(parts[2], 64)
		if err != nil {
			err = fmt.Errorf("unable to parse Position's altitude: %s", err)
			return
		}
	}

	return
}
