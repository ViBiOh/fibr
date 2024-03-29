package search

import (
	"fmt"
	"time"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func computeSize(unit string, size int64) int64 {
	switch unit {
	case "kb":
		return kilobytes * size
	case "mb":
		return megabytes * size
	case "gb":
		return gigabytes * size
	default:
		return size
	}
}

func computeSince(input time.Time, unit string, value int) time.Time {
	switch unit {
	case "days":
		return input.AddDate(0, 0, -value)

	case "months":
		output := input.AddDate(0, -value, 0)
		for input.Day() > 28 && output.Day() < 28 {
			output = output.AddDate(0, 0, -1)
		}

		return output

	case "years":
		output := input.AddDate(-value, 0, 0)
		if output.Month() > input.Month() {
			output = output.AddDate(0, 0, -1)
		}

		return output

	default:
		return input
	}
}

func computeMimes(aliases []string) []string {
	var output []string

	for _, alias := range aliases {
		switch alias {
		case "archive":
			return append(output, getKeysOfMap(provider.ArchiveExtensions)...)
		case "audio":
			return append(output, getKeysOfMap(provider.AudioExtensions)...)
		case "code":
			return append(output, getKeysOfMap(provider.CodeExtensions)...)
		case "excel":
			return append(output, getKeysOfMap(provider.ExcelExtensions)...)
		case "image":
			return append(output, getKeysOfMap(provider.ImageExtensions)...)
		case "pdf":
			return append(output, getKeysOfMap(provider.PdfExtensions)...)
		case "video":
			return append(output, getKeysOfMap(provider.VideoExtensions)...)
		case "stream":
			return append(output, getKeysOfMap(provider.StreamExtensions)...)
		case "word":
			return append(output, getKeysOfMap(provider.WordExtensions)...)
		}
	}

	return output
}

func getKeysOfMap[T any](input map[string]T) []string {
	output := make([]string, len(input))
	var i int64

	for key := range input {
		output[i] = key
		i++
	}

	return output
}

func parseDate(raw string) (time.Time, error) {
	if len(raw) == 0 {
		return time.Time{}, nil
	}

	value, err := time.Parse(isoDateLayout, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse date: %w", err)
	}

	return value, nil
}
