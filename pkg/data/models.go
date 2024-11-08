package data

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidData      = errors.New("invalid data format")
	ErrInvalidID        = errors.New("invalid identifier")
	ErrInvalidPrice     = errors.New("invalid price")
	ErrInvalidTime      = errors.New("invalid timestamp")
	ErrMissingSignature = errors.New("missing required signature")
	ErrInvalidAmount    = errors.New("invalid amount")
)

// MarketData represents a single market data point
type MarketData struct {
	ID              string            `json:"id"`
	Symbol          string            `json:"symbol"`
	Price           float64           `json:"price"`
	Volume          float64           `json:"volume"`
	Timestamp       time.Time         `json:"timestamp"`
	Source          string            `json:"source"`
	DataType        string            `json:"data_type"`
	Signatures      map[string][]byte `json:"signatures"`
	MetaData        map[string]string `json:"metadata,omitempty"`
	ValidationScore float64           `json:"validation_score"`
	Hash            string            `json:"hash"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

// NewMarketData creates a new MarketData instance with validation
func NewMarketData(symbol string, price float64, volume float64, source string, dataType string) (*MarketData, error) {
	if symbol == "" {
		return nil, errors.New("symbol cannot be empty")
	}
	if price <= 0 {
		return nil, ErrInvalidPrice
	}
	if volume < 0 {
		return nil, errors.New("volume cannot be negative")
	}

	now := time.Now().UTC()
	md := &MarketData{
		ID:         uuid.New().String(),
		Symbol:     symbol,
		Price:      price,
		Volume:     volume,
		Timestamp:  now,
		Source:     source,
		DataType:   dataType,
		Signatures: make(map[string][]byte),
		MetaData:   make(map[string]string),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	md.UpdateHash()
	return md, nil
}

// Validate checks if the market data is valid
func (md *MarketData) Validate() error {
	if md.ID == "" {
		return ErrInvalidID
	}
	if md.Symbol == "" {
		return errors.New("symbol cannot be empty")
	}
	if md.Price <= 0 {
		return ErrInvalidPrice
	}
	if md.Volume < 0 {
		return errors.New("volume cannot be negative")
	}
	if md.Timestamp.IsZero() {
		return ErrInvalidTime
	}
	if md.Source == "" {
		return errors.New("source cannot be empty")
	}
	if md.DataType == "" {
		return errors.New("data type cannot be empty")
	}
	return nil
}

// UpdateHash generates and updates the hash of the market data
func (md *MarketData) UpdateHash() {
	hasher := sha256.New()
	hasher.Write([]byte(md.Symbol))
	hasher.Write([]byte(string(rune(md.Price))))
	hasher.Write([]byte(string(rune(md.Volume))))
	hasher.Write([]byte(md.Timestamp.String()))
	hasher.Write([]byte(md.Source))
	md.Hash = hex.EncodeToString(hasher.Sum(nil))
}

// AddSignature adds a validator's signature
func (md *MarketData) AddSignature(validatorID string, signature []byte) {
	md.Signatures[validatorID] = signature
	md.UpdatedAt = time.Now().UTC()
}

// Vote represents a validation vote on market data
type Vote struct {
	ID           string    `json:"id"`
	MarketDataID string    `json:"market_data_id"`
	ValidatorID  string    `json:"validator_id"`
	IsValid      bool      `json:"is_valid"`
	Confidence   float64   `json:"confidence"`
	Timestamp    time.Time `json:"timestamp"`
	Signature    []byte    `json:"signature"`
	Reason       string    `json:"reason,omitempty"`
}

// NewVote creates a new Vote instance
func NewVote(marketDataID string, validatorID string, isValid bool, confidence float64) (*Vote, error) {
	if marketDataID == "" {
		return nil, errors.New("market data ID cannot be empty")
	}
	if validatorID == "" {
		return nil, errors.New("validator ID cannot be empty")
	}
	if confidence < 0 || confidence > 1 {
		return nil, errors.New("confidence must be between 0 and 1")
	}

	return &Vote{
		ID:           uuid.New().String(),
		MarketDataID: marketDataID,
		ValidatorID:  validatorID,
		IsValid:      isValid,
		Confidence:   confidence,
		Timestamp:    time.Now().UTC(),
	}, nil
}

// Validate checks if the vote is valid
func (v *Vote) Validate() error {
	if v.ID == "" {
		return ErrInvalidID
	}
	if v.MarketDataID == "" {
		return errors.New("market data ID cannot be empty")
	}
	if v.ValidatorID == "" {
		return errors.New("validator ID cannot be empty")
	}
	if v.Confidence < 0 || v.Confidence > 1 {
		return errors.New("confidence must be between 0 and 1")
	}
	if v.Signature == nil {
		return ErrMissingSignature
	}
	if v.Timestamp.IsZero() {
		return ErrInvalidTime
	}
	return nil
}

// Peer represents a network participant
type Peer struct {
	ID          string    `json:"id"`
	Address     string    `json:"address"`
	PublicKey   []byte    `json:"public_key"`
	Reputation  float64   `json:"reputation"`
	LastSeen    time.Time `json:"last_seen"`
	IsAuthority bool      `json:"is_authority"`
	Roles       []string  `json:"roles"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Status      string    `json:"status"`
}

