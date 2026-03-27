package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"p2p_market_data/pkg/config"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

var (
	ErrNotFound      = errors.New("record not found")
	ErrDuplicate     = errors.New("duplicate record")
	ErrInvalidFilter = errors.New("invalid filter parameters")
)

// Repository defines the interface for data persistence
type Repository interface {
	// Market Data operations
	SaveMarketData(ctx context.Context, data *MarketData) error
	GetMarketData(ctx context.Context, id string) (*MarketData, error)
	ListMarketData(ctx context.Context, filter MarketDataFilter) ([]*MarketData, error)
	UpdateMarketData(ctx context.Context, data *MarketData) error
	DeleteMarketData(ctx context.Context, id string) error

	// Vote operations
	SaveVote(ctx context.Context, vote *Vote) error
	GetVotesByMarketData(ctx context.Context, marketDataID string) ([]*Vote, error)
	GetVotesByValidator(ctx context.Context, validatorID string) ([]*Vote, error)

	// Peer operations
	SavePeer(ctx context.Context, peer *Peer) error
	GetPeer(ctx context.Context, id string) (*Peer, error)
	ListPeers(ctx context.Context, filter PeerFilter) ([]*Peer, error)
	UpdatePeer(ctx context.Context, peer *Peer) error
	DeletePeer(ctx context.Context, id string) error

	// Stake operations
	CreateStake(ctx context.Context, stake *Stake) error
	ListStakesByPeer(ctx context.Context, peerID string) ([]*Stake, error)
	SaveStake(ctx context.Context, stake *Stake) error
	GetStake(ctx context.Context, id string) (*Stake, error)
	GetStakesByPeer(ctx context.Context, peerID string) ([]*Stake, error)
	UpdateStake(ctx context.Context, stake *Stake) error

	// Data retrieval methods
	GetEODData(ctx context.Context, symbol, startDate, endDate string) ([]EODData, error)
	GetInsiderData(ctx context.Context, symbol, startDate, endDate string) ([]InsiderTrade, error)
	GetDataSources(ctx context.Context) ([]DataSource, error)
	SearchData(ctx context.Context, request DataRequest) ([]DataSource, error)

	// Dividend Data operations
	SaveDividendData(ctx context.Context, dividend *DividendData) error
	GetDividendData(ctx context.Context, symbol string, startDate, endDate time.Time) ([]*DividendData, error)
}

// MarketDataFilter defines filter parameters for market data queries
type MarketDataFilter struct {
	Symbol   string
	MinPrice *float64
	MaxPrice *float64
	FromTime *time.Time
	ToTime   *time.Time
	Source   string
	DataType string
	Limit    int
	Offset   int
}

// PeerFilter defines filter parameters for peer queries
type PeerFilter struct {
	MinReputation *float64
	MaxReputation *float64
	IsAuthority   *bool
	Status        string
	Roles         []string
	Limit         int
	Offset        int
}

// PostgresRepository implements Repository interface using PostgreSQL
type PostgresRepository struct {
	conn   *pgx.Conn
	logger *zap.Logger
	pool   *pgxpool.Pool
}

// GetPeer implements Repository.
func (r *PostgresRepository) GetPeer(ctx context.Context, id string) (*Peer, error) {
	query := `
		SELECT id, address, reputation, last_seen, roles, metadata, created_at, updated_at
		FROM peers
		WHERE id = $1
	`

	peer := &Peer{}
	var metadataJSON []byte
	err := r.conn.QueryRow(ctx, query, id).Scan(
		&peer.ID,
		&peer.Address,
		&peer.Reputation,
		&peer.LastSeen,
		&peer.Roles,
		&metadataJSON,
		&peer.CreatedAt,
		&peer.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("querying peer: %w", err)
	}

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &peer.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshaling peer metadata: %w", err)
		}
	}

	return peer, nil
}

