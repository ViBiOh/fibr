package push

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/big"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/hash"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/hkdf"
)

const (
	defaultTTL    = "259200" // 3 days
	jwtDuration   = time.Hour * 4
	saltSize      = 16
	maxRecordSize = uint32(4096)
)

var (
	webpushHeader        = []byte("WebPush: info\x00")
	contentEncryptionKey = []byte("Content-Encoding: aes128gcm\x00")
)

var (
	ErrNoConfig             = errors.New("vapid key not set")
	ErrMaxPadExceeded       = errors.New("payload has exceeded the maximum length")
	ErrNoEncryptionPossible = errors.New("no encryption possible for subscription")
)

type Service struct {
	exclusive  exclusive.Service
	storage    absto.Storage
	publicKey  string
	privateKey []byte
}

type Config struct {
	publicKey  string
	privateKey string
}

func Flags(fs *flag.FlagSet, prefix string, overrides ...flags.Override) *Config {
	var config Config

	flags.New("PublicKey", "VAPID Public Key").Prefix(prefix).DocPrefix("push").StringVar(fs, &config.publicKey, "", overrides)
	flags.New("PrivateKey", "VAPID Private Key").Prefix(prefix).DocPrefix("push").StringVar(fs, &config.privateKey, "", overrides)

	return &config
}

func New(config *Config, storageService absto.Storage, exclusiveService exclusive.Service) (*Service, error) {
	if len(config.privateKey) == 0 || len(config.publicKey) == 0 {
		return nil, ErrNoConfig
	}

	privateKey, err := base64.RawURLEncoding.DecodeString(config.privateKey)
	if err != nil {
		return nil, fmt.Errorf("private key is not base64 encoded: %w", err)
	}

	return &Service{
		publicKey:  config.publicKey,
		privateKey: privateKey,
		storage:    storageService,
		exclusive:  exclusiveService,
	}, nil
}

func (s *Service) GetPublicKey() string {
	if s == nil {
		return ""
	}

	return s.publicKey
}

func (s *Service) Notify(ctx context.Context, subscription Subscription, notification Notification) (int, error) {
	if s == nil {
		return 0, ErrNoConfig
	}

	if len(subscription.Auth) == 0 || len(subscription.PublicKey) == 0 {
		return 0, ErrNoEncryptionPossible
	}

	encrypted, err := s.encryptContent(subscription, notification)
	if err != nil {
		return 0, fmt.Errorf("encrypt: %w", err)
	}

	authorization, err := s.generateJWT(subscription, jwtDuration)
	if err != nil {
		return 0, fmt.Errorf("generate jwt: %w", err)
	}

	res, err := request.Post(subscription.Endpoint).
		Header("TTL", defaultTTL).
		Header("Content-Type", "application/octet-stream").
		Header("Content-Length", strconv.Itoa(len(encrypted))).
		Header("Content-Encoding", "aes128gcm").
		Header("Authorization", authorization).
		Header("Urgency", "normal").
		Header("Topic", hash.String(path.Dir(notification.URL))).
		Send(ctx, io.NopCloser(bytes.NewBuffer(encrypted)))

	return res.StatusCode, err
}

