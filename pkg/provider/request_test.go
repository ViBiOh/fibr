package provider

import (
	"testing"
)

func TestGetFilepath(t *testing.T) {
	cases := []struct {
		intention string
		request   Request
		input     string
		want      string
	}{
		{
			"simple",
			Request{
				Path: "index",
			},
			"",
			"/index",
		},
		{
			"directory",
			Request{
				Path: "www/",
			},
			"",
			"/www/",
		},
		{
			"directory file",
			Request{
				Path: "www/",
			},
			"index.html",
			"/www/index.html",
		},
		{
			"with given path",
			Request{
				Path: "index",
			},
			"root.html",
			"/index/root.html",
		},
		{
			"with share",
			Request{
				Path: "index",
				Share: Share{
					ID:   "a1b2c3d4",
					Path: "/shared/",
				},
			},
			"root.html",
			"/shared/index/root.html",
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if result := tc.request.GetFilepath(tc.input); result != tc.want {
				t.Errorf("GetFilepath() = `%s`, want `%s`", result, tc.want)
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
			"grid",
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
			"list",
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
