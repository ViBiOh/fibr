package provider

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/ViBiOh/auth/v3/pkg/argon"
	"golang.org/x/crypto/bcrypt"
)

type Share struct {
	Created  time.Time     `json:"creation"`
	ID       string        `json:"id"`
	Path     string        `json:"path"`
	RootName string        `json:"rootName"`
	Password string        `json:"password"`
	Duration time.Duration `json:"duration"`
	Edit     bool          `json:"edit"`
	Story    bool          `json:"story"`
	File     bool          `json:"file"`
}

func (s Share) String() string {
	var output strings.Builder

	output.WriteString(s.ID)
	output.WriteString(strconv.FormatBool(s.Edit))
	output.WriteString(s.Path)
	output.WriteString(strconv.FormatBool(s.Story))
	output.WriteString(s.RootName)
	output.WriteString(strconv.FormatBool(s.File))

	return output.String()
}

func (s Share) IsZero() bool {
	return len(s.ID) == 0
}

func (s Share) CheckPassword(ctx context.Context, password string, shareApp ShareManager) error {
	if s.Password == "" {
		return nil
	}

	if password == "" {
		return errors.New("empty password authorization")
	}

	switch {
	case strings.HasPrefix(string(s.Password), "$argon2id"):
		if argon.CompareHashAndPassword(s.Password, password) == nil {
			return nil
		}

	default:
		if bcrypt.CompareHashAndPassword([]byte(s.Password), []byte(password)) == nil {
			if err := shareApp.UpdatePassword(ctx, s.ID, password); err != nil {
				slog.LogAttrs(ctx, slog.LevelError, "update password", slog.Any("error", err))
			}

			return nil
		}
	}

	return errors.New("invalid credentials")
}

func (s Share) IsExpired(now time.Time) bool {
	return s.Duration != 0 && s.Created.Add(s.Duration).Before(now)
}