func (s *Service) encryptContent(subscription Subscription, notification Notification) ([]byte, error) {
	subscriptionAuth, err := subscription.decodedAuth()
	if err != nil {
		return nil, fmt.Errorf("decode auth: %w", err)
	}

	subscriptionPublicKey, err := subscription.decodedPublicKey()
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}

	salt := make([]byte, saltSize)
	_, _ = rand.Read(salt)

	curve := ecdh.P256()

	localPrivateKey, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate local key: %w", err)
	}

	localPublicKey := localPrivateKey.PublicKey().Bytes()

	sharedPublic, err := curve.NewPublicKey(subscriptionPublicKey)
	if err != nil {
		return nil, fmt.Errorf("generate shared public: %w", err)
	}

	sharedSecret, err := localPrivateKey.ECDH(sharedPublic)
	if err != nil {
		return nil, fmt.Errorf("get shared secret: %w", err)
	}

	hash := sha256.New

	prkInfoBuf := bytes.NewBuffer(webpushHeader)
	prkInfoBuf.Write(subscriptionPublicKey)
	prkInfoBuf.Write(localPublicKey)

	prkHKDF := hkdf.New(hash, sharedSecret, subscriptionAuth, prkInfoBuf.Bytes())
	ikm, err := getFirstBytes(prkHKDF, 32)
	if err != nil {
		return nil, fmt.Errorf("get ikm: %w", err)
	}

	contentHKDF := hkdf.New(hash, ikm, salt, contentEncryptionKey)
	contentEncryptionKey, err := getFirstBytes(contentHKDF, 16)
	if err != nil {
		return nil, fmt.Errorf("get encryption key: %w", err)
	}

	nonceInfo := []byte("Content-Encoding: nonce\x00")
	nonceHKDF := hkdf.New(hash, ikm, salt, nonceInfo)
	nonce, err := getFirstBytes(nonceHKDF, 12)
	if err != nil {
		return nil, fmt.Errorf("get nonce: %w", err)
	}

	block, err := aes.NewCipher(contentEncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create gcm: %w", err)
	}

	recordLength := int(maxRecordSize) - 16
	output := bytes.NewBuffer(salt)

	recordSize := make([]byte, 4)
	binary.BigEndian.PutUint32(recordSize, maxRecordSize)

	output.Write(recordSize)
	output.Write([]byte{byte(len(localPublicKey))})
	output.Write(localPublicKey)

	payload, err := json.Marshal(notification)
	if err != nil {
		return nil, fmt.Errorf("marshal notification: %w", err)
	}

	dataBuf := bytes.NewBuffer(payload)
	dataBuf.Write([]byte("\x02"))

	// No padding for firefox
	// https://github.com/mozilla-services/autopush/issues/748
	if !strings.Contains(subscription.Endpoint, "mozilla.com") {
		if err := fillPadding(dataBuf, recordLength-output.Len()); err != nil {
			return nil, err
		}
	}

	ciphertext := aesgcm.Seal(nil, nonce, dataBuf.Bytes(), nil)
	output.Write(ciphertext)

	return output.Bytes(), nil
}

func getFirstBytes(reader io.Reader, length int) ([]byte, error) {
	key := make([]byte, length)

	_, err := io.ReadFull(reader, key)
	return key, err
}

func fillPadding(payload *bytes.Buffer, maxPadLen int) error {
	payloadLen := payload.Len()
	if payloadLen > maxPadLen {
		return ErrMaxPadExceeded
	}

	payload.Write(make([]byte, maxPadLen-payloadLen))

	return nil
}

func (s *Service) generateJWT(subscription Subscription, duration time.Duration) (string, error) {
	parsedEndpoint, err := url.Parse(subscription.Endpoint)
	if err != nil {
		return "", fmt.Errorf("parse endpoint URL: %w", err)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"aud": parsedEndpoint.Scheme + "://" + parsedEndpoint.Host,
		"exp": time.Now().Add(duration).Unix(),
		"sub": "mailto:bob@vibioh.fr",
	})

	jwtString, err := token.SignedString(s.getPrivateKey())
	if err != nil {
		return "", fmt.Errorf("sign with private key: %w", err)
	}

	return "vapid t=" + jwtString + ", k=" + s.publicKey, nil
}

func (s *Service) getPrivateKey() *ecdsa.PrivateKey {
	curve := elliptic.P256()

	px, py := curve.ScalarMult(
		curve.Params().Gx,
		curve.Params().Gy,
		s.privateKey,
	)

	pubKey := ecdsa.PublicKey{
		Curve: curve,
		X:     px,
		Y:     py,
	}

	d := &big.Int{}
	d.SetBytes(s.privateKey)

	return &ecdsa.PrivateKey{
		PublicKey: pubKey,
		D:         d,
	}
}
