package provider

import (
	"fmt"
	"time"

	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"golang.org/x/crypto/bcrypt"
)

var (
	// BcryptCost is the cost for the bcrypt fonction
	BcryptCost = bcrypt.DefaultCost
)

func init() {
	cost, err := findBestCost(time.Second / 4) // We want to be able to decipher 4 passwords per second
	if err != nil {
		logger.Error("unable to find best bcrypt cost: %s", err)
	}

	logger.Info("Bcrypt cost will be %d", cost)
	BcryptCost = cost
}

func findBestCost(maxDuration time.Duration) (int, error) {
	password := []byte("b6aa8c7d9931406946efe9ba2fadc1a6") // random string

	for i := bcrypt.MinCost + 1; i <= bcrypt.MaxCost; i++ {
		hashedPassword, err := bcrypt.GenerateFromPassword(password, i)
		if err != nil {
			return i, fmt.Errorf("unable to generate password: %s", err)
		}

		start := time.Now()
		_ = bcrypt.CompareHashAndPassword(hashedPassword, password)
		duration := time.Since(start)

		if duration > maxDuration {
			return i - 1, nil
		}
	}

	return bcrypt.MaxCost, nil
}
