package push

import (
	"crypto/ecdh"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

const (
	anyHost         = "*"
	wildcardPrefix  = "*."
	httpsScheme     = "https"
	httpsPort       = "443"
	maxEndpointSize = 2048
	authSecretSize  = 16
)

var (
	// DefaultAllowedHosts lists the push services used by the main browsers: Chrome, Edge, Firefox and Safari.
	DefaultAllowedHosts = []string{
		"fcm.googleapis.com",
		"*.push.services.mozilla.com",
		"*.push.apple.com",
		"*.notify.windows.com",
	}

	ErrInvalidEndpoint     = errors.New("invalid push endpoint")
	ErrInvalidSubscription = errors.New("invalid push subscription")
)

func (s *Service) checkEndpoint(endpoint string) error {
	if len(endpoint) == 0 || len(endpoint) > maxEndpointSize {
		return fmt.Errorf("%w: empty or too long", ErrInvalidEndpoint)
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("%w: parse: %s", ErrInvalidEndpoint, err)
	}

	if parsed.Scheme != httpsScheme {
		return fmt.Errorf("%w: scheme must be %s", ErrInvalidEndpoint, httpsScheme)
	}

	if parsed.User != nil {
		return fmt.Errorf("%w: credentials are not allowed", ErrInvalidEndpoint)
	}

	if port := parsed.Port(); len(port) != 0 && port != httpsPort {
		return fmt.Errorf("%w: port is not allowed", ErrInvalidEndpoint)
	}

	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	if len(host) == 0 {
		return fmt.Errorf("%w: host is empty", ErrInvalidEndpoint)
	}

	if !isHostAllowed(host, s.allowedHosts) {
		return fmt.Errorf("%w: host `%s` is not an allowed push service", ErrInvalidEndpoint, host)
	}

	return nil
}

func isHostAllowed(host string, allowedHosts []string) bool {
	for _, allowed := range allowedHosts {
		allowed = strings.ToLower(strings.TrimSpace(allowed))

		switch {
		case len(allowed) == 0:
			continue

		case allowed == anyHost:
			return true

		case strings.HasPrefix(allowed, wildcardPrefix):
			suffix := allowed[1:]
			if len(host) > len(suffix) && strings.HasSuffix(host, suffix) {
				return true
			}

		case host == allowed:
			return true
		}
	}

	return false
}

func (s *Service) validate(subscription Subscription) error {
	if err := s.checkEndpoint(subscription.Endpoint); err != nil {
		return err
	}

	auth, err := subscription.decodedAuth()
	if err != nil || len(auth) != authSecretSize {
		return fmt.Errorf("%w: auth secret must be %d bytes", ErrInvalidSubscription, authSecretSize)
	}

	publicKey, err := subscription.decodedPublicKey()
	if err != nil {
		return fmt.Errorf("%w: decode public key", ErrInvalidSubscription)
	}

	if _, err = ecdh.P256().NewPublicKey(publicKey); err != nil {
		return fmt.Errorf("%w: public key is not a P-256 point", ErrInvalidSubscription)
	}

	return nil
}
