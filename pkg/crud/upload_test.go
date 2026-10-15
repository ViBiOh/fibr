package crud

import (
	"errors"
	"testing"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func TestGetUploadNameAndPath(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		request   provider.Request
		inputName string
		wantName  string
		wantPath  string
		wantErr   error
	}{
		"regular name": {
			provider.Request{Path: "/holidays"},
			"Photo 1.JPG",
			"photo_1.jpg",
			"/holidays/photo_1.jpg",
			nil,
		},
		"name without any supported char": {
			provider.Request{Path: "/holidays"},
			"日本語",
			"",
			"",
			provider.ErrInvalidName,
		},
		"single dot": {
			provider.Request{Path: "/holidays"},
			".",
			"",
			"",
			provider.ErrInvalidName,
		},
	}

	for intention, testCase := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			gotName, gotPath, gotErr := getUploadNameAndPath(testCase.request, testCase.inputName, nil)

			if !errors.Is(gotErr, testCase.wantErr) {
				t.Errorf("getUploadNameAndPath() error = %v, want %v", gotErr, testCase.wantErr)
			}

			if gotName != testCase.wantName {
				t.Errorf("getUploadNameAndPath() name = `%s`, want `%s`", gotName, testCase.wantName)
			}

			if gotPath != testCase.wantPath {
				t.Errorf("getUploadNameAndPath() path = `%s`, want `%s`", gotPath, testCase.wantPath)
			}
		})
	}
}
