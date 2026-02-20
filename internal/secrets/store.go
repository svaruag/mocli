package secrets

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/svaruag/mocli/internal/config"
	"golang.org/x/crypto/argon2"
)

const (
	serviceName        = "mocli"
	defaultBackendMode = "auto"
	argon2TimeCost     = 2
	argon2MemoryKiB    = 64 * 1024
	argon2Threads      = 1
	argon2KeyLength    = 32
)

var ErrNotFound = errors.New("secret not found")
var ErrFileBackendPasswordRequired = errors.New("file keyring backend requires MO_KEYRING_PASSWORD")

type BackendInfo struct {
	Requested string `json:"requested"`
	Resolved  string `json:"resolved"`
}

type Token struct {
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope,omitempty"`
	UpdatedAt    string `json:"updated_at,omitempty"`
}

type Store struct {
	backend rawBackend
}

type rawBackend interface {
	Put(key string, value []byte) error
	Get(key string) ([]byte, error)
	Delete(key string) error
}

func ResolveBackend(lookup config.LookupFunc, cfg config.AppConfig) (BackendInfo, error) {
	requested := strings.ToLower(strings.TrimSpace(config.String(lookup, "MO_KEYRING_BACKEND", "")))
	if requested == "" {
		requested = strings.ToLower(strings.TrimSpace(cfg.KeyringBackend))
	}
	if requested == "" {
		requested = defaultBackendMode
	}

	switch requested {
	case "auto":
		if keyringAvailable() {
			return BackendInfo{Requested: requested, Resolved: "keychain"}, nil
		}
		return BackendInfo{Requested: requested, Resolved: "file"}, nil
	case "keychain":
		if !keyringAvailable() {
			return BackendInfo{}, errors.New("keychain backend requested but secret-tool is not available")
		}
		return BackendInfo{Requested: requested, Resolved: "keychain"}, nil
	case "file":
		return BackendInfo{Requested: requested, Resolved: "file"}, nil
	default:
		return BackendInfo{}, fmt.Errorf("invalid MO_KEYRING_BACKEND value %q (allowed: auto|keychain|file)", requested)
	}
}

func OpenStore(lookup config.LookupFunc, cfg config.AppConfig) (*Store, BackendInfo, error) {
	info, err := ResolveBackend(lookup, cfg)
	if err != nil {
		return nil, BackendInfo{}, err
	}

	var b rawBackend
	switch info.Resolved {
	case "keychain":
		b = &secretToolBackend{}
	case "file":
		dir, err := config.KeyringDir()
		if err != nil {
			return nil, BackendInfo{}, err
		}
		if mkErr := os.MkdirAll(dir, 0o700); mkErr != nil {
			return nil, BackendInfo{}, fmt.Errorf("ensure keyring dir: %w", mkErr)
		}
		password := config.String(lookup, "MO_KEYRING_PASSWORD", "")
		if strings.TrimSpace(password) == "" {
			return nil, BackendInfo{}, ErrFileBackendPasswordRequired
		}
		b = &fileBackend{dir: dir, password: password}
	default:
		return nil, BackendInfo{}, fmt.Errorf("unsupported backend %q", info.Resolved)
	}

	return &Store{backend: b}, info, nil
}

func TokenKey(client, email string) string {
	client = normalizeClient(client)
	email = strings.ToLower(strings.TrimSpace(email))
	return "token:" + client + ":" + email
}

func (s *Store) PutToken(client, email string, token Token) error {
	if s == nil || s.backend == nil {
		return errors.New("store is not initialized")
	}
	if strings.TrimSpace(token.RefreshToken) == "" {
		return errors.New("missing refresh token")
	}
	token.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}
	if err := s.backend.Put(TokenKey(client, email), data); err != nil {
		return err
	}
	return nil
}

