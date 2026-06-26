package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
)

// EncryptedState represents encrypted lock state
type EncryptedState struct {
	Data      string `json:"data"`
	Signature string `json:"signature"`
	Nonce     string `json:"nonce"`
	Version   string `json:"version"`
	Salt      string `json:"salt"` // Add salt for key derivation
}

// StateEncryptor handles encryption and signing of lock state
type StateEncryptor struct {
	masterPassword *MasterPassword
}

func NewStateEncryptor() *StateEncryptor {
	return &StateEncryptor{
		masterPassword: NewMasterPassword(),
	}
}

// EncryptAndSign encrypts and signs the lock state
func (se *StateEncryptor) EncryptAndSign(state *LockState) error {
	// Convert state to JSON
	jsonData, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Load existing state to get salt, or generate new salt
	existingSalt := se.loadSalt()
	if existingSalt == "" {
		// Generate new salt
		salt := make([]byte, 16)
		if _, err := rand.Read(salt); err != nil {
			return fmt.Errorf("failed to generate salt: %w", err)
		}
		existingSalt = base64.URLEncoding.EncodeToString(salt)
	}

	// Get encryption key (derived from master password or fixed key with salt)
	key := se.getEncryptionKey(existingSalt)

	// Generate nonce
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt data using AES-GCM
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, jsonData, nil)

	// Sign the encrypted data
	signature := se.signData(ciphertext, existingSalt)

	// Create encrypted state
	encryptedState := EncryptedState{
		Data:      base64.URLEncoding.EncodeToString(ciphertext),
		Signature: base64.URLEncoding.EncodeToString(signature),
		Nonce:     base64.URLEncoding.EncodeToString(nonce),
		Version:   "v1",
		Salt:      existingSalt,
	}

	// Save encrypted state
	encryptedJSON, err := json.MarshalIndent(encryptedState, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal encrypted state: %w", err)
	}

	return os.WriteFile("/home/lleite/.cockpit/vault/lock_state.json", encryptedJSON, 0600)
}

// loadSalt loads the salt from existing encrypted state
func (se *StateEncryptor) loadSalt() string {
	data, err := os.ReadFile("/home/lleite/.cockpit/vault/lock_state.json")
	if err != nil {
		return "" // File doesn't exist
	}

	var encryptedState EncryptedState
	if err := json.Unmarshal(data, &encryptedState); err != nil {
		return "" // Invalid format
	}

	return encryptedState.Salt
}

// DecryptAndVerify decrypts and verifies the lock state
func (se *StateEncryptor) DecryptAndVerify() (*LockState, error) {
	// Read encrypted file
	encryptedData, err := os.ReadFile("/home/lleite/.cockpit/vault/lock_state.json")
	if err != nil {
		if os.IsNotExist(err) {
			// File doesn't exist, return default state
			return &LockState{
				IsLocked:     true,
				PackageLocks: make(map[string]bool),
				GlobalUnlock: false,
			}, nil
		}
		return nil, fmt.Errorf("failed to read encrypted state: %w", err)
	}

	// Parse encrypted state
	var encryptedState EncryptedState
	if err := json.Unmarshal(encryptedData, &encryptedState); err != nil {
		return nil, fmt.Errorf("failed to parse encrypted state: %w", err)
	}

	// Decode data
	ciphertext, err := base64.URLEncoding.DecodeString(encryptedState.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode data: %w", err)
	}

	signature, err := base64.URLEncoding.DecodeString(encryptedState.Signature)
	if err != nil {
		return nil, fmt.Errorf("failed to decode signature: %w", err)
	}

	// Verify signature with salt from file
	if !se.verifySignature(ciphertext, signature, encryptedState.Salt) {
		return nil, fmt.Errorf("signature verification failed - file may have been tampered with")
	}

	// Get decryption key with salt from file
	key := se.getEncryptionKey(encryptedState.Salt)

	// Decrypt data using AES-GCM
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, cipherData := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	// Parse decrypted state
	var state LockState
	if err := json.Unmarshal(plaintext, &state); err != nil {
		return nil, fmt.Errorf("failed to parse decrypted state: %w", err)
	}

	return &state, nil
}

// getEncryptionKey derives encryption key from master password or fixed key
func (se *StateEncryptor) getEncryptionKey(salt string) []byte {
	// If master password is set, derive key from it
	if se.masterPassword.IsEnabled() {
		// In production, we would need the actual password to derive key
		// For now, use a fixed key derived from system info + salt
		return se.deriveFixedKey(salt)
	}

	// If no master password, use fixed key (dev mode) + salt
	return se.deriveFixedKey(salt)
}

// deriveFixedKey derives a fixed key from system information
func (se *StateEncryptor) deriveFixedKey(salt string) []byte {
	// Derive key from system-specific information + salt
	hostname, _ := os.Hostname()
	userID := os.Getuid()
	data := fmt.Sprintf("cockpit-vault-encryption|%s|%d|%s", hostname, userID, salt)

	hash := sha256.Sum256([]byte(data))
	return hash[:]
}

// signData signs data with HMAC
func (se *StateEncryptor) signData(data []byte, salt string) []byte {
	key := se.deriveFixedKey(salt)
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// verifySignature verifies the HMAC signature
func (se *StateEncryptor) verifySignature(data, signature []byte, salt string) bool {
	key := se.deriveFixedKey(salt)
	h := hmac.New(sha256.New, key)
	h.Write(data)
	expectedSig := h.Sum(nil)

	return hmac.Equal(signature, expectedSig)
}
