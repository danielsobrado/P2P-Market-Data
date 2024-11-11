package data

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Error variables for consistent error handling
var (
	ErrInvalidData      = errors.New("invalid data format")
	ErrInvalidID        = errors.New("invalid identifier")
	ErrInvalidPrice     = errors.New("invalid price")
	ErrInvalidTime      = errors.New("invalid timestamp")
	ErrMissingSignature = errors.New("missing required signature")
	ErrInvalidAmount    = errors.New("invalid amount")
)

const (
	DataTypeEOD          = "EOD"
	DataTypeDividend     = "DIVIDEND"
	DataTypeInsiderTrade = "INSIDER_TRADE"
	DataTypeSplit        = "SPLIT"
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

// SearchResult represents a search result item
type SearchResult struct {
	ID    string
	Score float64
}

// DataSource represents a data source in the network
type DataSource struct {
	ID               string    `json:"id"`
	PeerID           string    `json:"peer_id"`
	Reputation       float64   `json:"reputation"`
	DataTypes        []string  `json:"data_types"`
	AvailableSymbols []string  `json:"available_symbols"`
	DataRangeStart   time.Time `json:"data_range_start"`
	DataRangeEnd     time.Time `json:"data_range_end"`
	LastUpdate       time.Time `json:"last_update"`
	Reliability      float64   `json:"reliability"`
}

// DataRequest represents a request for data
type DataRequest struct {
	Type        string    `json:"type"`
	Symbol      string    `json:"symbol"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	Granularity string    `json:"granularity"`
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
	hasher.Write([]byte(fmt.Sprintf("%f", md.Price)))
	hasher.Write([]byte(fmt.Sprintf("%f", md.Volume)))
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
	Metadata    Metadata  `json:"metadata,omitempty"`
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
type MarketDataRepository interface {
	// Define repository methods as needed
}

// MarketDataBase serves as a base struct for different market data types
type MarketDataBase struct {
	ID              string            `json:"id"`
	Symbol          string            `json:"symbol"`
	Timestamp       time.Time         `json:"timestamp"`
	Source          string            `json:"source"`
	DataType        string            `json:"data_type"`
	ValidationScore float64           `json:"validation_score"`
	UpVotes         int               `json:"up_votes"`
	DownVotes       int               `json:"down_votes"`
	Metadata        map[string]string `json:"metadata"`
}

// Validate checks if the market data base is valid
func (b *MarketDataBase) Validate() error {
	if b.ID == "" {
		return ErrInvalidID
	}
	if b.Symbol == "" {
		return errors.New("symbol cannot be empty")
	}
	if b.Timestamp.IsZero() {
		return ErrInvalidTime
	}
	if b.Source == "" {
		return errors.New("source cannot be empty")
	}
	if b.DataType == "" {
		return errors.New("data type cannot be empty")
	}
	return nil
}

// Extended market data types
type EODData struct {
	MarketDataBase
	Open          float64   `json:"open"`
	High          float64   `json:"high"`
	Low           float64   `json:"low"`
	Close         float64   `json:"close"`
	Volume        float64   `json:"volume"`
	AdjustedClose float64   `json:"adjusted_close"`
	Date          time.Time `json:"date"`
}

type DividendData struct {
	MarketDataBase
	Amount          float64   `json:"amount"`
	ExDate          time.Time `json:"ex_date"`
	PayDate         time.Time `json:"pay_date"`
	RecordDate      time.Time `json:"record_date"`
	DeclarationDate time.Time `json:"declaration_date"`
	Frequency       string    `json:"frequency"`
	Type            string    `json:"type"`
}

type InsiderTrade struct {
	MarketDataBase
	InsiderName     string    `json:"insider_name"`
	InsiderTitle    string    `json:"insider_title"`
	TradeType       string    `json:"trade_type"`
	TradeDate       time.Time `json:"trade_date"`
	Position        string    `json:"position"`
	Shares          int64     `json:"shares"`
	PricePerShare   float64   `json:"price_per_share"`
	Value           float64   `json:"value"`
	TransactionType string    `json:"transaction_type"`
}

// SplitData represents stock split information
type SplitData struct {
	MarketDataBase
	SplitRatio       float64   `json:"split_ratio"` // e.g., 2.0 for 2:1 split
	AnnouncementDate time.Time `json:"announcement_date"`
	ExDate           time.Time `json:"ex_date"`
	OldShares        int       `json:"old_shares"`
	NewShares        int       `json:"new_shares"`
	Status           string    `json:"status"` // announced, completed, cancelled
}

// Add validation method for SplitData
func (s *SplitData) Validate() error {
	if err := s.MarketDataBase.Validate(); err != nil {
		return err
	}
	if s.SplitRatio <= 0 {
		return errors.New("split ratio must be positive")
	}
	if s.ExDate.IsZero() {
		return errors.New("ex-date is required")
	}
	if s.NewShares <= 0 || s.OldShares <= 0 {
		return errors.New("shares count must be positive")
	}
	if s.Status == "" {
		return errors.New("status is required")
	}
	return nil
}

// Add constructor
func NewSplitData(
	symbol string,
	ratio float64,
	exDate time.Time,
	oldShares int,
	newShares int,
) (*SplitData, error) {
	base := MarketDataBase{
		ID:        uuid.New().String(),
		Symbol:    symbol,
		Timestamp: time.Now().UTC(),
		DataType:  DataTypeSplit,
	}

	split := &SplitData{
		MarketDataBase: base,
		SplitRatio:     ratio,
		ExDate:         exDate,
		OldShares:      oldShares,
		NewShares:      newShares,
		Status:         "announced",
	}

	if err := split.Validate(); err != nil {
		return nil, err
	}

	return split, nil
}

// Implement GetDataSources in PostgresRepository
func (r *PostgresRepository) GetDataSources(ctx context.Context) ([]DataSource, error) {
	query := `
        SELECT peer_id, reputation, data_types, available_symbols, 
               data_range_start, data_range_end, last_update, reliability
        FROM data_sources`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("querying data sources: %w", err)
	}
	defer rows.Close()

	var sources []DataSource
	for rows.Next() {
		var s DataSource
		err := rows.Scan(&s.PeerID, &s.Reputation, &s.DataTypes, &s.AvailableSymbols,
			&s.DataRangeStart, &s.DataRangeEnd, &s.LastUpdate, &s.Reliability)
		if err != nil {
			return nil, fmt.Errorf("scanning data source: %w", err)
		}
		sources = append(sources, s)
	}

	return sources, nil
}
