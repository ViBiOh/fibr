package crud

import (
	"testing"

	"github.com/ViBiOh/fibr/pkg/mocks"
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/golang/mock/gomock"
)

func TestBestSharePath(t *testing.T) {
	type args struct {
		pathname string
	}

	cases := map[string]struct {
		instance App
		args     args
		want     string
	}{
		"no share": {
			App{},
			args{
				pathname: "/website/index.html",
			},
			"",
		},
		"matching share": {
			App{},
			args{
				pathname: "/website/index.html",
			},
			"/abcdef123456/index.html",
		},
		"distance share": {
			App{},
			args{
				pathname: "/website/path/to/deep/folder/index.html",
			},
			"/abcdef123456/folder/index.html",
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockShare := mocks.NewShareManager(ctrl)

			tc.instance.shareApp = mockShare

			switch intention {
			case "no share":
				mockShare.EXPECT().List().Return(nil)
			case "matching share":
				mockShare.EXPECT().List().Return([]provider.Share{
					{
						ID:   "abcdef123456",
						Path: "/website",
					},
				})
			case "distance share":
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

			if got := tc.instance.bestSharePath(tc.args.pathname); got != tc.want {
				t.Errorf("bestSharePath() = `%s`, want `%s`", got, tc.want)
			}
		})
	}
}
