package crud

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	"github.com/ViBiOh/fibr/pkg/provider"
)

var (
	filename      = "README.md"
	otherFilename = "CONTRIBUTING.md"
	templatesPath = "/templates"
)

func TestGetPreviousAndNext(t *testing.T) {
	type args struct {
		file  provider.StorageItem
		files []provider.StorageItem
	}

	var cases = []struct {
		intention    string
		args         args
		wantPrevious *provider.StorageItem
		wantNext     *provider.StorageItem
	}{
		{
			"one item",
			args{
				file: provider.StorageItem{Name: filename},
				files: []provider.StorageItem{
					{Name: filename},
				},
			},
			nil,
			nil,
		},
		{
			"no next item",
			args{
				file: provider.StorageItem{Name: filename},
				files: []provider.StorageItem{
					{Name: ".git", IsDir: true},
					{Name: otherFilename},
					{Name: filename},
				},
			},
			&provider.StorageItem{Name: otherFilename},
			nil,
		},
		{
			"full items",
			args{
				file: provider.StorageItem{Name: filename},
				files: []provider.StorageItem{
					{Name: ".git", IsDir: true},
					{Name: otherFilename},
					{Name: filename},
					{Name: "main.go"},
				},
			},
			&provider.StorageItem{Name: otherFilename},
			&provider.StorageItem{Name: "main.go"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if gotPrevious, gotNext := getPreviousAndNext(tc.args.file, tc.args.files); !reflect.DeepEqual(gotPrevious, tc.wantPrevious) || !reflect.DeepEqual(gotNext, tc.wantNext) {
				t.Errorf("getPreviousAndNext() = (%v, %v), want (%v, %v)", gotPrevious, gotNext, tc.wantPrevious, tc.wantNext)
			}
		})
	}
}

func TestCheckFormName(t *testing.T) {
	type args struct {
		r        *http.Request
		formName string
	}

	rootValues := url.Values{
		"filename": {"/"},
	}
	rootRequest := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(rootValues.Encode()))
	rootRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	validValues := url.Values{
		"filename": {"/README.md"},
	}
	validRequest := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(validValues.Encode()))
	validRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var cases = []struct {
		intention string
		args      args
		want      string
		wantErr   error
	}{
		{
			"empty",
			args{
				r:        httptest.NewRequest(http.MethodPost, "/", nil),
				formName: "filename",
			},
			"",
			provider.NewError(http.StatusBadRequest, ErrEmptyName),
		},
		{
			"root",
			args{
				r:        rootRequest,
				formName: "filename",
			},
			"",
			provider.NewError(http.StatusForbidden, ErrNotAuthorized),
		},
		{
			"valid",
			args{
				r:        validRequest,
				formName: "filename",
			},
			"/README.md",
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			got, gotErr := checkFormName(tc.args.r, tc.args.formName)

			failed := false

			if tc.wantErr == nil && gotErr != nil {
				failed = true
			} else if tc.wantErr != nil && gotErr == nil {
				failed = true
			} else if tc.wantErr != nil && !strings.Contains(gotErr.Error(), tc.wantErr.Error()) {
				failed = true
			} else if got != tc.want {
				failed = true
			}

			if failed {
				t.Errorf("checkFormName() = (`%s`, `%s`), want (`%s`, `%s`)", got, gotErr, tc.want, tc.wantErr)
			}
		})
	}
}

func TestCheckFolderName(t *testing.T) {
	type args struct {
		folderName string
		request    provider.Request
	}

	var cases = []struct {
		intention string
		args      args
		want      string
		wantErr   *provider.Error
	}{
		{
			"empty value",
			args{
				folderName: "",
			},
			"",
			provider.NewError(http.StatusBadRequest, ErrEmptyFolder),
		},
		{
			"no prefix",
			args{
				folderName: "templates/",
			},
			"",
			provider.NewError(http.StatusBadRequest, ErrAbsoluteFolder),
		},
		{
			"share",
			args{
				folderName: templatesPath,
				request: provider.Request{
					Share: &provider.Share{
						ID: "abcdef1234",
					},
				},
			},
			"",
			provider.NewError(http.StatusForbidden, ErrNotAuthorized),
		},
		{
			"valid share",
			args{
				folderName: "/abcdef1234/templates",
				request: provider.Request{
					Share: &provider.Share{
						ID: "abcdef1234",
					},
				},
			},
			templatesPath,
			nil,
		},
		{
			"valid",
			args{
				folderName: templatesPath,
			},
			templatesPath,
			nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			got, gotErr := checkFolderName(tc.args.folderName, tc.args.request)

			failed := false

			if tc.wantErr == nil && gotErr != nil {
				failed = true
			} else if tc.wantErr != nil && gotErr == nil {
				failed = true
			} else if tc.wantErr != nil && !strings.Contains(gotErr.Error(), tc.wantErr.Error()) {
				failed = true
			} else if got != tc.want {
				failed = true
			}

			if failed {
				t.Errorf("checkFolderName() = (`%s`, `%s`), want (`%s`, `%s`)", got, gotErr, tc.want, tc.wantErr)
			}
		})
	}
}

func TestGetPathParts(t *testing.T) {
	type args struct {
		uri string
	}

	var cases = []struct {
		intention string
		args      args
		want      []string
	}{
		{
			"root",
			args{
				uri: "/",
			},
			nil,
		},
		{
			"simple",
			args{
				uri: "/hello/world/",
			},
			[]string{"hello", "world"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if got := getPathParts(tc.args.uri); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("getPathParts() = %v, want %v", got, tc.want)
			}
		})
	}
}
