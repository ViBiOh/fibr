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
	"github.com/golang-jwt/jwt/v5"
)

var (
	signMethod       = jwt.SigningMethodHS256
	signValidMethods = []string{signMethod.Alg()}
)

type BasicClaim struct {
	jwt.RegisteredClaims
	Login    string `json:"login"`
	Password string `json:"password"`
}

type Service struct {
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

func New(config *Config) Service {
	return Service{
		hmacSecret:    []byte(config.hmacSecret),
		jwtExpiration: config.jwtExpiration,
		devMode:       os.Getenv("ENV") == "dev",
	}
}

func (s Service) IsEnabled() bool {
	return len(s.hmacSecret) != 0
}

func (s Service) Get(r *http.Request, name string) (BasicClaim, error) {
	var basic BasicClaim

	auth, err := r.Cookie(name)
	if err != nil {
		if errors.Is(err, http.ErrNoCookie) {
			return basic, model.ErrMalformedContent
		}

		return basic, fmt.Errorf("get auth cookie: %w", err)
	}

	if _, err = jwt.ParseWithClaims(auth.Value, &basic, s.jwtKeyFunc, jwt.WithValidMethods(signValidMethods)); err != nil {
		return basic, fmt.Errorf("parse JWT: %w", err)
	}

	return basic, nil
}

func (s Service) Set(ctx context.Context, w http.ResponseWriter, name, login, password string) bool {
	token := jwt.NewWithClaims(signMethod, s.newClaim(login, password))

	tokenString, err := token.SignedString(s.hmacSecret)
	if err != nil {
		httperror.InternalServerError(ctx, w, fmt.Errorf("sign JWT: %w", err))
		return false
	}

	s.setCookie(w, name, tokenString)
	return true
}

func (s Service) Clear(w http.ResponseWriter, name string) {
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

func (s Service) jwtKeyFunc(_ *jwt.Token) (any, error) {
	return s.hmacSecret, nil
}

func (s Service) newClaim(login, password string) BasicClaim {
	now := time.Now()

	return BasicClaim{
		Login:    login,
		Password: password,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.jwtExpiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "auth",
			Subject:   login,
			ID:        login,
		},
	}
}

func (s Service) setCookie(w http.ResponseWriter, name, value string) {
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