// GetStake implements Repository.
func (r *PostgresRepository) GetStake(ctx context.Context, id string) (*Stake, error) {
	query := `
		SELECT id, peer_id, amount, created_at, expires_at, status
		FROM stakes
		WHERE id = $1
	`

	stake := &Stake{}
	err := r.conn.QueryRow(ctx, query, id).Scan(
		&stake.ID,
		&stake.PeerID,
		&stake.Amount,
		&stake.CreatedAt,
		&stake.ExpiresAt,
		&stake.Status,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("querying stake: %w", err)
	}

	return stake, nil
}

// GetStakesByPeer implements Repository.
func (r *PostgresRepository) GetStakesByPeer(ctx context.Context, peerID string) ([]*Stake, error) {
	return r.ListStakesByPeer(ctx, peerID)
}

// ListPeers implements Repository.
func (r *PostgresRepository) ListPeers(ctx context.Context, filter PeerFilter) ([]*Peer, error) {
	query := `
		SELECT id, address, reputation, last_seen, roles, metadata, created_at, updated_at
		FROM peers
		WHERE 1=1
	`

	args := make([]interface{}, 0)
	argCount := 1

	if filter.MinReputation != nil {
		query += fmt.Sprintf(" AND reputation >= $%d", argCount)
		args = append(args, *filter.MinReputation)
		argCount++
	}
	if filter.MaxReputation != nil {
		query += fmt.Sprintf(" AND reputation <= $%d", argCount)
		args = append(args, *filter.MaxReputation)
		argCount++
	}
	if filter.Status != "" {
		query += fmt.Sprintf(" AND COALESCE((metadata->>'status'),'') = $%d", argCount)
		args = append(args, filter.Status)
		argCount++
	}
	if len(filter.Roles) > 0 {
		query += fmt.Sprintf(" AND roles && $%d", argCount)
		args = append(args, filter.Roles)
		argCount++
	}

	query += " ORDER BY reputation DESC, last_seen DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
		argCount++
	}
	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
	}

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying peers: %w", err)
	}
	defer rows.Close()

	peers := make([]*Peer, 0)
	for rows.Next() {
		peer := &Peer{}
		var metadataJSON []byte
		if err := rows.Scan(
			&peer.ID,
			&peer.Address,
			&peer.Reputation,
			&peer.LastSeen,
			&peer.Roles,
			&metadataJSON,
			&peer.CreatedAt,
			&peer.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning peer row: %w", err)
		}

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &peer.Metadata); err != nil {
				return nil, fmt.Errorf("unmarshaling peer metadata: %w", err)
			}
		}

		peers = append(peers, peer)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating peer rows: %w", err)
	}

	return peers, nil
}

// SavePeer implements Repository.
func (r *PostgresRepository) SavePeer(ctx context.Context, peer *Peer) error {
	query := `
	        INSERT INTO peers (id, address, reputation, last_seen, roles, metadata)
        VALUES ($1, $2, $3, $4, $5, $6)
        ON CONFLICT (id) DO UPDATE SET
            address = EXCLUDED.address,
            reputation = EXCLUDED.reputation,
            last_seen = EXCLUDED.last_seen,
            roles = EXCLUDED.roles,
            metadata = EXCLUDED.metadata
    `

	metadataJSON, err := json.Marshal(peer.Metadata)
	if err != nil {
		return fmt.Errorf("marshaling peer metadata: %w", err)
	}

	_, err = r.conn.Exec(ctx, query,
		peer.ID,
		peer.Address,
		peer.Reputation,
		peer.LastSeen,
		peer.Roles,
		metadataJSON,
	)

	if err != nil {
		return fmt.Errorf("saving peer: %w", err)
	}

	return nil
}

// SaveStake saves a new stake to the database
func (r *PostgresRepository) SaveStake(ctx context.Context, stake *Stake) error {
	query := `
	        INSERT INTO stakes (id, peer_id, amount, created_at, expires_at, status)
	        VALUES ($1, $2, $3, $4, $5, $6)
	        ON CONFLICT (id) DO NOTHING
	    `

	_, err := r.conn.Exec(ctx, query,
		stake.ID,
		stake.PeerID,
		stake.Amount,
		stake.CreatedAt,
		stake.ExpiresAt,
		stake.Status,
	)
	if err != nil {
		return fmt.Errorf("saving stake: %w", err)
	}
	return nil
}

