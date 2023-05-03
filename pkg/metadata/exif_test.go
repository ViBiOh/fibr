package metadata

import (
	"flag"
	"strings"
	"testing"

	"github.com/ViBiOh/httputils/v4/pkg/request"
)

func TestFlags(t *testing.T) {
	cases := map[string]struct {
		want string
	}{
		"simple": {
			"Usage of simple:\n  -amqpExchange string\n    \t[exif] AMQP Exchange Name ${SIMPLE_AMQP_EXCHANGE} (default \"fibr\")\n  -amqpRoutingKey string\n    \t[exif] AMQP Routing Key for exif ${SIMPLE_AMQP_ROUTING_KEY} (default \"exif_input\")\n  -directAccess\n    \t[exif] Use Exas with direct access to filesystem (no large file upload, send a GET request, Basic Auth recommended) ${SIMPLE_DIRECT_ACCESS}\n  -maxSize int\n    \t[exif] Max file size (in bytes) for extracting exif (0 to no limit). Not used if DirectAccess enabled. ${SIMPLE_MAX_SIZE} (default 209715200)\n  -password string\n    \t[exif] Exif Tool URL Basic Password ${SIMPLE_PASSWORD}\n  -uRL string\n    \t[exif] Exif Tool URL (exas) ${SIMPLE_URL} (default \"http://exas:1080\")\n  -user string\n    \t[exif] Exif Tool URL Basic User ${SIMPLE_USER}\n",
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			fs := flag.NewFlagSet(intention, flag.ContinueOnError)
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
	cases := map[string]struct {
		instance App
		want     bool
	}{
		"disabled": {
			App{},
			false,
		},
		"enabled": {
			App{
				exifRequest: request.New().URL("http://localhost"),
			},
			true,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			if got := tc.instance.enabled(); got != tc.want {
				t.Errorf("Enabled() = %t, want %t", got, tc.want)
			}
		})
	}
}
