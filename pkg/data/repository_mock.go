package data

import (
	"context"
)

type MockRepository struct{}

// Ensure MockRepository implements the Repository interface
var _ Repository = (*MockRepository)(nil)

func NewMockRepository() Repository {
	return &MockRepository{}
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

func (m *MockRepository) GetStake(ctx context.Context, id string) (*Stake, error) {
	return nil, ErrNotFound
}

func (m *MockRepository) GetStakesByPeer(ctx context.Context, peerID string) ([]*Stake, error) {
	return nil, nil
}

func (m *MockRepository) UpdateStake(ctx context.Context, stake *Stake) error {
	return nil
}
