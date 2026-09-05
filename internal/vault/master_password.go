package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/term"
)

const (
	masterPasswordVerifierVersion = "v2"
	masterPasswordSaltSize        = 16
	masterPasswordKeySize         = 32
)

// masterPasswordIterations follows the OWASP-recommended PBKDF2-HMAC-SHA256
// work factor. Tests replace it in TestMain to keep the suite fast while the
// production default remains deliberately expensive.
var masterPasswordIterations = 600000

type MasterPassword struct {
	enabled      bool
	passwordHash string
	storagePath  string
	initErr      error
}

func NewMasterPassword() *MasterPassword {
	return NewMasterPasswordAt(defaultMasterPasswordPath())
}

func NewMasterPasswordAt(storagePath string) *MasterPassword {
	mp := &MasterPassword{
		enabled:     false, // Default disabled until set
		storagePath: storagePath,
	}
	if err := mp.load(); err != nil {
		mp.initErr = err
	}
	return mp
}

func defaultMasterPasswordPath() string {
	return filepath.Join(filepath.Dir(DefaultLockStatePath()), "master_password.dat")
}

func (mp *MasterPassword) InitializationError() error {
	return mp.initErr
}

// SetPassword sets the master password
func (mp *MasterPassword) SetPassword(password string) error {
	if mp.initErr != nil {
		return fmt.Errorf("master password state unavailable: %w", mp.initErr)
	}
	if mp.enabled {
		return fmt.Errorf("master password already set; authenticate before changing it")
	}
	verifier, err := newPasswordVerifier(password)
	if err != nil {
		return fmt.Errorf("failed to derive master password verifier: %w", err)
	}

	oldEnabled, oldHash := mp.enabled, mp.passwordHash
	mp.passwordHash = verifier
	mp.enabled = true

	if err := mp.save(); err != nil {
		mp.enabled, mp.passwordHash = oldEnabled, oldHash
		return err
	}
	return nil
}

// Validate validates the master password
func (mp *MasterPassword) Validate(password string) bool {
	if mp.initErr != nil {
		return false
	}
	if !mp.enabled {
		return true
	}

	return verifyPassword(password, mp.passwordHash)
}

// Enable enables master password protection
func (mp *MasterPassword) Enable(password string) error {
	return mp.SetPassword(password)
}

// Disable disables master password protection
func (mp *MasterPassword) Disable() error {
	if mp.initErr != nil {
		return fmt.Errorf("master password state unavailable: %w", mp.initErr)
	}
	oldEnabled := mp.enabled
	mp.enabled = false
	if err := mp.save(); err != nil {
		mp.enabled = oldEnabled
		return err
	}
	return nil
}

// ChangePassword changes the master password (requires old password)
func (mp *MasterPassword) ChangePassword(oldPassword, newPassword string) error {
	if mp.initErr != nil {
		return fmt.Errorf("master password state unavailable: %w", mp.initErr)
	}
	// Validate old password first
	if !mp.Validate(oldPassword) {
		return fmt.Errorf("invalid old password")
	}
	verifier, err := newPasswordVerifier(newPassword)
	if err != nil {
		return fmt.Errorf("failed to derive master password verifier: %w", err)
	}
	oldHash := mp.passwordHash
	mp.passwordHash = verifier
	if err := mp.save(); err != nil {
		mp.passwordHash = oldHash
		return err
	}
	return nil
}

// ForceSet forces setting a password even if already set (for recovery)
func (mp *MasterPassword) ForceSet(password string) error {
	if mp.initErr != nil {
		return fmt.Errorf("master password state unavailable: %w", mp.initErr)
	}
	verifier, err := newPasswordVerifier(password)
	if err != nil {
		return fmt.Errorf("failed to derive master password verifier: %w", err)
	}
	oldEnabled, oldHash := mp.enabled, mp.passwordHash
	mp.enabled = true
	mp.passwordHash = verifier
	if err := mp.save(); err != nil {
		mp.enabled, mp.passwordHash = oldEnabled, oldHash
		return err
	}
	return nil
}

