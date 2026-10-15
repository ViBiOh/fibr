package fibr

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ViBiOh/fibr/pkg/mocks"
	"github.com/ViBiOh/fibr/pkg/provider"
	httpModel "github.com/ViBiOh/httputils/v4/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/renderer"
	"go.uber.org/mock/gomock"
)

func TestTemplateFuncMetadataPath(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		method  string
		target  string
		wantErr bool
	}{
		"metadata file": {
			http.MethodGet,
			"/.fibr/webhooks.json",
			true,
		},
		"metadata directory": {
			http.MethodGet,
			"/.fibr/",
			true,
		},
		"metadata write": {
			http.MethodDelete,
			"/.fibr/shares.json",
			true,
		},
		"regular file": {
			http.MethodGet,
			"/.fibrary/file.txt",
			false,
		},
	}

	for intention, testCase := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			shareManager := mocks.NewShareManager(ctrl)
			shareManager.EXPECT().Get(gomock.Any()).Return(provider.Share{}).AnyTimes()

			crudService := mocks.NewCrud(ctrl)
			if !testCase.wantErr {
				crudService.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any()).Return(renderer.Page{}, nil)
			}

			instance := Service{share: shareManager, crud: crudService}

			_, err := instance.TemplateFunc(httptest.NewRecorder(), httptest.NewRequest(testCase.method, testCase.target, nil))

			if testCase.wantErr && !errors.Is(err, httpModel.ErrNotFound) {
				t.Errorf("TemplateFunc() error = %v, want %v", err, httpModel.ErrNotFound)
			}

			if !testCase.wantErr && err != nil {
				t.Errorf("TemplateFunc() error = %v, want nil", err)
			}
		})
	}
}