func (s *Store) GetToken(client, email string) (Token, error) {
	if s == nil || s.backend == nil {
		return Token{}, errors.New("store is not initialized")
	}
	data, err := s.backend.Get(TokenKey(client, email))
	if err != nil {
		return Token{}, err
	}
	var tok Token
	if err := json.Unmarshal(data, &tok); err != nil {
		return Token{}, fmt.Errorf("parse token: %w", err)
	}
	if strings.TrimSpace(tok.RefreshToken) == "" {
		return Token{}, errors.New("refresh token is empty")
	}
	return tok, nil
}

func (s *Store) DeleteToken(client, email string) error {
	if s == nil || s.backend == nil {
		return errors.New("store is not initialized")
	}
	if err := s.backend.Delete(TokenKey(client, email)); err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	return nil
}

func normalizeClient(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "default"
	}
	return v
}

func keyringAvailable() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	_, err := exec.LookPath("secret-tool")
	return err == nil
}

type secretToolBackend struct{}

func (b *secretToolBackend) Put(key string, value []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "secret-tool", "store", "--label", "mocli token", "service", serviceName, "key", key)
	cmd.Stdin = strings.NewReader(string(value))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("secret-tool store failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (b *secretToolBackend) Get(key string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "secret-tool", "lookup", "service", serviceName, "key", key)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.ToLower(strings.TrimSpace(string(out)))
		if strings.Contains(msg, "not found") || strings.Contains(msg, "no such") {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("secret-tool lookup failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	v := strings.TrimSpace(string(out))
	if v == "" {
		return nil, ErrNotFound
	}
	return []byte(v), nil
}

func (b *secretToolBackend) Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "secret-tool", "clear", "service", serviceName, "key", key)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.ToLower(strings.TrimSpace(string(out)))
		if strings.Contains(msg, "not found") || strings.Contains(msg, "no such") {
			return ErrNotFound
		}
		return fmt.Errorf("secret-tool clear failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}

type fileBackend struct {
	dir      string
	password string
}

type envelope struct {
	Salt       string `json:"salt"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
}

func (b *fileBackend) Put(key string, value []byte) error {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("read salt: %w", err)
	}
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("read nonce: %w", err)
	}
	block, err := aes.NewCipher(deriveKeyArgon2id(b.password, salt))
	if err != nil {
		return fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("new gcm: %w", err)
	}
	ciphertext := gcm.Seal(nil, nonce, value, nil)

	env := envelope{
		Salt:       base64.RawURLEncoding.EncodeToString(salt),
		Nonce:      base64.RawURLEncoding.EncodeToString(nonce),
		Ciphertext: base64.RawURLEncoding.EncodeToString(ciphertext),
	}
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}
	if err := os.WriteFile(b.pathForKey(key), data, 0o600); err != nil {
		return fmt.Errorf("write encrypted token: %w", err)
	}
	return nil
}

func (b *fileBackend) Get(key string) ([]byte, error) {
	data, err := os.ReadFile(b.pathForKey(key))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("read encrypted token: %w", err)
	}
	var env envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return nil, fmt.Errorf("parse envelope: %w", err)
	}
	salt, err := base64.RawURLEncoding.DecodeString(env.Salt)
	if err != nil {
		return nil, fmt.Errorf("decode salt: %w", err)
	}
	nonce, err := base64.RawURLEncoding.DecodeString(env.Nonce)
	if err != nil {
		return nil, fmt.Errorf("decode nonce: %w", err)
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(env.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(deriveKeyArgon2id(b.password, salt))
	if err != nil {
		return nil, fmt.Errorf("new cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("new gcm: %w", err)
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt token: %w", err)
	}
	return plain, nil
}

func (b *fileBackend) Delete(key string) error {
	err := os.Remove(b.pathForKey(key))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrNotFound
		}
		return fmt.Errorf("delete encrypted token: %w", err)
	}
	return nil
}

func (b *fileBackend) pathForKey(key string) string {
	digest := sha256.Sum256([]byte(key))
	return filepath.Join(b.dir, hex.EncodeToString(digest[:])+".enc")
}

func deriveKeyArgon2id(password string, salt []byte) []byte {
	return argon2.IDKey(
		[]byte(password),
		salt,
		argon2TimeCost,
		argon2MemoryKiB,
		argon2Threads,
		argon2KeyLength,
	)
}
