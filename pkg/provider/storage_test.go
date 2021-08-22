package provider

import "testing"

func TestExtension(t *testing.T) {
	var cases = []struct {
		intention string
		input     StorageItem
		want      string
	}{
		{
			"simple",
			StorageItem{
				Name: "test.TXT",
			},
			".txt",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			if result := testCase.input.Extension(); result != testCase.want {
				t.Errorf("Extension() = `%s`, want `%s`", result, testCase.want)
			}
		})
	}
}

func TestMime(t *testing.T) {
	var cases = []struct {
		intention string
		input     StorageItem
		want      string
	}{
		{
			"empty",
			StorageItem{
				Name: "test",
			},
			"",
		},
		{
			"simple",
			StorageItem{
				Name: "test.TXT",
			},
			"text/plain; charset=utf-8",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			if result := testCase.input.Mime(); result != testCase.want {
				t.Errorf("Mime() = `%s`, want `%s`", result, testCase.want)
			}
		})
	}
}

func TestIsPdf(t *testing.T) {
	var cases = []struct {
		intention string
		input     StorageItem
		want      bool
	}{
		{
			"simple",
			StorageItem{
				Name: "test.pdf",
			},
			true,
		},
		{
			"raw image",
			StorageItem{
				Name: "test.raw",
			},
			false,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			if result := testCase.input.IsPdf(); result != testCase.want {
				t.Errorf("IsPdf() = `%v`, want `%v`", result, testCase.want)
			}
		})
	}
}

func TestIsImage(t *testing.T) {
	var cases = []struct {
		intention string
		input     StorageItem
		want      bool
	}{
		{
			"simple",
			StorageItem{
				Name: "test.png",
			},
			true,
		},
		{
			"raw image",
			StorageItem{
				Name: "test.raw",
			},
			false,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			if result := testCase.input.IsImage(); result != testCase.want {
				t.Errorf("IsImage() = `%v`, want `%v`", result, testCase.want)
			}
		})
	}
}

func TestIsVideo(t *testing.T) {
	var cases = []struct {
		intention string
		input     StorageItem
		want      bool
	}{
		{
			"simple",
			StorageItem{
				Name: "test.mov",
			},
			true,
		},
		{
			"old video",
			StorageItem{
				Name: "test.divx",
			},
			false,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.intention, func(t *testing.T) {
			if result := testCase.input.IsVideo(); result != testCase.want {
				t.Errorf("IsVideo() = `%v`, want `%v`", result, testCase.want)
			}
		})
	}
}

func TestDir(t *testing.T) {
	var cases = []struct {
		intention string
		instance  StorageItem
		want      string
	}{
		{
			"simple",
			StorageItem{
				Pathname: "/parent/test.mov",
			},
			"/parent",
		},
		{
			"directory",
			StorageItem{
				Pathname: "/parent",
				IsDir:    true,
			},
			"/parent",
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if got := tc.instance.Dir(); got != tc.want {
				t.Errorf("Dir() = `%s`, want `%s`", got, tc.want)
			}
		})
	}
}
