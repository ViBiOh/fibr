package provider

import (
	"testing"

	absto "github.com/ViBiOh/absto/pkg/model"
)

func TestAbsoluteURL(t *testing.T) {
	type args struct {
		name string
	}

	cases := []struct {
		intention string
		instance  Request
		args      args
		want      string
	}{
		{
			"simple",
			Request{
				Path: "/",
			},
			args{
				name: "index.html",
			},
			"/index.html",
		},
		{
			"dir",
			Request{
				Path: "/",
			},
			args{
				name: "folder/",
			},
			"/folder/",
		},
		{
			"share",
			Request{
				Path: "/",
				Share: Share{
					ID:   "abcdef123456",
					Path: "/folder",
				},
			},
			args{
				name: "index.html",
			},
			"/abcdef123456/index.html",
		},
	}
	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if got := tc.instance.AbsoluteURL(tc.args.name); got != tc.want {
				t.Errorf("AbsoluteURL() = `%s`, want `%s`", got, tc.want)
			}
		})
	}
}

func TestRelativeURL(t *testing.T) {
	type args struct {
		item absto.Item
	}

	cases := []struct {
		intention string
		instance  Request
		args      args
		want      string
	}{
		{
			"simple",
			Request{
				Path: "/",
			},
			args{
				item: absto.Item{
					Pathname: "/index.html",
				},
			},
			"index.html",
		},
		{
			"dir",
			Request{
				Path: "/",
			},
			args{
				item: absto.Item{
					Pathname: "/folder",
					IsDir:    true,
				},
			},
			"folder/",
		},
		{
			"share",
			Request{
				Path: "/subpath/",
				Share: Share{
					ID:   "abcdef123456",
					Path: "/folder/",
				},
			},
			args{
				item: absto.Item{
					Pathname: "/folder/subpath/index.html",
				},
			},
			"index.html",
		},
		{
			"nested folder",
			Request{
				Path: "/sub/folder/",
			},
			args{
				item: absto.Item{
					Pathname: "/sub/folder/index.html",
				},
			},
			"index.html",
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if got := tc.instance.RelativeURL(tc.args.item); got != tc.want {
				t.Errorf("RelativeURL() = `%s`, want `%s`", got, tc.want)
			}
		})
	}
}

func TestFilepath(t *testing.T) {
	type args struct {
		name string
	}

	cases := []struct {
		intention string
		request   Request
		args      args
		want      string
	}{
		{
			"simple",
			Request{
				Path: "/index.html",
			},
			args{
				name: "",
			},
			"/index.html",
		},
		{
			"directory",
			Request{
				Path: "/www/",
			},
			args{
				name: "",
			},
			"/www/",
		},
		{
			"sub directory",
			Request{
				Path: "/",
			},
			args{
				name: "www/",
			},
			"/www/",
		},
		{
			"directory file",
			Request{
				Path: "/www/",
			},
			args{
				name: "index.html",
			},
			"/www/index.html",
		},
		{
			"with share",
			Request{
				Path: "/folder/",
				Share: Share{
					ID:   "abcdef123456",
					Path: "/shared/",
				},
			},
			args{
				name: "root.html",
			},
			"/shared/folder/root.html",
		},
		{
			"root share",
			Request{
				Path: "/",
				Share: Share{
					ID:   "abcdef123456",
					Path: "/shared/",
				},
			},
			args{
				name: "",
			},
			"/shared/",
		},
		{
			"shared folder",
			Request{
				Path: "/",
				Share: Share{
					ID:   "abcdef123456",
					Path: "/shared/",
				},
			},
			args{
				name: "www/",
			},
			"/shared/www/",
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if result := tc.request.SubPath(tc.args.name); result != tc.want {
				t.Errorf("Filepath() = `%s`, want `%s`", result, tc.want)
			}
		})
	}
}

func TestLayoutPath(t *testing.T) {
	type args struct {
		path string
	}

	cases := []struct {
		intention string
		instance  Request
		args      args
		want      string
	}{
		{
			"empty list",
			Request{},
			args{
				path: "/reports",
			},
			GridDisplay,
		},
		{
			"empty list",
			Request{
				Preferences: Preferences{
					ListLayoutPath: []string{"/sheets", "/reports"},
				},
			},
			args{
				path: "/reports",
			},
			ListDisplay,
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if got := tc.instance.LayoutPath(tc.args.path); got != tc.want {
				t.Errorf("LayoutPath() = `%s`, want `%s`", got, tc.want)
			}
		})
	}
}

func TestTitle(t *testing.T) {
	cases := []struct {
		intention string
		instance  Request
		want      string
	}{
		{
			"simple",
			Request{},
			"fibr",
		},
		{
			"without share",
			Request{
				Path: "/subDir/",
			},
			"fibr - subDir",
		},
		{
			"with share",
			Request{
				Path: "/",
				Share: Share{
					ID:       "a1b2c3d4",
					RootName: "abcd1234",
				},
			},
			"fibr - abcd1234",
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if result := tc.instance.Title(); result != tc.want {
				t.Errorf("Title() = `%s`, want `%s`", result, tc.want)
			}
		})
	}
}

func TestDescription(t *testing.T) {
	cases := []struct {
		intention string
		instance  Request
		want      string
	}{
		{
			"simple",
			Request{},
			"FIle BRowser",
		},
		{
			"without share",
			Request{
				Path: "/subDir/",
			},
			"FIle BRowser - subDir",
		},
		{
			"with share",
			Request{
				Path: "/",
				Share: Share{
					ID:       "a1b2c3d4",
					RootName: "abcd1234",
				},
			},
			"FIle BRowser - abcd1234",
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if result := tc.instance.Description(); result != tc.want {
				t.Errorf("Description() = `%s`, want `%s`", result, tc.want)
			}
		})
	}
}
