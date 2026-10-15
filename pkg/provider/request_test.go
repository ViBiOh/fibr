package provider

import (
	"testing"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/stretchr/testify/assert"
)

func TestAbsoluteURL(t *testing.T) {
	t.Parallel()

	type args struct {
		name string
	}

	cases := map[string]struct {
		instance Request
		args     args
		want     string
	}{
		"simple": {
			Request{
				Path: "/",
			},
			args{
				name: "index.html",
			},
			"/index.html",
		},
		"dir": {
			Request{
				Path: "/",
			},
			args{
				name: "folder/",
			},
			"/folder/",
		},
		"share": {
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
	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			if got := tc.instance.AbsoluteURL(tc.args.name); got != tc.want {
				t.Errorf("AbsoluteURL() = `%s`, want `%s`", got, tc.want)
			}
		})
	}
}

func TestRelativeURL(t *testing.T) {
	t.Parallel()

	type args struct {
		item absto.Item
	}

	cases := map[string]struct {
		instance Request
		args     args
		want     string
	}{
		"simple": {
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
		"dir": {
			Request{
				Path: "/",
			},
			args{
				item: absto.Item{
					Pathname:   "/folder",
					IsDirValue: true,
				},
			},
			"folder/",
		},
		"share": {
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
		"nested folder": {
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

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			if got := tc.instance.RelativeURL(tc.args.item); got != tc.want {
				t.Errorf("RelativeURL() = `%s`, want `%s`", got, tc.want)
			}
		})
	}
}

func TestFilepath(t *testing.T) {
	t.Parallel()

	type args struct {
		name string
	}

	cases := map[string]struct {
		request Request
		args    args
		want    string
	}{
		"simple": {
			Request{
				Path: "/index.html",
			},
			args{
				name: "",
			},
			"/index.html",
		},
		"directory": {
			Request{
				Path: "/www/",
			},
			args{
				name: "",
			},
			"/www/",
		},
		"sub directory": {
			Request{
				Path: "/",
			},
			args{
				name: "www/",
			},
			"/www/",
		},
		"directory file": {
			Request{
				Path: "/www/",
			},
			args{
				name: "index.html",
			},
			"/www/index.html",
		},
		"with share": {
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
		"root share": {
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
		"shared folder": {
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

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			if result := tc.request.SubPath(tc.args.name); result != tc.want {
				t.Errorf("Filepath() = `%s`, want `%s`", result, tc.want)
			}
		})
	}
}

func TestLayoutPath(t *testing.T) {
	t.Parallel()

	type args struct {
		path string
	}

	cases := map[string]struct {
		instance Request
		args     args
		want     Display
	}{
		"empty list": {
			Request{},
			args{
				path: "/reports",
			},
			GridDisplay,
		},
		"story": {
			Request{
				Preferences: Preferences{
					LayoutPaths: map[string]Display{
						"/sheets":  ListDisplay,
						"/reports": StoryDisplay,
					},
				},
			},
			args{
				path: "/reports",
			},
			StoryDisplay,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			if got := tc.instance.LayoutPath(tc.args.path); got != tc.want {
				t.Errorf("LayoutPath() = `%s`, want `%s`", got, tc.want)
			}
		})
	}
}

func TestTitle(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		instance Request
		want     string
	}{
		"simple": {
			Request{},
			"fibr",
		},
		"without share": {
			Request{
				Path: "/subDir/",
			},
			"fibr - subDir",
		},
		"with share": {
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

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			if result := tc.instance.Title(); result != tc.want {
				t.Errorf("Title() = `%s`, want `%s`", result, tc.want)
			}
		})
	}
}

func TestDescription(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		instance Request
		want     string
	}{
		"simple": {
			Request{},
			"FIle BRowser",
		},
		"without share": {
			Request{
				Path: "/subDir/",
			},
			"FIle BRowser - subDir",
		},
		"with share": {
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

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			if result := tc.instance.Description(); result != tc.want {
				t.Errorf("Description() = `%s`, want `%s`", result, tc.want)
			}
		})
	}
}

func TestChildPath(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		request Request
		name    string
		want    string
		wantErr error
	}{
		"file in directory": {
			Request{Path: "/photos/"},
			"index.html",
			"/photos/index.html",
			nil,
		},
		"nested directory keeps trailing slash": {
			Request{Path: "/photos/"},
			"2020/",
			"/photos/2020/",
			nil,
		},
		"file view with leading slash": {
			Request{Path: "/photos"},
			"/index.html",
			"/photos/index.html",
			nil,
		},
		"inside share": {
			Request{Path: "/", Share: Share{ID: "abcdef12", Path: "/photos/"}},
			"index.html",
			"/photos/index.html",
			nil,
		},
		"empty name is the directory itself": {
			Request{Path: "/photos/"},
			"",
			"",
			ErrOutOfScope,
		},
		"dot is the directory itself": {
			Request{Path: "/photos/"},
			".",
			"",
			ErrOutOfScope,
		},
		"root": {
			Request{Path: "/"},
			"",
			"",
			ErrOutOfScope,
		},
		"parent directory": {
			Request{Path: "/photos/"},
			"../secret.txt",
			"",
			ErrOutOfScope,
		},
		"escape share root": {
			Request{Path: "/", Share: Share{ID: "abcdef12", Path: "/photos/"}},
			"../secret.txt",
			"",
			ErrOutOfScope,
		},
		"escape share root with nested traversal": {
			Request{Path: "/", Share: Share{ID: "abcdef12", Path: "/photos/"}},
			"2020/../../secret.txt",
			"",
			ErrOutOfScope,
		},
		"sibling sharing the prefix": {
			Request{Path: "/", Share: Share{ID: "abcdef12", Path: "/photos"}},
			"../photos-private/secret.txt",
			"",
			ErrOutOfScope,
		},
		"metadata directory": {
			Request{Path: "/"},
			".fibr",
			"",
			ErrOutOfScope,
		},
		"metadata content": {
			Request{Path: "/"},
			".fibr/shares.json",
			"",
			ErrOutOfScope,
		},
		"metadata reached through traversal": {
			Request{Path: "/", Share: Share{ID: "abcdef12", Path: "/photos/"}},
			"../.fibr/shares.json",
			"",
			ErrOutOfScope,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			got, err := tc.request.ChildPath(tc.name)

			assert.ErrorIs(t, err, tc.wantErr)
			assert.Equal(t, tc.want, got)
		})
	}
}