// NewPeer creates a new Peer instance
func NewPeer(address string, publicKey []byte) (*Peer, error) {
	if address == "" {
		return nil, errors.New("address cannot be empty")
	}
	if len(publicKey) == 0 {
		return nil, errors.New("public key cannot be empty")
	}

	now := time.Now().UTC()
	return &Peer{
		ID:         uuid.New().String(),
		Address:    address,
		PublicKey:  publicKey,
		Reputation: 0.5, // Initial neutral reputation
		LastSeen:   now,
		Roles:      []string{"basic"},
		CreatedAt:  now,
		UpdatedAt:  now,
		Status:     "active",
	}, nil
}

// UpdateReputation updates the peer's reputation score
func (p *Peer) UpdateReputation(delta float64) {
	p.Reputation += delta
	if p.Reputation < 0 {
		p.Reputation = 0
	}
	if p.Reputation > 1 {
		p.Reputation = 1
	}
	p.UpdatedAt = time.Now().UTC()
}

// UpdateLastSeen updates the peer's last seen timestamp
func (p *Peer) UpdateLastSeen() {
	p.LastSeen = time.Now().UTC()
	p.UpdatedAt = p.LastSeen
}

// Stake represents a peer's stake in the network
type Stake struct {
	ID        string    `json:"id"`
	PeerID    string    `json:"peer_id"`
	Amount    float64   `json:"amount"`
	Purpose   string    `json:"purpose"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Status    string    `json:"status"`
}

// NewStake creates a new Stake instance
func NewStake(peerID string, amount float64, purpose string, duration time.Duration) (*Stake, error) {
	if peerID == "" {
		return nil, errors.New("peer ID cannot be empty")
	}
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if purpose == "" {
		return nil, errors.New("purpose cannot be empty")
	}

	now := time.Now().UTC()
	return &Stake{
		ID:        uuid.New().String(),
		PeerID:    peerID,
		Amount:    amount,
		Purpose:   purpose,
		CreatedAt: now,
		ExpiresAt: now.Add(duration),
		Status:    "active",
	}, nil
}

// IsExpired checks if the stake has expired
func (s *Stake) IsExpired() bool {
	return time.Now().UTC().After(s.ExpiresAt)
}

// IsActive checks if the stake is active
func (s *Stake) IsActive() bool {
	return s.Status == "active" && !s.IsExpired()
}

// Metadata type for extensibility
type Metadata map[string]interface{}

// Validate checks if metadata is valid
func (m Metadata) Validate() error {
	if len(m) == 0 {
		return errors.New("metadata cannot be empty")
	}
	return nil
}

// MarketDataRepository represents a data repository for market data
type MarketDataRepository struct {
	// Add fields as needed
}