// UpdatePeer updates an existing peer in the database
func (r *PostgresRepository) UpdatePeer(ctx context.Context, peer *Peer) error {
	query := `
	        UPDATE peers SET
	            address = $2,
	            reputation = $3,
            last_seen = $4,
            roles = $5,
            metadata = $6
        WHERE id = $1
    `

	metadataJSON, err := json.Marshal(peer.Metadata)
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}

	_, err = r.conn.Exec(ctx, query,
		peer.ID,
		peer.Address,
		peer.Reputation,
		peer.LastSeen,
		peer.Roles,
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("updating peer: %w", err)
	}
	return nil
}

// UpdateStake updates an existing stake in the database
func (r *PostgresRepository) UpdateStake(ctx context.Context, stake *Stake) error {
	query := `
	        UPDATE stakes SET
	            amount = $2,
	            expires_at = $3,
            status = $4
        WHERE id = $1
    `

	_, err := r.conn.Exec(ctx, query,
		stake.ID,
		stake.Amount,
		stake.ExpiresAt,
		stake.Status,
	)
	if err != nil {
		return fmt.Errorf("updating stake: %w", err)
	}
	return nil
}

// NewPostgresRepository creates a new PostgreSQL repository instance
func NewPostgresRepository(ctx context.Context, conn *pgx.Conn, logger *zap.Logger) (*PostgresRepository, error) {
	// Verify connection
	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	// Create connection pool config from existing connection
	poolConfig, err := pgxpool.ParseConfig(conn.Config().ConnString())
	if err != nil {
		return nil, fmt.Errorf("parsing pool config: %w", err)
	}

	// Set pool options
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2

	// Create the connection pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	return &PostgresRepository{
		conn:   conn,
		pool:   pool,
		logger: logger,
	}, nil
}

// Close releases all database resources
func (r *PostgresRepository) Close(ctx context.Context) {
	if r.pool != nil {
		r.pool.Close()
	}
	r.conn.Close(ctx)
}

// SaveMarketData persists market data to the database
func (r *PostgresRepository) SaveMarketData(ctx context.Context, data *MarketData) error {
	if err := data.Validate(); err != nil {
		return fmt.Errorf("validating market data: %w", err)
	}

	query := `
			INSERT INTO market_data (
				id, symbol, price, volume, timestamp, source, data_type,
				signatures, metadata, validation_score, hash, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
			)`

	signaturesJSON, err := json.Marshal(data.Signatures)
	if err != nil {
		return fmt.Errorf("marshaling signatures: %w", err)
	}
	metadataJSON, err := json.Marshal(data.MetaData)
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}

	_, err = r.conn.Exec(ctx, query,
		data.ID, data.Symbol, data.Price, data.Volume, data.Timestamp,
		data.Source, data.DataType, signaturesJSON, metadataJSON,
		data.ValidationScore, data.Hash, data.CreatedAt, data.UpdatedAt,
	)

	if err != nil {
		if isPgDuplicateError(err) {
			return ErrDuplicate
		}
		return fmt.Errorf("inserting market data: %w", err)
	}

	return nil
}

// GetMarketData retrieves market data by ID
func (r *PostgresRepository) GetMarketData(ctx context.Context, id string) (*MarketData, error) {
	query := `
		SELECT id, symbol, price, volume, timestamp, source, data_type,
			   signatures, metadata, validation_score, hash, created_at, updated_at
		FROM market_data
		WHERE id = $1`

	data := &MarketData{}
	var signaturesJSON []byte
	var metadataJSON []byte
	err := r.conn.QueryRow(ctx, query, id).Scan(
		&data.ID, &data.Symbol, &data.Price, &data.Volume, &data.Timestamp,
		&data.Source, &data.DataType, &signaturesJSON, &metadataJSON,
		&data.ValidationScore, &data.Hash, &data.CreatedAt, &data.UpdatedAt,
	)

	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("querying market data: %w", err)
	}

	if len(signaturesJSON) > 0 {
		if err := json.Unmarshal(signaturesJSON, &data.Signatures); err != nil {
			return nil, fmt.Errorf("unmarshaling signatures: %w", err)
		}
	}
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &data.MetaData); err != nil {
			return nil, fmt.Errorf("unmarshaling metadata: %w", err)
		}
	}

	return data, nil
}

