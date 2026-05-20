package data

import (
	"context"
	"sort"
	"sync"
	"time"
)

// MemoryRepository is a small in-process Repository implementation used by
// headless demo nodes and integration tests.
type MemoryRepository struct {
	mu         sync.RWMutex
	marketData map[string]*MarketData
	votes      map[string]*Vote
	peers      map[string]*Peer
	stakes     map[string]*Stake
	dividends  map[string]*DividendData
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		marketData: make(map[string]*MarketData),
		votes:      make(map[string]*Vote),
		peers:      make(map[string]*Peer),
		stakes:     make(map[string]*Stake),
		dividends:  make(map[string]*DividendData),
	}
}

func cloneMarketData(item *MarketData) *MarketData {
	if item == nil {
		return nil
	}
	copyItem := *item
	copyItem.Signatures = make(map[string][]byte, len(item.Signatures))
	for k, v := range item.Signatures {
		copyItem.Signatures[k] = append([]byte(nil), v...)
	}
	copyItem.MetaData = make(map[string]string, len(item.MetaData))
	for k, v := range item.MetaData {
		copyItem.MetaData[k] = v
	}
	return &copyItem
}

func (r *MemoryRepository) SaveMarketData(ctx context.Context, item *MarketData) error {
	if err := item.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.marketData[item.ID]; exists {
		return ErrDuplicate
	}
	r.marketData[item.ID] = cloneMarketData(item)
	return nil
}

func (r *MemoryRepository) GetMarketData(ctx context.Context, id string) (*MarketData, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, exists := r.marketData[id]
	if !exists {
		return nil, ErrNotFound
	}
	return cloneMarketData(item), nil
}

func (r *MemoryRepository) ListMarketData(ctx context.Context, filter MarketDataFilter) ([]*MarketData, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]*MarketData, 0, len(r.marketData))
	for _, item := range r.marketData {
		if filter.Symbol != "" && item.Symbol != filter.Symbol {
			continue
		}
		if filter.MinPrice != nil && item.Price < *filter.MinPrice {
			continue
		}
		if filter.MaxPrice != nil && item.Price > *filter.MaxPrice {
			continue
		}
		if filter.FromTime != nil && item.Timestamp.Before(*filter.FromTime) {
			continue
		}
		if filter.ToTime != nil && item.Timestamp.After(*filter.ToTime) {
			continue
		}
		if filter.Source != "" && item.Source != filter.Source {
			continue
		}
		if filter.DataType != "" && item.DataType != filter.DataType {
			continue
		}
		items = append(items, cloneMarketData(item))
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Timestamp.After(items[j].Timestamp)
	})

	if filter.Offset > 0 {
		if filter.Offset >= len(items) {
			return []*MarketData{}, nil
		}
		items = items[filter.Offset:]
	}
	if filter.Limit > 0 && filter.Limit < len(items) {
		items = items[:filter.Limit]
	}

	return items, nil
}

func (r *MemoryRepository) UpdateMarketData(ctx context.Context, item *MarketData) error {
	if err := item.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.marketData[item.ID]; !exists {
		return ErrNotFound
	}
	r.marketData[item.ID] = cloneMarketData(item)
	return nil
}

func (r *MemoryRepository) DeleteMarketData(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.marketData[id]; !exists {
		return ErrNotFound
	}
	delete(r.marketData, id)
	return nil
}

func (r *MemoryRepository) SaveVote(ctx context.Context, vote *Vote) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.votes[vote.ID]; exists {
		return ErrDuplicate
	}
	copyVote := *vote
	r.votes[vote.ID] = &copyVote
	return nil
}

func (r *MemoryRepository) GetVotesByMarketData(ctx context.Context, marketDataID string) ([]*Vote, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	votes := make([]*Vote, 0)
	for _, vote := range r.votes {
		if vote.MarketDataID == marketDataID {
			copyVote := *vote
			votes = append(votes, &copyVote)
		}
	}
	return votes, nil
}

func (r *MemoryRepository) GetVotesByValidator(ctx context.Context, validatorID string) ([]*Vote, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	votes := make([]*Vote, 0)
	for _, vote := range r.votes {
		if vote.ValidatorID == validatorID {
			copyVote := *vote
			votes = append(votes, &copyVote)
		}
	}
	return votes, nil
}

func (r *MemoryRepository) SavePeer(ctx context.Context, peer *Peer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.peers[peer.ID]; exists {
		return ErrDuplicate
	}
	copyPeer := *peer
	r.peers[peer.ID] = &copyPeer
	return nil
}

func (r *MemoryRepository) GetPeer(ctx context.Context, id string) (*Peer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	peer, exists := r.peers[id]
	if !exists {
		return nil, ErrNotFound
	}
	copyPeer := *peer
	return &copyPeer, nil
}

func (r *MemoryRepository) ListPeers(ctx context.Context, filter PeerFilter) ([]*Peer, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	peers := make([]*Peer, 0, len(r.peers))
	for _, peer := range r.peers {
		if filter.MinReputation != nil && peer.Reputation < *filter.MinReputation {
			continue
		}
		if filter.IsAuthority != nil && peer.IsAuthority != *filter.IsAuthority {
			continue
		}
		if filter.Status != "" && peer.Status != filter.Status {
			continue
		}
		copyPeer := *peer
		peers = append(peers, &copyPeer)
	}
	return peers, nil
}

