package provider

import "testing"

func TestComputePublicURL(t *testing.T) {
	var cases = []struct {
		intention string
		config    *Config
		request   *Request
		want      string
	}{
		{
			"simple",
			&Config{
				PublicURL: "http://localhost:1080",
			},
			nil,
			"http://localhost:1080",
		},
		{
			"with request",
			&Config{
				PublicURL: "http://localhost:1080",
			},
			&Request{
				Path: "/photos",
			},
			"http://localhost:1080/photos",
		},
		{
			"with relative request",
			&Config{
				PublicURL: "http://localhost:1080",
			},
			&Request{
				Path: "photos",
			},
			"http://localhost:1080/photos",
		},
		{
			"with share",
			&Config{
				PublicURL: "https://localhost:1080",
			},
			&Request{
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
				t.Errorf("computePublicURL(%#v, %#v) = `%s`, want `%s`", testCase.config, testCase.request, result, testCase.want)
			}
		})
	}
}

func TestComputeTitle(t *testing.T) {
	var cases = []struct {
		intention string
		config    *Config
		request   *Request
		want      string
	}{
		{
			"simple",
			&Config{
				RootName: "test",
				Seo: &Seo{
					Title: "fibr",
				},
			},
			nil,
			"fibr - test",
		},
		{
			"without share",
			&Config{
				RootName: "test",
				Seo: &Seo{
					Title: "fibr",
				},
			},
			&Request{
				Path: "/subDir/",
			},
			"fibr - test - subDir",
		},
		{
			"with share",
			&Config{
				RootName: "test",
				Seo: &Seo{
					Title: "fibr",
				},
			},
			&Request{
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
				t.Errorf("computeTitle(%#v, %#v) = `%s`, want `%s`", testCase.config, testCase.request, result, testCase.want)
			}
		})
	}
}

func TestComputeDescription(t *testing.T) {
	var cases = []struct {
		intention string
		config    *Config
		request   *Request
		want      string
	}{
		{
			"simple",
			&Config{
				RootName: "test",
				Seo: &Seo{
					Description: "fibr",
				},
			},
			nil,
			"fibr - test",
		},
		{
			"without share",
			&Config{
				RootName: "test",
				Seo: &Seo{
					Description: "fibr",
				},
			},
			&Request{
				Path: "/subDir/",
			},
			"fibr - test - subDir",
		},
		{
			"with share",
			&Config{
				RootName: "test",
				Seo: &Seo{
					Description: "fibr",
				},
			},
			&Request{
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
				t.Errorf("computeDescription(%#v, %#v) = `%s`, want `%s`", testCase.config, testCase.request, result, testCase.want)
			}
		})
	}
}
