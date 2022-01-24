package crud

import (
	"testing"

	"github.com/ViBiOh/fibr/pkg/mocks"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/golang/mock/gomock"
)

func TestBestSharePath(t *testing.T) {
	type args struct {
		request provider.Request
		name    string
	}

	cases := []struct {
		intention string
		instance  App
		args      args
		want      string
	}{
		{
			"already shared",
			App{},
			args{
				request: provider.Request{
					Path: "/",
					Share: provider.Share{
						ID:   "abcdef123456",
						Path: "/website",
					},
				},
				name: "index.html",
			},
			"/abcdef123456/index.html",
		},
		{
			"no share",
			App{},
			args{
				request: provider.Request{
					Path: "/website",
				},
				name: "index.html",
			},
			"/website/index.html",
		},
		{
			"matching share",
			App{},
			args{
				request: provider.Request{
					Path: "/website",
				},
				name: "index.html",
			},
			"/abcdef123456/index.html",
		},
		{
			"distance share",
			App{},
			args{
				request: provider.Request{
					Path: "/website/path/to/deep/folder",
				},
				name: "index.html",
			},
			"/abcdef123456/folder/index.html",
		},
		{
			"share with password",
			App{},
			args{
				request: provider.Request{
					Path: "/website/path/to/deep/folder",
					Share: provider.Share{
						ID:       "azerty",
						Path:     "/website",
						Password: "abcd",
					},
				},
				name: "index.html",
			},
			"/abcdef123456/folder/index.html",
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockShare := mocks.NewShare(ctrl)

			tc.instance.shareApp = mockShare

			switch tc.intention {
			case "no share":
				mockShare.EXPECT().List().Return(nil)
			case "matching share":
				mockShare.EXPECT().List().Return([]provider.Share{
					{
						ID:   "abcdef123456",
						Path: "/website",
					},
				})
			case "distance share", "share with password":
				mockShare.EXPECT().List().Return([]provider.Share{
					{
						ID:   "abcdef123456",
						Path: "/newsite/",
					},
					{
						ID:   "a1b2c3d4e5f6",
						Path: "/website/path/to",
					},
					{
						ID:   "abcdef123456",
						Path: "/website/path/to/deep/",
					},
					{
						ID:       "654321fedcba",
						Path:     "/website/path/to/deep/folder",
						Password: "secret",
					},
				})
			}

			if got := tc.instance.bestSharePath(tc.args.request, tc.args.name); got != tc.want {
				t.Errorf("bestSharePath() = `%s`, want `%s`", got, tc.want)
			}
		})
	}
}