func (r *MemoryRepository) UpdatePeer(ctx context.Context, peer *Peer) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.peers[peer.ID]; !exists {
		return ErrNotFound
	}
	copyPeer := *peer
	r.peers[peer.ID] = &copyPeer
	return nil
}

func (r *MemoryRepository) DeletePeer(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.peers[id]; !exists {
		return ErrNotFound
	}
	delete(r.peers, id)
	return nil
}

func (r *MemoryRepository) CreateStake(ctx context.Context, stake *Stake) error {
	return r.SaveStake(ctx, stake)
}

func (r *MemoryRepository) ListStakesByPeer(ctx context.Context, peerID string) ([]*Stake, error) {
	return r.GetStakesByPeer(ctx, peerID)
}

func (r *MemoryRepository) SaveStake(ctx context.Context, stake *Stake) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.stakes[stake.ID]; exists {
		return ErrDuplicate
	}
	copyStake := *stake
	r.stakes[stake.ID] = &copyStake
	return nil
}

func (r *MemoryRepository) GetStake(ctx context.Context, id string) (*Stake, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	stake, exists := r.stakes[id]
	if !exists {
		return nil, ErrNotFound
	}
	copyStake := *stake
	return &copyStake, nil
}

func (r *MemoryRepository) GetStakesByPeer(ctx context.Context, peerID string) ([]*Stake, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	stakes := make([]*Stake, 0)
	for _, stake := range r.stakes {
		if stake.PeerID == peerID {
			copyStake := *stake
			stakes = append(stakes, &copyStake)
		}
	}
	return stakes, nil
}

func (r *MemoryRepository) UpdateStake(ctx context.Context, stake *Stake) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.stakes[stake.ID]; !exists {
		return ErrNotFound
	}
	copyStake := *stake
	r.stakes[stake.ID] = &copyStake
	return nil
}

func (r *MemoryRepository) GetEODData(ctx context.Context, symbol, startDate, endDate string) ([]EODData, error) {
	items, err := r.ListMarketData(ctx, MarketDataFilter{Symbol: symbol, DataType: DataTypeEOD})
	if err != nil {
		return nil, err
	}
	results := make([]EODData, 0, len(items))
	for _, item := range items {
		results = append(results, EODData{
			MarketDataBase: MarketDataBase{
				ID:              item.ID,
				Symbol:          item.Symbol,
				Timestamp:       item.Timestamp,
				Source:          item.Source,
				DataType:        item.DataType,
				ValidationScore: item.ValidationScore,
				Metadata:        item.MetaData,
			},
			Close:  item.Price,
			Volume: item.Volume,
			Date:   item.Timestamp,
		})
	}
	return results, nil
}

func (r *MemoryRepository) GetInsiderData(ctx context.Context, symbol, startDate, endDate string) ([]InsiderTrade, error) {
	return []InsiderTrade{}, nil
}

func (r *MemoryRepository) GetDataSources(ctx context.Context) ([]DataSource, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	symbols := make(map[string]struct{})
	dataTypes := make(map[string]struct{})
	var start time.Time
	var end time.Time
	for _, item := range r.marketData {
		symbols[item.Symbol] = struct{}{}
		dataTypes[item.DataType] = struct{}{}
		if start.IsZero() || item.Timestamp.Before(start) {
			start = item.Timestamp
		}
		if end.IsZero() || item.Timestamp.After(end) {
			end = item.Timestamp
		}
	}
	if len(r.marketData) == 0 {
		return []DataSource{}, nil
	}

	source := DataSource{
		ID:               "memory",
		PeerID:           "local",
		Reputation:       1,
		AvailableSymbols: keys(symbols),
		DataTypes:        keys(dataTypes),
		DataRangeStart:   start,
		DataRangeEnd:     end,
		LastUpdate:       end,
		Reliability:      1,
	}
	return []DataSource{source}, nil
}

func (r *MemoryRepository) SearchData(ctx context.Context, request DataRequest) ([]DataSource, error) {
	sources, err := r.GetDataSources(ctx)
	if err != nil {
		return nil, err
	}
	filtered := make([]DataSource, 0, len(sources))
	for _, source := range sources {
		typeMatch := request.Type == ""
		symbolMatch := request.Symbol == ""
		for _, dataType := range source.DataTypes {
			if dataType == request.Type {
				typeMatch = true
				break
			}
		}
		for _, symbol := range source.AvailableSymbols {
			if symbol == request.Symbol {
				symbolMatch = true
				break
			}
		}
		if typeMatch && symbolMatch {
			filtered = append(filtered, source)
		}
	}
	return filtered, nil
}

func (r *MemoryRepository) SaveDividendData(ctx context.Context, dividend *DividendData) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.dividends[dividend.ID]; exists {
		return ErrDuplicate
	}
	copyDividend := *dividend
	r.dividends[dividend.ID] = &copyDividend
	return nil
}

func (r *MemoryRepository) GetDividendData(ctx context.Context, symbol string, startDate, endDate time.Time) ([]*DividendData, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	dividends := make([]*DividendData, 0)
	for _, dividend := range r.dividends {
		if dividend.Symbol != symbol {
			continue
		}
		if dividend.Timestamp.Before(startDate) || dividend.Timestamp.After(endDate) {
			continue
		}
		copyDividend := *dividend
		dividends = append(dividends, &copyDividend)
	}
	return dividends, nil
}

func keys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
