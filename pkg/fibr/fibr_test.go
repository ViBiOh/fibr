package fibr

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/ViBiOh/auth/v2/pkg/auth"
	"github.com/ViBiOh/auth/v2/pkg/ident"
	"github.com/ViBiOh/fibr/pkg/provider"
)

func TestConvertAuthenticationError(t *testing.T) {
	type args struct {
		err error
	}

	var cases = []struct {
		intention string
		args      args
		want      *provider.Error
	}{
		{
			"forbidden",
			args{
				err: fmt.Errorf("forbidden access: %w", auth.ErrForbidden),
			},
			provider.NewError(http.StatusForbidden, errors.New("you're not authorized to speak to me")),
		},
		{
			"malformed",
			args{
				err: fmt.Errorf("invalid access: %w", ident.ErrMalformedAuth),
			},
			provider.NewError(http.StatusBadRequest, errors.New("malformed auth")),
		},
		{
			"unauthorized",
			args{
				err: fmt.Errorf("invalid: %w", ident.ErrInvalidCredentials),
			},
			provider.NewError(http.StatusUnauthorized, errors.New("try again")),
		},
	}

	for _, tc := range cases {
		t.Run(tc.intention, func(t *testing.T) {
			if got := convertAuthenticationError(tc.args.err); got.Status != tc.want.Status {
				t.Errorf("convertAuthenticationError() = %d, want %d", got.Status, tc.want.Status)
			}
		})
	}
}
