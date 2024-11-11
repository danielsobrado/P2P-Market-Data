package data

import (
	"context"
	"time"
)

type MockRepository struct{}

// Ensure MockRepository implements the Repository interface
var _ Repository = (*MockRepository)(nil)

func NewMockRepository() Repository {
	return &MockRepository{}
}

// SaveDividendData implements Repository.
func (m *MockRepository) SaveDividendData(ctx context.Context, dividend *DividendData) error {
	return nil
}

// Market Data operations
func (m *MockRepository) SaveMarketData(ctx context.Context, data *MarketData) error {
	return nil
}

func (m *MockRepository) GetMarketData(ctx context.Context, id string) (*MarketData, error) {
	return nil, ErrNotFound
}

func (m *MockRepository) ListMarketData(ctx context.Context, filter MarketDataFilter) ([]*MarketData, error) {
	return nil, nil
}

func (m *MockRepository) UpdateMarketData(ctx context.Context, data *MarketData) error {
	return nil
}

func (m *MockRepository) DeleteMarketData(ctx context.Context, id string) error {
	return nil
}

func (m *MockRepository) GetEODData(ctx context.Context, symbol, startDate, endDate string) ([]EODData, error) {
	// Return empty slice and nil error for mock implementation
	return []EODData{}, nil
}

// Vote operations
func (m *MockRepository) SaveVote(ctx context.Context, vote *Vote) error {
	return nil
}

func (m *MockRepository) GetVotesByMarketData(ctx context.Context, marketDataID string) ([]*Vote, error) {
	return nil, nil
}

func (m *MockRepository) GetVotesByValidator(ctx context.Context, validatorID string) ([]*Vote, error) {
	return nil, nil
}

// Peer operations
func (m *MockRepository) SavePeer(ctx context.Context, peer *Peer) error {
	return nil
}

func (m *MockRepository) GetPeer(ctx context.Context, id string) (*Peer, error) {
	return nil, ErrNotFound
}

func (m *MockRepository) ListPeers(ctx context.Context, filter PeerFilter) ([]*Peer, error) {
	return nil, nil
}

func (m *MockRepository) UpdatePeer(ctx context.Context, peer *Peer) error {
	return nil
}

func (m *MockRepository) DeletePeer(ctx context.Context, id string) error {
	return nil
}

// Stake operations
func (m *MockRepository) SaveStake(ctx context.Context, stake *Stake) error {
	return nil
}

func (m *MockRepository) ListStakesByPeer(ctx context.Context, peerID string) ([]*Stake, error) {
	return nil, nil
}

func (m *MockRepository) CreateStake(ctx context.Context, stake *Stake) error {
	return nil
}

func (m *MockRepository) GetDividendData(ctx context.Context, symbol string, startDate, endDate time.Time) ([]*DividendData, error) {
	return []*DividendData{}, nil
}

// Insider Data operations
func (m *MockRepository) GetInsiderData(ctx context.Context, symbol, startDate, endDate string) ([]InsiderTrade, error) {
	return []InsiderTrade{}, nil
}

func (m *MockRepository) GetStake(ctx context.Context, id string) (*Stake, error) {
	return nil, ErrNotFound
}

func (m *MockRepository) GetStakesByPeer(ctx context.Context, peerID string) ([]*Stake, error) {
	return nil, nil
}

func (m *MockRepository) UpdateStake(ctx context.Context, stake *Stake) error {
	return nil
}

// SearchData operation
func (m *MockRepository) SearchData(ctx context.Context, request DataRequest) ([]DataSource, error) {
	return []DataSource{}, nil
}

// GetDataSources operation
func (m *MockRepository) GetDataSources(ctx context.Context) ([]DataSource, error) {
	return []DataSource{}, nil
}
