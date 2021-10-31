package filesystem

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/ViBiOh/fibr/pkg/provider"
)

func TestCheckPathname(t *testing.T) {
	type args struct {
		pathname string
	}

	cases := []struct {
		intention string
		args      args
		want      error
	}{
		{
			"valid",
			args{
				pathname: "/test",
			},
			nil,
		},
		{
			"invalid",
			args{
				pathname: "/test/../root",
			},
			ErrRelativePath,
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if got := checkPathname(tc.args.pathname); got != tc.want {
				t.Errorf("checkPathname() = %t, want %t", got, tc.want)
			}
		})
	}
}

func TestGetFullPath(t *testing.T) {
	type args struct {
		pathname string
	}

	cases := []struct {
		intention string
		instance  App
		args      args
		want      string
	}{
		{
			"simple",
			App{
				rootDirectory: "/home/users",
			},
			args{
				pathname: "/test",
			},
			"/home/users/test",
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if got := tc.instance.path(tc.args.pathname); got != tc.want {
				t.Errorf("getFullPath() = `%s`, want `%s`", got, tc.want)
			}
		})
	}
}

func TestGetRelativePath(t *testing.T) {
	type args struct {
		pathname string
	}

	cases := []struct {
		intention string
		instance  App
		args      args
		want      string
	}{
		{
			"simple",
			App{
				rootDirectory: "/home/users",
			},
			args{
				pathname: "/home/users/test",
			},
			"/test",
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if got := tc.instance.getRelativePath(tc.args.pathname); got != tc.want {
				t.Errorf("getRelativePath() = `%s`, want `%s`", got, tc.want)
			}
		})
	}
}

func TestGetMode(t *testing.T) {
	type args struct {
		name string
	}

	cases := []struct {
		intention string
		args      args
		want      os.FileMode
	}{
		{
			"directory",
			args{
				name: "/photos/",
			},
			0o700,
		},
		{
			"file",
			args{
				name: "/photo.png",
			},
			0o600,
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if got := getMode(tc.args.name); got != tc.want {
				t.Errorf("getMode() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestConvertToItem(t *testing.T) {
	type args struct {
		pathname string
		info     os.FileInfo
	}

	readmeInfo, err := os.Stat("../../README.md")
	if err != nil {
		t.Error(err)
	}

	cases := []struct {
		intention string
		args      args
		want      provider.StorageItem
	}{
		{
			"simple",
			args{
				pathname: "/README.md",
				info:     readmeInfo,
			},
			provider.StorageItem{
				Name:     "README.md",
				Pathname: "/README.md",
				IsDir:    false,
				Date:     readmeInfo.ModTime(),
				Size:     readmeInfo.Size(),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if got := convertToItem(tc.args.pathname, tc.args.info); got != tc.want {
				t.Errorf("convertToItem() = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestConvertError(t *testing.T) {
	type args struct {
		err error
	}

	cases := []struct {
		intention string
		args      args
		want      error
	}{
		{
			"nil",
			args{
				err: nil,
			},
			nil,
		},
		{
			"not exist",
			args{
				err: os.ErrNotExist,
			},
			provider.ErrNotExist(os.ErrNotExist),
		},
		{
			"standard",
			args{
				err: errors.New("unable to read"),
			},
			errors.New("unable to read"),
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			failed := false
			got := convertError(tc.args.err)

			if tc.want == nil && got != nil {
				failed = true
			} else if tc.want != nil && got == nil {
				failed = true
			} else if tc.want != nil && !strings.Contains(got.Error(), tc.want.Error()) {
				failed = true
			}

			if failed {
				t.Errorf("convertError() = %v, want %v", got, tc.want)
			}
		})
	}
}