// ListMarketData retrieves market data based on filter criteria
func (r *PostgresRepository) ListMarketData(ctx context.Context, filter MarketDataFilter) ([]*MarketData, error) {
	query := `
			SELECT id, symbol, price, volume, timestamp, source, data_type,
				   signatures, metadata, validation_score, hash, created_at, updated_at
			FROM market_data
			WHERE 1=1`

	args := make([]interface{}, 0)
	argCount := 1

	// Build dynamic query based on filter
	if filter.Symbol != "" {
		query += fmt.Sprintf(" AND symbol = $%d", argCount)
		args = append(args, filter.Symbol)
		argCount++
	}

	if filter.MinPrice != nil {
		query += fmt.Sprintf(" AND price >= $%d", argCount)
		args = append(args, *filter.MinPrice)
		argCount++
	}

	if filter.MaxPrice != nil {
		query += fmt.Sprintf(" AND price <= $%d", argCount)
		args = append(args, *filter.MaxPrice)
		argCount++
	}

	if filter.FromTime != nil {
		query += fmt.Sprintf(" AND timestamp >= $%d", argCount)
		args = append(args, *filter.FromTime)
		argCount++
	}

	if filter.ToTime != nil {
		query += fmt.Sprintf(" AND timestamp <= $%d", argCount)
		args = append(args, *filter.ToTime)
		argCount++
	}

	if filter.Source != "" {
		query += fmt.Sprintf(" AND source = $%d", argCount)
		args = append(args, filter.Source)
		argCount++
	}

	if filter.DataType != "" {
		query += fmt.Sprintf(" AND data_type = $%d", argCount)
		args = append(args, filter.DataType)
		argCount++
	}

	// Add ordering
	query += " ORDER BY timestamp DESC"

	// Add pagination
	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
		argCount++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
	}

	rows, err := r.conn.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("querying market data list: %w", err)
	}
	defer rows.Close()

	var results []*MarketData
	for rows.Next() {
		data := &MarketData{}
		var signaturesJSON []byte
		var metadataJSON []byte
		err := rows.Scan(
			&data.ID, &data.Symbol, &data.Price, &data.Volume, &data.Timestamp,
			&data.Source, &data.DataType, &signaturesJSON, &metadataJSON,
			&data.ValidationScore, &data.Hash, &data.CreatedAt, &data.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning market data row: %w", err)
		}
		if len(signaturesJSON) > 0 {
			if err := json.Unmarshal(signaturesJSON, &data.Signatures); err != nil {
				return nil, fmt.Errorf("unmarshaling signatures: %w", err)
			}
		}
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &data.MetaData); err != nil {
				return nil, fmt.Errorf("unmarshaling metadata: %w", err)
			}
		}
		results = append(results, data)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating market data rows: %w", err)
	}

	return results, nil
}

