package exif

import (
	"flag"
	"strings"
	"testing"
)

func TestFlags(t *testing.T) {
	var cases = []struct {
		intention string
		want      string
	}{
		{
			"simple",
			"Usage of simple:\n  -geocodeURL string\n    \t[exif] Nominatim Geocode Service URL. This can leak GPS metadatas to a third-party (e.g. \"https://nominatim.openstreetmap.org\") {SIMPLE_GEOCODE_URL}\n  -uRL string\n    \t[exif] Exif Tool URL (exas) {SIMPLE_URL} (default \"http://exas:1080\")\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			fs := flag.NewFlagSet(tc.intention, flag.ContinueOnError)
			Flags(fs, "")

			var writer strings.Builder
			fs.SetOutput(&writer)
			fs.Usage()

			result := writer.String()

			if result != tc.want {
				t.Errorf("Flags() = `%s`, want `%s`", result, tc.want)
			}
		})
	}
}

func TestEnabled(t *testing.T) {
	var cases = []struct {
		intention string
		instance  app
		want      bool
	}{
		{
			"disabled",
			app{
				exifURL: "",
			},
			false,
		},
		{
			"enabled",
			app{
				exifURL: "http://exas",
			},
			true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if got := tc.instance.enabled(); got != tc.want {
				t.Errorf("Enabled() = %t, want %t", got, tc.want)
			}
		})
	}
}
