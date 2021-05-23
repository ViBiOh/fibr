package provider

import (
	"reflect"
	"testing"

	"github.com/ViBiOh/httputils/v4/pkg/renderer"
)

var (
	publicURL = "http://localhost:1080"
)

func TestBuild(t *testing.T) {
	config := Config{
		PublicURL: publicURL,
		Seo: Seo{
			Description: "fibr",
			Title:       "fibr",
		},
	}

	var cases = []struct {
		intention string
		config    Config
		request   Request
		message   renderer.Message
		layout    string
		content   map[string]interface{}
		want      Page
	}{
		{
			"default layout",
			Config{},
			Request{},
			renderer.Message{},
			"",
			nil,
			Page{
				Layout: "grid",
			},
		},
		{
			"compute metadata",
			config,
			Request{},
			renderer.Message{},
			"list",
			nil,
			Page{
				Config:      config,
				Layout:      "list",
				PublicURL:   publicURL,
				Title:       "fibr",
				Description: "fibr",
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			result := (&PageBuilder{}).Config(testCase.config).Request(testCase.request).Message(testCase.message).Layout(testCase.layout).Content(testCase.content).Build()

			if !reflect.DeepEqual(result, testCase.want) {
				t.Errorf("Build() = %#v, want %#v", result, testCase.want)
			}
		})
	}
}

func TestComputePublicURL(t *testing.T) {
	var cases = []struct {
		intention string
		config    Config
		request   Request
		want      string
	}{
		{
			"simple",
			Config{
				PublicURL: publicURL,
			},
			Request{},
			publicURL,
		},
		{
			"with request",
			Config{
				PublicURL: publicURL,
			},
			Request{
				Path: "/photos",
			},
			"http://localhost:1080/photos",
		},
		{
			"with relative request",
			Config{
				PublicURL: publicURL,
			},
			Request{
				Path: "photos",
			},
			"http://localhost:1080/photos",
		},
		{
			"with share",
			Config{
				PublicURL: "https://localhost:1080",
			},
			Request{
				Path: "/photos",
				Share: Share{
					ID: "abcd1234",
				},
			},
			"https://localhost:1080/abcd1234/photos",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			if result := computePublicURL(testCase.config, testCase.request); result != testCase.want {
				t.Errorf("computePublicURL() = `%s`, want `%s`", result, testCase.want)
			}
		})
	}
}

func TestComputeTitle(t *testing.T) {
	var cases = []struct {
		intention string
		config    Config
		request   Request
		want      string
	}{
		{
			"simple",
			Config{
				Seo: Seo{
					Title: "fibr",
				},
			},
			Request{},
			"fibr",
		},
		{
			"without share",
			Config{
				Seo: Seo{
					Title: "fibr",
				},
			},
			Request{
				Path: "/subDir/",
			},
			"fibr - subDir",
		},
		{
			"with share",
			Config{
				Seo: Seo{
					Title: "fibr",
				},
			},
			Request{
				Path: "/",
				Share: Share{
					ID:       "a1b2c3d4",
					RootName: "abcd1234",
				},
			},
			"fibr - abcd1234",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			if result := computeTitle(testCase.config, testCase.request); result != testCase.want {
				t.Errorf("computeTitle() = `%s`, want `%s`", result, testCase.want)
			}
		})
	}
}

func TestComputeDescription(t *testing.T) {
	var cases = []struct {
		intention string
		config    Config
		request   Request
		want      string
	}{
		{
			"simple",
			Config{
				Seo: Seo{
					Description: "fibr",
				},
			},
			Request{},
			"fibr",
		},
		{
			"without share",
			Config{
				Seo: Seo{
					Description: "fibr",
				},
			},
			Request{
				Path: "/subDir/",
			},
			"fibr - subDir",
		},
		{
			"with share",
			Config{
				Seo: Seo{
					Description: "fibr",
				},
			},
			Request{
				Path: "/",
				Share: Share{
					ID:       "a1b2c3d4",
					RootName: "abcd1234",
				},
			},
			"fibr - abcd1234",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			if result := computeDescription(testCase.config, testCase.request); result != testCase.want {
				t.Errorf("computeDescription() = `%s`, want `%s`", result, testCase.want)
			}
		})
	}
}
