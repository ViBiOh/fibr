package crud

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/model"
)

var (
	filename      = "README.md"
	otherFilename = "CONTRIBUTING.md"
	templatesPath = "/templates"
)

func TestGetPreviousAndNext(t *testing.T) {
	type args struct {
		file  absto.Item
		files []absto.Item
	}

	cases := map[string]struct {
		args         args
		wantPrevious *absto.Item
		wantNext     *absto.Item
	}{
		"one item": {
			args{
				file: absto.Item{Name: filename},
				files: []absto.Item{
					{Name: filename},
				},
			},
			nil,
			nil,
		},
		"no next item": {
			args{
				file: absto.Item{Name: filename},
				files: []absto.Item{
					{Name: ".git", IsDir: true},
					{Name: otherFilename},
					{Name: filename},
				},
			},
			&absto.Item{Name: otherFilename},
			nil,
		},
		"full items": {
			args{
				file: absto.Item{Name: filename},
				files: []absto.Item{
					{Name: ".git", IsDir: true},
					{Name: otherFilename},
					{Name: filename},
					{Name: "main.go"},
				},
			},
			&absto.Item{Name: otherFilename},
			&absto.Item{Name: "main.go"},
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
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
	rootRequest.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	validValues := url.Values{
		"filename": {"/README.md"},
	}
	validRequest := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(validValues.Encode()))
	validRequest.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	cases := map[string]struct {
		args    args
		want    string
		wantErr error
	}{
		"empty": {
			args{
				r:        httptest.NewRequest(http.MethodPost, "/", nil),
				formName: "filename",
			},
			"",
			model.WrapInvalid(ErrEmptyName),
		},
		"root": {
			args{
				r:        rootRequest,
				formName: "filename",
			},
			"",
			model.WrapForbidden(ErrNotAuthorized),
		},
		"valid": {
			args{
				r:        validRequest,
				formName: "filename",
			},
			"/README.md",
			nil,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
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
	}

	cases := map[string]struct {
		args    args
		want    string
		wantErr error
	}{
		"empty value": {
			args{
				folderName: "",
			},
			"",
			model.WrapInvalid(ErrEmptyFolder),
		},
		"no prefix": {
			args{
				folderName: "templates/",
			},
			"",
			model.WrapInvalid(ErrAbsoluteFolder),
		},
		"valid": {
			args{
				folderName: templatesPath,
			},
			templatesPath,
			nil,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			got, gotErr := checkFolderName(tc.args.folderName)

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
		request provider.Request
	}

	cases := map[string]struct {
		args args
		want []string
	}{
		"root": {
			args{
				request: provider.Request{
					Path: "/",
				},
			},
			nil,
		},
		"simple": {
			args{
				request: provider.Request{
					Path: "/hello/world/",
				},
			},
			[]string{"hello", "world"},
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			if got := getPathParts(tc.args.request); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("getPathParts() = %v, want %v", got, tc.want)
			}
		})
	}
}
