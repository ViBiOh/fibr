package exif

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/httpjson"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/ViBiOh/httputils/v4/pkg/request"
)

const (
	gpsLatitude  = "GPSLatitude"
	gpsLongitude = "GPSLongitude"

	publicNominatimURL      = "https://nominatim.openstreetmap.org"
	publicNominatimInterval = time.Second * 2 // nominatim is at 1req/sec, so we take an extra step
)

var (
	gpsRegex = regexp.MustCompile(`(?im)([0-9]+)\s*deg\s*([0-9]+)'\s*([0-9]+(?:\.[0-9]+)?)"\s*([N|S|W|E])`)
)

type geocode struct {
	Address   map[string]string `json:"address"`
	Latitude  string            `json:"lat"`
	Longitude string            `json:"lon"`
}

func (a app) ExtractGeocodeFor(item provider.StorageItem) {
	if !a.enabled() {
		return
	}

	if len(a.geocodeURL) == 0 {
		return
	}

	if item.IsDir {
		return
	}

	if a.HasGeocode(item) {
		return
	}

	for {
		select {
		case <-a.done:
			logger.Warn("Service is going to shutdown, not adding more geocode to the queue `%s`", item.Pathname)
			return
		case a.geocodeQueue <- item:
			a.increaseMetric("geocode", "queued")
			return
		default:
			time.Sleep(publicNominatimInterval * 2)
		}
	}
}

func (a app) processGeocodeQueue() {
	var tick <-chan time.Time
	if strings.HasPrefix(a.geocodeURL, publicNominatimURL) {
		ticker := time.NewTicker(publicNominatimInterval)
		defer ticker.Stop()
		tick = ticker.C
	}

	for item := range a.geocodeQueue {
		a.decreaseMetric("geocode", "queued")

		if tick != nil {
			<-tick
		}

		if err := a.extractAndSaveGeocoding(item); err != nil {
			logger.Error("unable to extract geocoding for `%s`: %s", item.Pathname, err)
		}
	}
}

func (a app) extractAndSaveGeocoding(item provider.StorageItem) error {
	lat, lon, err := a.getLatitudeAndLongitude(item)
	if err != nil {
		return fmt.Errorf("unable to get gps coordinate: %s", err)
	}

	var geocode geocode
	if len(lat) != 0 && len(lon) != 0 {
		geocode, err = a.getReverseGeocode(context.Background(), lat, lon)
		if err != nil {
			return fmt.Errorf("unable to reverse geocode: %s", err)
		}
	}

	if err := a.saveMetadata(item, geocodeMetadataFilename, geocode); err != nil {
		return fmt.Errorf("unable to save geocode: %s", err)
	}

	a.increaseMetric("geocode", "saved")

	return nil
}

func (a app) getLatitudeAndLongitude(item provider.StorageItem) (string, string, error) {
	geocode, err := a.loadGeocode(item)
	if err != nil {
		return "", "", fmt.Errorf("unable to load geocode: %s", err)
	}
	if len(geocode.Latitude) != 0 {
		return geocode.Latitude, geocode.Longitude, nil
	}

	exif, err := a.loadExif(item)
	if err != nil {
		return "", "", fmt.Errorf("unable to load exif: %s", err)
	}

	return extractCoordinates(exif)
}

func extractCoordinates(data map[string]interface{}) (string, string, error) {
	lat, err := getCoordinate(data, gpsLatitude)
	if err != nil {
		return "", "", fmt.Errorf("unable to parse latitude: %s", err)
	}

	if len(lat) == 0 {
		return "", "", nil
	}

	lon, err := getCoordinate(data, gpsLongitude)
	if err != nil {
		return "", "", fmt.Errorf("unable to parse longitude: %s", err)
	}

	return lat, lon, nil
}

func getCoordinate(data map[string]interface{}, key string) (string, error) {
	rawCoordinate, ok := data[key]
	if !ok {
		return "", nil
	}

	coordinateStr, ok := rawCoordinate.(string)
	if !ok {
		return "", fmt.Errorf("key `%s` is not a string", key)
	}

	coordinate, err := convertDegreeMinuteSecondToDecimal(coordinateStr)
	if err != nil {
		return "", fmt.Errorf("unable to parse `%s` with value `%s`: %s", key, coordinateStr, err)
	}

	return coordinate, nil
}

func convertDegreeMinuteSecondToDecimal(location string) (string, error) {
	matches := gpsRegex.FindAllStringSubmatch(location, -1)
	if len(matches) == 0 {
		return "", fmt.Errorf("unable to parse GPS data `%s`", location)
	}

	match := matches[0]

	degrees, err := strconv.ParseFloat(match[1], 16)
	if err != nil {
		return "", fmt.Errorf("unable to parse GPS degrees: %s", err)
	}

	minutes, err := strconv.ParseFloat(match[2], 16)
	if err != nil {
		return "", fmt.Errorf("unable to parse GPS minutes: %s", err)
	}

	seconds, err := strconv.ParseFloat(match[3], 16)
	if err != nil {
		return "", fmt.Errorf("unable to parse GPS seconds: %s", err)
	}

	direction := match[4]

	dd := degrees + minutes/60.0 + seconds/3600.0

	if direction == "S" || direction == "W" {
		dd *= -1
	}

	return fmt.Sprintf("%.6f", dd), nil
}

func (a app) getReverseGeocode(ctx context.Context, lat, lon string) (geocode, error) {
	params := url.Values{}
	params.Set("lat", lat)
	params.Set("lon", lon)
	params.Set("format", "json")
	params.Set("zoom", "18")

	reverseURL := fmt.Sprintf("%s/reverse?%s", a.geocodeURL, params.Encode())

	a.increaseMetric("geocode", "requested")

	resp, err := request.New().Header("User-Agent", "fibr, reverse geocoding from exif data").Get(reverseURL).Send(ctx, nil)
	if err != nil {
		return geocode{}, fmt.Errorf("unable to reverse geocode from API: %s", err)
	}

	var data geocode
	if err := httpjson.Read(resp, &data); err != nil {
		return data, fmt.Errorf("unable to decode reverse geocoding: %s", err)
	}

	return data, nil
}
