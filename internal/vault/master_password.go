package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
)

type MasterPassword struct {
	enabled      bool
	passwordHash string
	storagePath  string
}

func NewMasterPassword() *MasterPassword {
	mp := &MasterPassword{
		enabled:     false, // Default disabled until set
		storagePath: "/home/lleite/.cockpit/vault/master_password.dat",
	}
	mp.load()
	return mp
}

// SetPassword sets the master password
func (mp *MasterPassword) SetPassword(password string) error {
	// Hash the password
	hash := sha256.Sum256([]byte(password))
	hashStr := base64.URLEncoding.EncodeToString(hash[:])

	mp.passwordHash = hashStr
	mp.enabled = true

	return mp.save()
}

// Validate validates the master password
func (mp *MasterPassword) Validate(password string) bool {
	if !mp.enabled {
		return true // If disabled, always valid (dev mode)
	}

	hash := sha256.Sum256([]byte(password))
	hashStr := base64.URLEncoding.EncodeToString(hash[:])

	return hashStr == mp.passwordHash
}

// Enable enables master password protection
func (mp *MasterPassword) Enable(password string) error {
	return mp.SetPassword(password)
}

// Disable disables master password protection
func (mp *MasterPassword) Disable() error {
	mp.enabled = false
	return mp.save()
}

// ChangePassword changes the master password (requires old password)
func (mp *MasterPassword) ChangePassword(oldPassword, newPassword string) error {
	// Validate old password first
	if !mp.Validate(oldPassword) {
		return fmt.Errorf("invalid old password")
	}

	// Set new password
	return mp.SetPassword(newPassword)
}

// ForceSet forces setting a password even if already set (for recovery)
func (mp *MasterPassword) ForceSet(password string) error {
	return mp.SetPassword(password)
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
		// If decryption fails, assume file is corrupted and use defaults
		mp.enabled = false
		return nil
	}

	// Parse: enabled|hash
	parts := strings.Split(string(decryptedData), "|")
	if len(parts) != 2 {
		mp.enabled = false
		return nil
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
