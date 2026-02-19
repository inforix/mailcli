package secrets

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/99designs/keyring"
	"golang.org/x/term"
	"gopkg.in/yaml.v3"

	"mailcli/internal/config"
)

const (
	keyringPasswordEnv = "MAILCLI_KEYRING_PASSWORD" //nolint:gosec // env var name, not a credential
	keyringBackendEnv  = "MAILCLI_KEYRING_BACKEND"  //nolint:gosec // env var name, not a credential
)

var (
	ErrSecretNotFound        = errors.New("secret not found")
	errMissingSecretKey      = errors.New("missing secret key")
	errMissingUsername       = errors.New("missing username")
	errMissingPassword       = errors.New("missing password")
	errNoTTY                 = errors.New("no TTY available for keyring file backend password prompt")
	errInvalidKeyringBackend = errors.New("invalid keyring backend")
	errKeyringTimeout        = errors.New("keyring connection timed out")
	openKeyringFunc          = openKeyring
	keyringOpenFunc          = keyring.Open
)

type KeyringBackendInfo struct {
	Value  string
	Source string
}

const (
	keyringBackendSourceEnv     = "env"
	keyringBackendSourceConfig  = "config"
	keyringBackendSourceDefault = "default"
	keyringBackendAuto          = "auto"
)

func keyringItem(key string, data []byte) keyring.Item {
	return keyring.Item{
		Key:   key,
		Data:  data,
		Label: config.AppName,
	}
}

type keyringConfig struct {
	KeyringBackend string `yaml:"keyring_backend"`
}

func readKeyringConfig() (keyringConfig, error) {
	path, err := config.ConfigPath()
	if err != nil {
		return keyringConfig{}, err
	}

	b, err := os.ReadFile(path) //nolint:gosec // config path is trusted
	if err != nil {
		if os.IsNotExist(err) {
			return keyringConfig{}, nil
		}
		return keyringConfig{}, fmt.Errorf("read config: %w", err)
	}

	var cfg keyringConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return keyringConfig{}, fmt.Errorf("parse config %s: %w", path, err)
	}

	return cfg, nil
}

func ResolveKeyringBackendInfo() (KeyringBackendInfo, error) {
	if v := normalizeKeyringBackend(os.Getenv(keyringBackendEnv)); v != "" {
		return KeyringBackendInfo{Value: v, Source: keyringBackendSourceEnv}, nil
	}

	cfg, err := readKeyringConfig()
	if err != nil {
		return KeyringBackendInfo{}, fmt.Errorf("resolve keyring backend: %w", err)
	}

	if cfg.KeyringBackend != "" {
		if v := normalizeKeyringBackend(cfg.KeyringBackend); v != "" {
			return KeyringBackendInfo{Value: v, Source: keyringBackendSourceConfig}, nil
		}
	}

	return KeyringBackendInfo{Value: keyringBackendAuto, Source: keyringBackendSourceDefault}, nil
}

func allowedBackends(info KeyringBackendInfo) ([]keyring.BackendType, error) {
	switch info.Value {
	case "", keyringBackendAuto:
		return nil, nil
	case "keychain":
		return []keyring.BackendType{keyring.KeychainBackend}, nil
	case "file":
		return []keyring.BackendType{keyring.FileBackend}, nil
	default:
		return nil, fmt.Errorf("%w: %q (expected %s, keychain, or file)", errInvalidKeyringBackend, info.Value, keyringBackendAuto)
	}
}

// wrapKeychainError wraps keychain errors with helpful guidance on macOS.
func wrapKeychainError(err error) error {
	if err == nil {
		return nil
	}

	if IsKeychainLockedError(err.Error()) {
		return fmt.Errorf("%w\n\nYour macOS keychain is locked. To unlock it, run:\n  security unlock-keychain ~/Library/Keychains/login.keychain-db", err)
	}

	return err
}

func fileKeyringPasswordFuncFrom(password string, passwordSet bool, isTTY bool) keyring.PromptFunc {
	// Treat "set to empty string" as intentional; empty passphrase is valid.
	if passwordSet {
		return keyring.FixedStringPrompt(password)
	}

	if isTTY {
		return keyring.TerminalPrompt
	}

	return func(_ string) (string, error) {
		return "", fmt.Errorf("%w; set %s", errNoTTY, keyringPasswordEnv)
	}
}

