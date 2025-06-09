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
	"strconv"
	"time"

	absto "github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/fibr/pkg/exclusive"
	"github.com/ViBiOh/flags"
	"github.com/ViBiOh/httputils/v4/pkg/request"
	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/hkdf"
)

const (
	defaultTTL    = "259200" // 3 days
	jwtDuration   = time.Hour * 4
	saltSize      = 16
	maxRecordSize = 4096
)

var (
	ErrNoConfig             = errors.New("vapid key not sed")
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
	return s.publicKey
}

func (s *Service) Notify(ctx context.Context, subscription Subscription, content any) error {
	if len(subscription.Auth) == 0 || len(subscription.PublicKey) == 0 {
		return ErrNoEncryptionPossible
	}

	encrypted, err := s.encryptContent(subscription, content)
	if err != nil {
		return fmt.Errorf("encrypt: %w", err)
	}

	authorization, err := s.generateJWT(subscription, jwtDuration)
	if err != nil {
		return fmt.Errorf("generate jwt: %w", err)
	}

	_, err = request.Post(subscription.Endpoint).
		Header("TTL", defaultTTL).
		Header("Content-Type", "application/octet-stream").
		Header("Content-Length", strconv.Itoa(len(encrypted))).
		Header("Content-Encoding", "aes128gcm").
		Header("Authorization", authorization).
		Send(ctx, io.NopCloser(bytes.NewBuffer(encrypted)))
	if err != nil {
		return fmt.Errorf("send notification: %w", err)
	}

	return nil
}

func (s *Service) encryptContent(subscription Subscription, content any) ([]byte, error) {
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

	prkInfoBuf := bytes.NewBuffer([]byte("WebPush: info\x00"))
	prkInfoBuf.Write(subscriptionPublicKey)
	prkInfoBuf.Write(localPublicKey)

	prkHKDF := hkdf.New(hash, sharedSecret, subscriptionAuth, prkInfoBuf.Bytes())
	ikm, err := getHKDFKey(prkHKDF, 32)
	if err != nil {
		return nil, err
	}

	contentEncryptionKeyInfo := []byte("Content-Encoding: aes128gcm\x00")
	contentHKDF := hkdf.New(hash, ikm, salt, contentEncryptionKeyInfo)
	contentEncryptionKey, err := getHKDFKey(contentHKDF, 16)
	if err != nil {
		return nil, err
	}

	nonceInfo := []byte("Content-Encoding: nonce\x00")
	nonceHKDF := hkdf.New(hash, ikm, salt, nonceInfo)
	nonce, err := getHKDFKey(nonceHKDF, 12)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(contentEncryptionKey)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	recordLength := int(maxRecordSize) - 16

	recordBuf := bytes.NewBuffer(salt)

	rs := make([]byte, 4)
	binary.BigEndian.PutUint32(rs, uint32(maxRecordSize))

	recordBuf.Write(rs)
	recordBuf.Write([]byte{byte(len(localPublicKey))})
	recordBuf.Write(localPublicKey)

	payload, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("marshal content: %w", err)
	}

	dataBuf := bytes.NewBuffer(payload)

	dataBuf.Write([]byte("\x02"))
	if err := pad(dataBuf, recordLength-recordBuf.Len()); err != nil {
		return nil, err
	}

	ciphertext := aesgcm.Seal(nil, nonce, dataBuf.Bytes(), nil)
	recordBuf.Write(ciphertext)

	return recordBuf.Bytes(), nil
}

func getHKDFKey(hkdf io.Reader, length int) ([]byte, error) {
	key := make([]byte, length)
	n, err := io.ReadFull(hkdf, key)
	if n != len(key) || err != nil {
		return key, err
	}

	return key, nil
}

func pad(payload *bytes.Buffer, maxPadLen int) error {
	payloadLen := payload.Len()
	if payloadLen > maxPadLen {
		return ErrMaxPadExceeded
	}

	padLen := maxPadLen - payloadLen

	padding := make([]byte, padLen)
	payload.Write(padding)

	return nil
}

func (s *Service) generateJWT(subscription Subscription, duration time.Duration) (string, error) {
	parsedEndpoint, err := url.Parse(subscription.Endpoint)
	if err != nil {
		return "", err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodES256, jwt.MapClaims{
		"aud": parsedEndpoint.Scheme + "://" + parsedEndpoint.Host,
		"exp": time.Now().Add(duration).Unix(),
		"sub": "mailto:bob@vibioh.fr",
	})

	privateKey := s.getPrivateKey()

	jwtString, err := token.SignedString(privateKey)
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