// UpdateMarketData updates an existing market data record
func (r *PostgresRepository) UpdateMarketData(ctx context.Context, data *MarketData) error {
	if err := data.Validate(); err != nil {
		return fmt.Errorf("validating market data: %w", err)
	}

	query := `
				UPDATE market_data
				SET symbol = $1, price = $2, volume = $3, timestamp = $4,
					source = $5, data_type = $6, signatures = $7, metadata = $8,
					validation_score = $9, hash = $10, updated_at = $11
				WHERE id = $12`

	signaturesJSON, err := json.Marshal(data.Signatures)
	if err != nil {
		return fmt.Errorf("marshaling signatures: %w", err)
	}
	metadataJSON, err := json.Marshal(data.MetaData)
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}

	result, err := r.conn.Exec(ctx, query,
		data.Symbol, data.Price, data.Volume, data.Timestamp,
		data.Source, data.DataType, signaturesJSON, metadataJSON,
		data.ValidationScore, data.Hash, time.Now().UTC(), data.ID,
	)

	if err != nil {
		return fmt.Errorf("updating market data: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// DeleteMarketData removes a market data record
func (r *PostgresRepository) DeleteMarketData(ctx context.Context, id string) error {
	query := `DELETE FROM market_data WHERE id = $1`

	result, err := r.conn.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting market data: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// SaveVote persists a vote to the database
func (r *PostgresRepository) SaveVote(ctx context.Context, vote *Vote) error {
	if err := vote.Validate(); err != nil {
		return fmt.Errorf("validating vote: %w", err)
	}

	query := `
			INSERT INTO votes (
				id, market_data_id, validator_id, is_valid, confidence,
				timestamp, signature, reason
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.conn.Exec(ctx, query,
		vote.ID, vote.MarketDataID, vote.ValidatorID, vote.IsValid,
		vote.Confidence, vote.Timestamp, vote.Signature, vote.Reason,
	)

	if err != nil {
		if isPgDuplicateError(err) {
			return ErrDuplicate
		}
		return fmt.Errorf("inserting vote: %w", err)
	}

	return nil
}

// GetVotesByMarketData retrieves all votes for a specific market data
func (r *PostgresRepository) GetVotesByMarketData(ctx context.Context, marketDataID string) ([]*Vote, error) {
	query := `
			SELECT id, market_data_id, validator_id, is_valid, confidence,
				   timestamp, signature, reason
			FROM votes
			WHERE market_data_id = $1
			ORDER BY timestamp DESC`

	return r.queryVotes(ctx, query, marketDataID)
}

// GetVotesByValidator retrieves all votes by a specific validator
func (r *PostgresRepository) GetVotesByValidator(ctx context.Context, validatorID string) ([]*Vote, error) {
	query := `
			SELECT id, market_data_id, validator_id, is_valid, confidence,
				   timestamp, signature, reason
			FROM votes
			WHERE validator_id = $1
			ORDER BY timestamp DESC`

	return r.queryVotes(ctx, query, validatorID)
}

// GetInsiderData retrieves insider trade data based on symbol and date range
func (r *PostgresRepository) GetInsiderData(ctx context.Context, symbol, startDate, endDate string) ([]InsiderTrade, error) {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start date: %w", err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end date: %w", err)
	}

	query := `
		SELECT symbol, trade_date, insider_name, insider_title, transaction_type, shares, price_per_share, value
		FROM insider_trades
		WHERE symbol = $1 AND trade_date BETWEEN $2 AND $3
		ORDER BY trade_date DESC
	`

	rows, err := r.conn.Query(ctx, query, symbol, start, end)
	if err != nil {
		return nil, fmt.Errorf("querying insider trades: %w", err)
	}
	defer rows.Close()

	results := make([]InsiderTrade, 0)
	for rows.Next() {
		item := InsiderTrade{
			MarketDataBase: MarketDataBase{
				ID:       "",
				Symbol:   symbol,
				DataType: DataTypeInsiderTrade,
			},
		}
		if err := rows.Scan(
			&item.Symbol,
			&item.TradeDate,
			&item.InsiderName,
			&item.InsiderTitle,
			&item.TransactionType,
			&item.Shares,
			&item.PricePerShare,
			&item.Value,
		); err != nil {
			return nil, fmt.Errorf("scanning insider trade: %w", err)
		}
		results = append(results, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating insider trades: %w", err)
	}

	return results, nil
}

// CreateStake creates a new stake record in the database
func (r *PostgresRepository) CreateStake(ctx context.Context, stake *Stake) error {
	return r.SaveStake(ctx, stake)
}

// ListStakesByPeer retrieves all stakes for a specific peer
func (r *PostgresRepository) ListStakesByPeer(ctx context.Context, peerID string) ([]*Stake, error) {
	query := `
		SELECT id, peer_id, amount, created_at, expires_at, status
		FROM stakes
		WHERE peer_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.conn.Query(ctx, query, peerID)
	if err != nil {
		return nil, fmt.Errorf("querying stakes: %w", err)
	}
	defer rows.Close()

	stakes := make([]*Stake, 0)
	for rows.Next() {
		stake := &Stake{}
		if err := rows.Scan(
			&stake.ID,
			&stake.PeerID,
			&stake.Amount,
			&stake.CreatedAt,
			&stake.ExpiresAt,
			&stake.Status,
		); err != nil {
			return nil, fmt.Errorf("scanning stake: %w", err)
		}
		stakes = append(stakes, stake)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating stakes: %w", err)
	}

	return stakes, nil
}

// GetDividendData retrieves dividend data based on symbol and date range
func (r *PostgresRepository) GetDividendData(ctx context.Context, symbol string, startDate, endDate time.Time) ([]*DividendData, error) {
	query := `
	        SELECT id, symbol, ex_date, payment_date, record_date, declared_date, amount, source,
	               currency, frequency, metadata
	        FROM dividends
	        WHERE symbol = $1 AND ex_date BETWEEN $2 AND $3
	        ORDER BY ex_date ASC
	    `

	rows, err := r.pool.Query(ctx, query, symbol, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("querying dividend data: %w", err)
	}
	defer rows.Close()

	var dividends []*DividendData
	for rows.Next() {
		var dividend DividendData
		var metadataJSON []byte

		err := rows.Scan(
			&dividend.ID,
			&dividend.Symbol,
			&dividend.ExDate,
			&dividend.PaymentDate,
			&dividend.RecordDate,
			&dividend.DeclaredDate,
			&dividend.Amount,
			&dividend.Source,
			&dividend.Currency,
			&dividend.Frequency,
			&metadataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning dividend data: %w", err)
		}

		if err := json.Unmarshal(metadataJSON, &dividend.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshaling metadata: %w", err)
		}

		dividends = append(dividends, &dividend)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return dividends, nil
}

// SearchData implements the Repository interface for searching data sources
func (r *PostgresRepository) SearchData(ctx context.Context, request DataRequest) ([]DataSource, error) {
	// Search currently maps to available data sources; request shape is retained for API compatibility.
	sources, err := r.GetDataSources(ctx)
	if err != nil {
		return nil, err
	}

	if request.Symbol == "" && request.Type == "" {
		return sources, nil
	}

	filtered := make([]DataSource, 0, len(sources))
	for _, source := range sources {
		typeMatch := request.Type == ""
		symbolMatch := request.Symbol == ""

		for _, t := range source.DataTypes {
			if t == request.Type {
				typeMatch = true
				break
			}
		}
		for _, s := range source.AvailableSymbols {
			if s == request.Symbol {
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

// GetEODData retrieves end-of-day data based on symbol and date range
func (r *PostgresRepository) GetEODData(ctx context.Context, symbol, startDate, endDate string) ([]EODData, error) {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start date: %w", err)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end date: %w", err)
	}

	query := `
		SELECT id, symbol, timestamp, source, data_type, validation_score, metadata
		FROM market_data
		WHERE symbol = $1 AND data_type = $2 AND timestamp BETWEEN $3 AND $4
		ORDER BY timestamp ASC
	`

	rows, err := r.conn.Query(ctx, query, symbol, DataTypeEOD, start, end)
	if err != nil {
		return nil, fmt.Errorf("querying EOD data: %w", err)
	}
	defer rows.Close()

	results := make([]EODData, 0)
	for rows.Next() {
		var item EODData
		var metadataJSON []byte
		if err := rows.Scan(
			&item.ID,
			&item.Symbol,
			&item.Timestamp,
			&item.Source,
			&item.DataType,
			&item.ValidationScore,
			&metadataJSON,
		); err != nil {
			return nil, fmt.Errorf("scanning EOD row: %w", err)
		}

		meta := map[string]string{}
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &meta); err != nil {
				return nil, fmt.Errorf("unmarshaling EOD metadata: %w", err)
			}
		}
		item.Open = parseFloat(meta["open"])
		item.High = parseFloat(meta["high"])
		item.Low = parseFloat(meta["low"])
		item.Close = parseFloat(meta["close"])
		item.AdjustedClose = parseFloat(meta["adjusted_close"])
		item.Volume = parseFloat(meta["volume"])
		item.Date = item.Timestamp
		item.Metadata = meta

		results = append(results, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating EOD rows: %w", err)
	}

	return results, nil
}

// Helper function to query votes
func (r *PostgresRepository) queryVotes(ctx context.Context, query string, arg interface{}) ([]*Vote, error) {
	rows, err := r.conn.Query(ctx, query, arg)
	if err != nil {
		return nil, fmt.Errorf("querying votes: %w", err)
	}
	defer rows.Close()

	var votes []*Vote
	for rows.Next() {
		vote := &Vote{}
		err := rows.Scan(
			&vote.ID, &vote.MarketDataID, &vote.ValidatorID, &vote.IsValid,
			&vote.Confidence, &vote.Timestamp, &vote.Signature, &vote.Reason,
		)
		if err != nil {
			return nil, fmt.Errorf("scanning vote row: %w", err)
		}
		votes = append(votes, vote)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating vote rows: %w", err)
	}

	return votes, nil
}

// Helper function to check for PostgreSQL duplicate key errors
func isPgDuplicateError(err error) bool {
	pgErr, ok := err.(*pgconn.PgError)
	return ok && pgErr.Code == "23505" // unique_violation
}

// DeletePeer removes a peer record from the database
func (r *PostgresRepository) DeletePeer(ctx context.Context, id string) error {
	query := `DELETE FROM peers WHERE id = $1`

	result, err := r.conn.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("deleting peer: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// NewRepository creates a new repository instance based on configuration
func NewRepository(ctx context.Context, cfg *config.DatabaseConfig, logger *zap.Logger) (Repository, error) {
	conn, err := pgx.Connect(ctx, cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		conn.Close(ctx)
		return nil, fmt.Errorf("failed to parse pool config: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		conn.Close(ctx)
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	repo := &PostgresRepository{
		conn:   conn,
		pool:   pool,
		logger: logger,
	}

	// Test connection
	if err := conn.Ping(ctx); err != nil {
		conn.Close(ctx)
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return repo, nil
}

// SaveDividendData saves dividend data to the database
func (r *PostgresRepository) SaveDividendData(ctx context.Context, dividend *DividendData) error {
	query := `
        INSERT INTO dividends (
            id, symbol, ex_date, payment_date, record_date, declared_date, amount, source,
            currency, frequency, metadata, created_at, updated_at
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8,
            $9, $10, $11, NOW(), NOW()
        )
        ON CONFLICT (id) DO UPDATE SET
            ex_date = EXCLUDED.ex_date,
            payment_date = EXCLUDED.payment_date,
            record_date = EXCLUDED.record_date,
            declared_date = EXCLUDED.declared_date,
            amount = EXCLUDED.amount,
            source = EXCLUDED.source,
            currency = EXCLUDED.currency,
            frequency = EXCLUDED.frequency,
            metadata = EXCLUDED.metadata,
            updated_at = NOW()
    `

	metadataJSON, err := json.Marshal(dividend.Metadata)
	if err != nil {
		return fmt.Errorf("marshaling metadata: %w", err)
	}

	_, err = r.pool.Exec(ctx, query,
		dividend.ID,
		dividend.Symbol,
		dividend.ExDate,
		dividend.PaymentDate,
		dividend.RecordDate,
		dividend.DeclaredDate,
		dividend.Amount,
		dividend.Source,
		dividend.Currency,
		dividend.Frequency,
		metadataJSON,
	)
	if err != nil {
		return fmt.Errorf("saving dividend data: %w", err)
	}
	return nil
}


func parseFloat(value string) float64 {
	if value == "" {
		return 0
	}
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return f
}
