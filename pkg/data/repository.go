package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
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
	SaveStake(ctx context.Context, stake *Stake) error
	GetStake(ctx context.Context, id string) (*Stake, error)
	GetStakesByPeer(ctx context.Context, peerID string) ([]*Stake, error)
	UpdateStake(ctx context.Context, stake *Stake) error
}

// MarketDataFilter defines filter parameters for market data queries
type MarketDataFilter struct {
	Symbol    string
	MinPrice  *float64
	MaxPrice  *float64
	FromTime  *time.Time
	ToTime    *time.Time
	Source    string
	DataType  string
	Limit     int
	Offset    int
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
	pool   *pgxpool.Pool
	logger *zap.Logger
}

// NewPostgresRepository creates a new PostgreSQL repository instance
func NewPostgresRepository(ctx context.Context, connStr string, logger *zap.Logger) (*PostgresRepository, error) {
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("parsing connection string: %w", err)
	}

	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.ConnectConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &PostgresRepository{
		pool:   pool,
		logger: logger,
	}, nil
}

// Close releases all database resources
func (r *PostgresRepository) Close() {
	r.pool.Close()
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

	_, err := r.pool.Exec(ctx, query,
		data.ID, data.Symbol, data.Price, data.Volume, data.Timestamp,
		data.Source, data.DataType, data.Signatures, data.MetaData,
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
		err := r.pool.QueryRow(ctx, query, id).Scan(
			&data.ID, &data.Symbol, &data.Price, &data.Volume, &data.Timestamp,
			&data.Source, &data.DataType, &data.Signatures, &data.MetaData,
			&data.ValidationScore, &data.Hash, &data.CreatedAt, &data.UpdatedAt,
		)
	
		if err != nil {
			if err == pgx.ErrNoRows {
				return nil, ErrNotFound
			}
			return nil, fmt.Errorf("querying market data: %w", err)
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
	
		rows, err := r.pool.Query(ctx, query, args...)
		if err != nil {
			return nil, fmt.Errorf("querying market data list: %w", err)
		}
		defer rows.Close()
	
		var results []*MarketData
		for rows.Next() {
			data := &MarketData{}
			err := rows.Scan(
				&data.ID, &data.Symbol, &data.Price, &data.Volume, &data.Timestamp,
				&data.Source, &data.DataType, &data.Signatures, &data.MetaData,
				&data.ValidationScore, &data.Hash, &data.CreatedAt, &data.UpdatedAt,
			)
			if err != nil {
				return nil, fmt.Errorf("scanning market data row: %w", err)
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
	
		result, err := r.pool.Exec(ctx, query,
			data.Symbol, data.Price, data.Volume, data.Timestamp,
			data.Source, data.DataType, data.Signatures, data.MetaData,
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
	
		result, err := r.pool.Exec(ctx, query, id)
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
	
		_, err := r.pool.Exec(ctx, query,
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
	
	// Helper function to query votes
	func (r *PostgresRepository) queryVotes(ctx context.Context, query string, arg interface{}) ([]*Vote, error) {
		rows, err := r.pool.Query(ctx, query, arg)
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