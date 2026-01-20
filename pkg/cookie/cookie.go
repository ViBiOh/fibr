package cookie

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/ViBiOh/auth/v3/pkg/model"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/httperror"
	"github.com/ViBiOh/httputils/v4/pkg/id"
	"github.com/golang-jwt/jwt/v5"
)

var (
	signMethod       = jwt.SigningMethodHS256
	signValidMethods = []string{signMethod.Alg()}
)

type ClaimUser interface {
	GetSubject() string
}

type Claim[T ClaimUser] struct {
	Content T
	jwt.RegisteredClaims
}

type Service[T ClaimUser] struct {
	hmacSecret    []byte
	jwtExpiration time.Duration
	devMode       bool
}

type Config struct {
	hmacSecret    string
	jwtExpiration time.Duration
}

func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) *Config {
	var config Config

	flags.New("HmacSecret", "HMAC Secret").Prefix(prefix).DocPrefix("cookie").StringVar(fs, &config.hmacSecret, "", overrides)
	flags.New("JwtExpiration", "JWT Expiration").Prefix(prefix).DocPrefix("cookie").DurationVar(fs, &config.jwtExpiration, time.Hour*24*5, overrides)

	return &config
}

func New[T ClaimUser](config *Config) Service[T] {
	return Service[T]{
		hmacSecret:    []byte(config.hmacSecret),
		jwtExpiration: config.jwtExpiration,
		devMode:       os.Getenv("ENV") == "dev",
	}
}

func (s Service[T]) IsEnabled() bool {
	return len(s.hmacSecret) != 0
}

func (s Service[T]) Get(r *http.Request, name string) (Claim[T], error) {
	var claim Claim[T]

	auth, err := r.Cookie(name)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return claim, model.ErrMalformedContent
		}

		return claim, fmt.Errorf("get auth cookie: %w", err)
	}

	if _, err = jwt.ParseWithClaims(auth.Value, &claim, s.jwtKeyFunc, jwt.WithValidMethods(signValidMethods)); err != nil {
		return claim, fmt.Errorf("parse JWT: %w", err)
	}

	return claim, nil
}

func (s Service[T]) Set(ctx context.Context, w http.ResponseWriter, name string, content T) bool {
	token := jwt.NewWithClaims(signMethod, s.newClaim(content))

	tokenString, err := token.SignedString(s.hmacSecret)
	if err != nil {
		httperror.InternalServerError(ctx, w, fmt.Errorf("sign JWT: %w", err))
		return false
	}

	s.setCookie(w, name, tokenString)
	return true
}

func (s Service[T]) Clear(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		Secure:   !s.devMode,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}

func (s Service[T]) jwtKeyFunc(_ *jwt.Token) (any, error) {
	return s.hmacSecret, nil
}

func (s Service[T]) newClaim(content T) Claim[T] {
	now := time.Now()

	return Claim[T]{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        id.New(),
			Subject:   content.GetSubject(),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.jwtExpiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "auth",
		},
		Content: content,
	}
}

func (s Service[T]) setCookie(w http.ResponseWriter, name, value string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		MaxAge:   int(s.jwtExpiration.Seconds()),
		Path:     "/",
		Secure:   !s.devMode,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
}
