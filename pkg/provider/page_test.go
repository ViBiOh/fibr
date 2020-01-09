package provider

import (
	"reflect"
	"testing"
)

func TestBuild(t *testing.T) {
	config := Config{
		PublicURL: "http://localhost:1080",
		RootName:  "test",
		Seo: Seo{
			Description: "fibr",
			Title:       "fibr",
		},
	}

	var cases = []struct {
		intention string
		config    Config
		request   Request
		message   *Message
		error     *Error
		layout    string
		content   map[string]interface{}
		want      Page
	}{
		{
			"default layout",
			Config{},
			Request{},
			nil,
			nil,
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
			nil,
			nil,
			"list",
			nil,
			Page{
				Config:      config,
				Layout:      "list",
				PublicURL:   "http://localhost:1080",
				Title:       "fibr - test",
				Description: "fibr - test",
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			result := (&PageBuilder{}).Config(testCase.config).Request(testCase.request).Message(testCase.message).Error(testCase.error).Layout(testCase.layout).Content(testCase.content).Build()

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
				PublicURL: "http://localhost:1080",
			},
			Request{},
			"http://localhost:1080",
		},
		{
			"with request",
			Config{
				PublicURL: "http://localhost:1080",
			},
			Request{
				Path: "/photos",
			},
			"http://localhost:1080/photos",
		},
		{
			"with relative request",
			Config{
				PublicURL: "http://localhost:1080",
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
				Share: &Share{
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
				RootName: "test",
				Seo: Seo{
					Title: "fibr",
				},
			},
			Request{},
			"fibr - test",
		},
		{
			"without share",
			Config{
				RootName: "test",
				Seo: Seo{
					Title: "fibr",
				},
			},
			Request{
				Path: "/subDir/",
			},
			"fibr - test - subDir",
		},
		{
			"with share",
			Config{
				RootName: "test",
				Seo: Seo{
					Title: "fibr",
				},
			},
			Request{
				Path: "/",
				Share: &Share{
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
				RootName: "test",
				Seo: Seo{
					Description: "fibr",
				},
			},
			Request{},
			"fibr - test",
		},
		{
			"without share",
			Config{
				RootName: "test",
				Seo: Seo{
					Description: "fibr",
				},
			},
			Request{
				Path: "/subDir/",
			},
			"fibr - test - subDir",
		},
		{
			"with share",
			Config{
				RootName: "test",
				Seo: Seo{
					Description: "fibr",
				},
			},
			Request{
				Path: "/",
				Share: &Share{
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
