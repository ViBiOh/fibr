package exif

import (
	"context"
	"encoding/json"
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

	publicNominatimURL = "https://nominatim.openstreetmap.org"
)

var (
	gpsRegex = regexp.MustCompile(`(?im)([0-9]+)\s*deg\s*([0-9]+)'\s*([0-9]+(?:\.[0-9]+)?)"\s*([N|S|W|E])`)
)

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

	select {
	case <-a.geocodeDone:
		logger.Warn("Service is going to shutdown, not adding more geocode to the queue `%s`", item.Pathname)
		return
	default:
	}

	a.gauge.WithLabelValues("queued").Inc()
	a.geocodeQueue <- item
}

func (a app) computeGeocode() {
	var tick <-chan time.Time

	if strings.HasPrefix(a.geocodeURL, publicNominatimURL) {
		ticker := time.NewTicker(time.Second * 2) // nominatim is at 1req/sec, so we take an extra step
		defer ticker.Stop()
		tick = ticker.C
	}

	for item := range a.geocodeQueue {
		a.gauge.WithLabelValues("queued").Dec()

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

	var geocode map[string]interface{}

	if len(lat) != 0 && len(lon) != 0 {
		geocode, err = a.getReverseGeocode(context.Background(), lat, lon)
		if err != nil {
			return fmt.Errorf("unable to reverse geocode: %s", err)
		}
	}

	writer, err := a.storageApp.WriterTo(getExifPath(item, "geocode"))
	if err != nil {
		return fmt.Errorf("unable to get geocode writer: %s", err)
	}

	defer func() {
		if err := writer.Close(); err != nil {
			logger.Error("unable to close geocode file: %s", err)
		}
	}()

	if err := json.NewEncoder(writer).Encode(geocode); err != nil {
		return fmt.Errorf("unable to encode geocode: %s", err)
	}

	return nil
}

func (a app) getLatitudeAndLongitude(item provider.StorageItem) (string, string, error) {
	var data map[string]interface{}

	reader, err := a.storageApp.ReaderFrom(getExifPath(item, "geocode"))
	if err == nil {
		if err := json.NewDecoder(reader).Decode(&data); err != nil {
			return "", "", fmt.Errorf("unable to decode: %s", err)
		}

		return data["lat"].(string), data["lon"].(string), nil
	}

	if !provider.IsNotExist(err) {
		return "", "", fmt.Errorf("unable to read: %s", err)
	}

	data, err = a.get(item)
	if err != nil {
		return "", "", fmt.Errorf("unable to retrieve: %s", err)
	}

	if data == nil {
		return "", "", nil
	}

	return extractCoordinates(data)
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

func (a app) getReverseGeocode(ctx context.Context, lat, lon string) (map[string]interface{}, error) {
	params := url.Values{}
	params.Set("lat", lat)
	params.Set("lon", lon)
	params.Set("format", "json")
	params.Set("zoom", "18")

	reverseURL := fmt.Sprintf("%s/reverse?%s", a.geocodeURL, params.Encode())

	a.gauge.WithLabelValues("geocode").Inc()

	resp, err := request.New().Header("User-Agent", "fibr").Get(reverseURL).Send(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to reverse geocode from API: %s", err)
	}

	var data map[string]interface{}
	if err := httpjson.Read(resp, &data); err != nil {
		return nil, fmt.Errorf("unable to decode reverse geocoding: %s", err)
	}

	return data, nil
}