// newPasswordVerifier derives a deliberately expensive verifier for the
// master password. The result contains its version, random salt and derived
// key so validation never relies on a fast unsalted password hash.
func newPasswordVerifier(password string) (string, error) {
	salt := make([]byte, masterPasswordSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	derived, err := pbkdf2.Key(sha256.New, password, salt, masterPasswordIterations, masterPasswordKeySize)
	if err != nil {
		return "", err
	}

	return strings.Join([]string{
		masterPasswordVerifierVersion,
		base64.RawURLEncoding.EncodeToString(salt),
		base64.RawURLEncoding.EncodeToString(derived),
	}, "$"), nil
}

func verifyPassword(password, verifier string) bool {
	parts := strings.Split(verifier, "$")
	if len(parts) != 3 || parts[0] != masterPasswordVerifierVersion {
		return false
	}

	salt, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(salt) != masterPasswordSaltSize {
		return false
	}
	expected, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(expected) != masterPasswordKeySize {
		return false
	}

	derived, err := pbkdf2.Key(sha256.New, password, salt, masterPasswordIterations, masterPasswordKeySize)
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(derived, expected) == 1
}

func validatePasswordVerifier(verifier string) error {
	parts := strings.Split(verifier, "$")
	if len(parts) != 3 || parts[0] != masterPasswordVerifierVersion {
		return fmt.Errorf("unsupported master password verifier version")
	}
	salt, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(salt) != masterPasswordSaltSize {
		return fmt.Errorf("invalid master password verifier salt")
	}
	derived, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(derived) != masterPasswordKeySize {
		return fmt.Errorf("invalid master password verifier")
	}
	return nil
}

// IsEnabled returns if master password is enabled
func (mp *MasterPassword) IsEnabled() bool {
	return mp.enabled
}

// PromptPassword prompts user for master password
func PromptPassword() (string, error) {
	fmt.Print("Master password: ")
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Println() // Print newline after hidden input

	return string(bytePassword), nil
}

// PromptAndValidate prompts for password and validates it
func (mp *MasterPassword) PromptAndValidate() error {
	password, err := PromptPassword()
	if err != nil {
		return err
	}

	if !mp.Validate(password) {
		return fmt.Errorf("invalid master password")
	}

	return nil
}

func (mp *MasterPassword) save() error {
	data := fmt.Sprintf("%v|%s", mp.enabled, mp.passwordHash)

	// Encrypt the data using a system-specific key
	encryptedData, err := encryptSystemData([]byte(data))
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(mp.storagePath), 0o700); err != nil {
		return fmt.Errorf("failed to create master password directory: %w", err)
	}
	if err := os.Chmod(filepath.Dir(mp.storagePath), 0o700); err != nil {
		return fmt.Errorf("failed to secure master password directory: %w", err)
	}
	return os.WriteFile(mp.storagePath, encryptedData, 0600)
}

func (mp *MasterPassword) load() error {
	data, err := os.ReadFile(mp.storagePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet
		}
		return err
	}

	// Decrypt the data
	decryptedData, err := decryptSystemData(data)
	if err != nil {
		return fmt.Errorf("failed to decrypt master password state: %w", err)
	}

	// Parse: enabled|hash
	parts := strings.Split(string(decryptedData), "|")
	if len(parts) != 2 {
		return fmt.Errorf("invalid master password state format")
	}
	if parts[0] != "true" && parts[0] != "false" {
		return fmt.Errorf("invalid master password enabled flag")
	}
	if parts[0] == "true" && parts[1] == "" {
		return fmt.Errorf("enabled master password has no verifier")
	}
	if parts[0] == "true" {
		if err := validatePasswordVerifier(parts[1]); err != nil {
			return fmt.Errorf("invalid master password verifier: %w", err)
		}
	}

	enabled := parts[0] == "true"
	mp.enabled = enabled
	mp.passwordHash = parts[1]

	return nil
}

// encryptSystemData encrypts data using system-specific key
func encryptSystemData(data []byte) ([]byte, error) {
	key := deriveSystemKey()

	// Generate nonce
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	// Encrypt using AES-GCM
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)

	// Return as base64
	return []byte(base64.URLEncoding.EncodeToString(ciphertext)), nil
}

// decryptSystemData decrypts data using system-specific key
func decryptSystemData(data []byte) ([]byte, error) {
	key := deriveSystemKey()

	// Decode base64
	ciphertext, err := base64.URLEncoding.DecodeString(string(data))
	if err != nil {
		return nil, err
	}

	// Decrypt using AES-GCM
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, cipherData := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

// deriveSystemKey derives a key from system-specific information
func deriveSystemKey() []byte {
	hostname, _ := os.Hostname()
	userID := os.Getuid()
	data := fmt.Sprintf("cockpit-master-password|%s|%d", hostname, userID)

	hash := sha256.Sum256([]byte(data))
	return hash[:]
}