func fileKeyringPasswordFunc() keyring.PromptFunc {
	password, passwordSet := os.LookupEnv(keyringPasswordEnv)
	return fileKeyringPasswordFuncFrom(password, passwordSet, term.IsTerminal(int(os.Stdin.Fd())))
}

func normalizeKeyringBackend(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// keyringOpenTimeout is the maximum time to wait for keyring.Open() to complete.
// On headless Linux, D-Bus SecretService can hang indefinitely if gnome-keyring
// is installed but not running.
const keyringOpenTimeout = 5 * time.Second

func shouldForceFileBackend(goos string, backendInfo KeyringBackendInfo, dbusAddr string) bool {
	return goos == "linux" && backendInfo.Value == keyringBackendAuto && dbusAddr == ""
}

func shouldUseKeyringTimeout(goos string, backendInfo KeyringBackendInfo, dbusAddr string) bool {
	return goos == "linux" && backendInfo.Value == "auto" && dbusAddr != ""
}

func openKeyring() (keyring.Keyring, error) {
	keyringDir, err := config.EnsureKeyringDir()
	if err != nil {
		return nil, fmt.Errorf("ensure keyring dir: %w", err)
	}

	backendInfo, err := ResolveKeyringBackendInfo()
	if err != nil {
		return nil, err
	}

	backends, err := allowedBackends(backendInfo)
	if err != nil {
		return nil, err
	}

	dbusAddr := os.Getenv("DBUS_SESSION_BUS_ADDRESS")
	// On Linux with "auto" backend and no D-Bus session, force file backend.
	if shouldForceFileBackend(runtime.GOOS, backendInfo, dbusAddr) {
		backends = []keyring.BackendType{keyring.FileBackend}
	}

	cfg := keyring.Config{
		ServiceName:              config.AppName,
		KeychainTrustApplication: false,
		AllowedBackends:          backends,
		FileDir:                  keyringDir,
		FilePasswordFunc:         fileKeyringPasswordFunc(),
	}

	if shouldUseKeyringTimeout(runtime.GOOS, backendInfo, dbusAddr) {
		return openKeyringWithTimeout(cfg, keyringOpenTimeout)
	}

	ring, err := keyringOpenFunc(cfg)
	if err != nil {
		return nil, fmt.Errorf("open keyring: %w", err)
	}

	return ring, nil
}

type keyringResult struct {
	ring keyring.Keyring
	err  error
}

// openKeyringWithTimeout wraps keyring.Open with a timeout to prevent indefinite hangs.
func openKeyringWithTimeout(cfg keyring.Config, timeout time.Duration) (keyring.Keyring, error) {
	ch := make(chan keyringResult, 1)

	go func() {
		ring, err := keyringOpenFunc(cfg)
		ch <- keyringResult{ring, err}
	}()

	select {
	case res := <-ch:
		if res.err != nil {
			return nil, fmt.Errorf("open keyring: %w", res.err)
		}

		return res.ring, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("%w after %v (D-Bus SecretService may be unresponsive); "+
			"set MAILCLI_KEYRING_BACKEND=file and MAILCLI_KEYRING_PASSWORD=<password> to use encrypted file storage instead",
			errKeyringTimeout, timeout)
	}
}

func SetSecret(key string, value []byte) error {
	key = strings.TrimSpace(key)
	if key == "" {
		return errMissingSecretKey
	}

	ring, err := openKeyringFunc()
	if err != nil {
		return err
	}

	if err := ring.Set(keyringItem(key, value)); err != nil {
		return wrapKeychainError(fmt.Errorf("store secret: %w", err))
	}

	return nil
}

func GetSecret(key string) ([]byte, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return nil, errMissingSecretKey
	}

	ring, err := openKeyringFunc()
	if err != nil {
		return nil, err
	}

	item, err := ring.Get(key)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return nil, ErrSecretNotFound
		}
		return nil, wrapKeychainError(fmt.Errorf("read secret: %w", err))
	}

	return item.Data, nil
}

func SetPassword(username, password string) error {
	user := normalize(username)
	if user == "" {
		return errMissingUsername
	}
	if password == "" {
		return errMissingPassword
	}

	return SetSecret(passwordKey(user), []byte(password))
}

func GetPassword(username string) (string, error) {
	user := normalize(username)
	if user == "" {
		return "", errMissingUsername
	}

	data, err := GetSecret(passwordKey(user))
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func passwordKey(username string) string {
	return fmt.Sprintf("auth:password:%s", username)
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
