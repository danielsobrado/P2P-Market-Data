package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/pbkdf2"
)

const (
	// Key derivation parameters
	pbkdfIterations = 100000
	saltLength      = 32
	keyLength       = 32

	// Token parameters
	tokenIssuer = "p2p_market_data"
)

// KeyPair represents a cryptographic key pair
type KeyPair struct {
	PublicKey  []byte
	PrivateKey []byte
	Algorithm  string
	Created    time.Time
}

// Token represents an authentication token
type Token struct {
	Value     string
	IssuedAt  time.Time
	ExpiresAt time.Time
	Claims    jwt.Claims
}

// Encryptor handles encryption and decryption operations
type Encryptor struct {
	key    []byte
	nonce  []byte
	cipher cipher.AEAD
}

// CryptoManager manages cryptographic operations
type CryptoManager struct {
	activeKeyPair *KeyPair
	encryptor     *Encryptor
	jwtSecret     []byte
}

// NewCryptoManager creates a new cryptographic manager
func NewCryptoManager(keyPair *KeyPair, jwtSecret []byte) (*CryptoManager, error) {
	// Initialize encryptor
	encryptor, err := newEncryptor(jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("initializing encryptor: %w", err)
	}

	return &CryptoManager{
		activeKeyPair: keyPair,
		encryptor:     encryptor,
		jwtSecret:     jwtSecret,
	}, nil
}

// GenerateKeyPair creates a new cryptographic key pair
func GenerateKeyPair() (*KeyPair, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generating key pair: %w", err)
	}

	return &KeyPair{
		PublicKey:  publicKey,
		PrivateKey: privateKey,
		Algorithm:  "Ed25519",
		Created:    time.Now(),
	}, nil
}

// Sign creates a digital signature for data
func (cm *CryptoManager) Sign(data []byte) ([]byte, error) {
	if len(cm.activeKeyPair.PrivateKey) == 0 {
		return nil, fmt.Errorf("private key not available")
	}

	signature := ed25519.Sign(cm.activeKeyPair.PrivateKey, data)
	return signature, nil
}

// Verify checks a digital signature
func (cm *CryptoManager) Verify(data, signature []byte, publicKey []byte) bool {
	return ed25519.Verify(publicKey, data, signature)
}

// Encrypt encrypts data using authenticated encryption
func (cm *CryptoManager) Encrypt(data []byte) ([]byte, error) {
	if cm.encryptor == nil {
		return nil, fmt.Errorf("encryptor not initialized")
	}

	// Generate random nonce
	nonce := make([]byte, cm.encryptor.cipher.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}

	// Encrypt and authenticate data
	ciphertext := cm.encryptor.cipher.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// Decrypt decrypts authenticated encrypted data
func (cm *CryptoManager) Decrypt(ciphertext []byte) ([]byte, error) {
	if cm.encryptor == nil {
		return nil, fmt.Errorf("encryptor not initialized")
	}

	// Extract nonce
	nonceSize := cm.encryptor.cipher.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertext[:nonceSize]
	ciphertext = ciphertext[nonceSize:]

	// Decrypt and verify data
	plaintext, err := cm.encryptor.cipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypting data: %w", err)
	}

	return plaintext, nil
}

// GenerateToken creates a new JWT token
func (cm *CryptoManager) GenerateToken(claims jwt.Claims, duration time.Duration) (*Token, error) {
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign token
	signedToken, err := token.SignedString(cm.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("signing token: %w", err)
	}

	return &Token{
		Value:     signedToken,
		IssuedAt:  now,
		ExpiresAt: now.Add(duration),
		Claims:    claims,
	}, nil
}

// ValidateToken validates a JWT token
func (cm *CryptoManager) ValidateToken(tokenString string) (jwt.Claims, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return cm.jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("parsing token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return token.Claims, nil
}

// HashData creates a cryptographic hash of data
func (cm *CryptoManager) HashData(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// DeriveKey derives an encryption key from a password
func DeriveKey(password, salt []byte) []byte {
	return pbkdf2.Key(password, salt, pbkdfIterations, keyLength, sha256.New)
}

// GenerateSalt generates a random salt
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, saltLength)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, fmt.Errorf("generating salt: %w", err)
	}
	return salt, nil
}

// Helper functions

func newEncryptor(key []byte) (*Encryptor, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	return &Encryptor{
		key:    key,
		cipher: gcm,
	}, nil
}

// Additional methods for key management

// RotateKeyPair generates and sets a new key pair
func (cm *CryptoManager) RotateKeyPair() error {
	newKeyPair, err := GenerateKeyPair()
	if err != nil {
		return fmt.Errorf("generating new key pair: %w", err)
	}

	cm.activeKeyPair = newKeyPair
	return nil
}

// ExportPublicKey exports the public key in PEM format
func (cm *CryptoManager) ExportPublicKey() string {
	return base64.StdEncoding.EncodeToString(cm.activeKeyPair.PublicKey)
}

// ValidateSignature checks if a signature is valid for given data and public key
func (cm *CryptoManager) ValidateSignature(data, signature []byte, publicKey []byte) bool {
	return ed25519.Verify(publicKey, data, signature)
}

// GenerateSecureToken generates a cryptographically secure random token
func (cm *CryptoManager) GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generating secure token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
