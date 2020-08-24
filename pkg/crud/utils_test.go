package crud

import (
	"net/http"
	"strings"
	"testing"

	"github.com/ViBiOh/fibr/pkg/provider"
)

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
				folderName: "/templates",
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
			"/templates",
			nil,
		},
		{
			"valid",
			args{
				folderName: "/templates",
			},
			"/templates",
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
