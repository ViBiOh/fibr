package provider

import (
	"context"
	"errors"
	"fmt"
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

func (s Share) RemainingDuration() string {
	now := time.Now()

	if s.IsExpired(now) {
		return ""
	}

	return HumanDuration(s.Created.Add(s.Duration).Sub(now))
}

// from https://github.com/kubernetes/kubernetes/blob/4925c6bea44efd05082cbe03d02409e0e7201252/staging/src/k8s.io/apimachinery/pkg/util/duration/duration.go
func HumanDuration(d time.Duration) string {
	// Allow deviation no more than 2 seconds(excluded) to tolerate machine time
	// inconsistence, it can be considered as almost now.
	if seconds := int(d.Seconds()); seconds < -1 {
		return "<invalid>"
	} else if seconds < 0 {
		return "0s"
	} else if seconds < 60*2 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := int(d / time.Minute)
	if minutes < 10 {
		s := int(d/time.Second) % 60
		if s == 0 {
			return fmt.Sprintf("%dm", minutes)
		}
		return fmt.Sprintf("%dm%ds", minutes, s)
	} else if minutes < 60*3 {
		return fmt.Sprintf("%dm", minutes)
	}
	hours := int(d / time.Hour)
	if hours < 8 {
		m := int(d/time.Minute) % 60
		if m == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dh%dm", hours, m)
	} else if hours < 48 {
		return fmt.Sprintf("%dh", hours)
	} else if hours < 24*8 {
		h := hours % 24
		if h == 0 {
			return fmt.Sprintf("%dd", hours/24)
		}
		return fmt.Sprintf("%dd%dh", hours/24, h)
	} else if hours < 24*365*2 {
		return fmt.Sprintf("%dd", hours/24)
	} else if hours < 24*365*8 {
		dy := int(hours/24) % 365
		if dy == 0 {
			return fmt.Sprintf("%dy", hours/24/365)
		}
		return fmt.Sprintf("%dy%dd", hours/24/365, dy)
	}
	return fmt.Sprintf("%dy", int(hours/24/365))
}
