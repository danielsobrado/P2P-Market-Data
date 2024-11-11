package voting

import (
	"context"
	"sync"
	"testing"
	"time"

	"p2p_market_data/pkg/config"
	"p2p_market_data/pkg/data"
	"p2p_market_data/pkg/p2p/host"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockRepository implements the data.Repository interface for testing
type MockRepository struct{}

// Ensure MockRepository implements the data.Repository interface
var _ data.Repository = (*MockRepository)(nil)

// Implement all methods of the data.Repository interface
// Market Data operations
func (m *MockRepository) SaveMarketData(ctx context.Context, marketData *data.MarketData) error {
	return nil
}

func (m *MockRepository) GetMarketData(ctx context.Context, id string) (*data.MarketData, error) {
	return nil, data.ErrNotFound
}

func (m *MockRepository) ListMarketData(ctx context.Context, filter data.MarketDataFilter) ([]*data.MarketData, error) {
	return nil, nil
}

func (m *MockRepository) UpdateMarketData(ctx context.Context, marketData *data.MarketData) error {
	return nil
}

func (m *MockRepository) DeleteMarketData(ctx context.Context, id string) error {
	return nil
}

// Vote operations
func (m *MockRepository) SaveVote(ctx context.Context, vote *data.Vote) error {
	return nil
}

func (m *MockRepository) GetVotesByMarketData(ctx context.Context, marketDataID string) ([]*data.Vote, error) {
	return nil, nil
}

func (m *MockRepository) GetVotesByValidator(ctx context.Context, validatorID string) ([]*data.Vote, error) {
	return nil, nil
}

// Peer operations
func (m *MockRepository) SavePeer(ctx context.Context, peer *data.Peer) error {
	return nil
}

func (m *MockRepository) GetPeer(ctx context.Context, id string) (*data.Peer, error) {
	return nil, data.ErrNotFound
}

func (m *MockRepository) ListPeers(ctx context.Context, filter data.PeerFilter) ([]*data.Peer, error) {
	return nil, nil
}

func (m *MockRepository) UpdatePeer(ctx context.Context, peer *data.Peer) error {
	return nil
}

func (m *MockRepository) DeletePeer(ctx context.Context, id string) error {
	return nil
}

// Stake operations
func (m *MockRepository) CreateStake(ctx context.Context, stake *data.Stake) error {
	return nil
}

func (m *MockRepository) SaveStake(ctx context.Context, stake *data.Stake) error {
	return nil
}

func (m *MockRepository) GetStake(ctx context.Context, id string) (*data.Stake, error) {
	return nil, data.ErrNotFound
}

func (m *MockRepository) GetStakesByPeer(ctx context.Context, peerID string) ([]*data.Stake, error) {
	return nil, nil
}

func (m *MockRepository) ListStakesByPeer(ctx context.Context, peerID string) ([]*data.Stake, error) {
	return nil, nil
}

func (m *MockRepository) UpdateStake(ctx context.Context, stake *data.Stake) error {
	return nil
}

// Dividend operations
func (m *MockRepository) GetDividendData(ctx context.Context, id string, param1 string, param2 string) ([]data.DividendData, error) {
	return nil, nil
}

// EOD operations
func (m *MockRepository) GetEODData(ctx context.Context, symbol string, startDate string, endDate string) ([]data.EODData, error) {
	return nil, nil
}

// Data source operations
func (m *MockRepository) GetDataSources(ctx context.Context) ([]data.DataSource, error) {
	return nil, nil
}

// Insider Data operations
func (m *MockRepository) GetInsiderData(ctx context.Context, symbol string, startDate string, endDate string) ([]data.InsiderTrade, error) {
	return nil, nil
}

// SearchData operation
func (m *MockRepository) SearchData(ctx context.Context, request data.DataRequest) ([]data.DataSource, error) {
	return nil, nil
}

// TopicStub implements the Topic interface for testing purposes
type TopicStub struct {
	mu       sync.Mutex
	messages [][]byte
}

func NewTopicStub() *TopicStub {
	return &TopicStub{
		messages: make([][]byte, 0),
	}
}

func (t *TopicStub) Publish(ctx context.Context, data []byte, opts ...pubsub.PubOpt) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.messages = append(t.messages, data)
	return nil
}

// Optionally, you can add methods to retrieve messages for assertions
func (t *TopicStub) GetMessages() [][]byte {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.messages
}

func TestVotingSystem_StartVoting_Success(t *testing.T) {
	// Setup
	ctx := context.Background()
	logger := zap.NewNop()

	// Create test configuration
	cfg := &config.Config{
		P2P: config.P2PConfig{
			VotingTimeout:    5 * time.Second,
			MinVoters:        1,
			ValidationQuorum: 0.5,
			Port:             0, // Use any available port for testing
		},
		Security: config.SecurityConfig{
			KeyFile: "", // Use an empty string or a test key file path
		},
	}

	// Create a mock repository
	repo := &MockRepository{}

	// Initialize the Host properly using host.NewHost
	h, err := host.NewHost(ctx, cfg, logger, repo)
	require.NoError(t, err)

	// Start the Host
	err = h.Start(ctx)
	require.NoError(t, err)
	defer h.Stop()

	// Initialize the VotingSystem
	votingSystem := NewVotingSystem(h, logger, &cfg.P2P)

	// Mock market data
	marketData := &data.MarketData{
		ID:     "md1",
		Symbol: "BTCUSD",
		Price:  50000.0,
	}

	// Start voting in a separate goroutine
	resultCh := make(chan *VoteResult)
	go func() {
		result, err := votingSystem.StartVoting(ctx, marketData)
		require.NoError(t, err)
		resultCh <- result
	}()

	// Simulate receiving a vote
	vote := &data.Vote{
		MarketDataID: "md1",
		ValidatorID:  "validator1",
		IsValid:      true,
		Confidence:   1.0,
		Signature:    []byte("signature"),
		Timestamp:    time.Now(),
	}

	time.Sleep(100 * time.Millisecond) // Allow time for the voting session to start

	err = votingSystem.SubmitVote(vote)
	require.NoError(t, err)

	// Get the result
	select {
	case result := <-resultCh:
		assert.True(t, result.Accepted)
		assert.Equal(t, 1, result.VoteCount)
		assert.Equal(t, 1.0, result.Score)
	case <-time.After(2 * time.Second):
		t.Fatal("Timed out waiting for voting result")
	}
}
