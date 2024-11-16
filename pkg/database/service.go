// pkg/database/service.go
package database

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"p2p_market_data/pkg/config"
	"p2p_market_data/pkg/data"
)

// Service manages database connections and provides access to repositories
type Service struct {
	conn   *pgx.Conn
	pool   *pgxpool.Pool
	logger *zap.Logger
	config *config.DatabaseConfig
	repo   data.Repository
	schema *data.SchemaManager

	mu        sync.RWMutex
	isRunning bool
}

// NewService creates a new database service
func NewService(cfg *config.DatabaseConfig, logger *zap.Logger) (*Service, error) {
	svc := &Service{
		config: cfg,
		logger: logger,
	}
	return svc, nil
}

// Start initializes database connections and repositories
func (s *Service) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("database service already running")
	}

	// Connect to database
	conn, err := s.connect(ctx)
	if err != nil {
		return err
	}
	s.conn = conn

	// Create connection pool
	pool, err := s.createPool(ctx)
	if err != nil {
		s.conn.Close(ctx)
		return err
	}
	s.pool = pool

	// Initialize repository
	repo, err := data.NewPostgresRepository(ctx, conn, s.logger)
	if err != nil {
		s.cleanup(ctx)
		return fmt.Errorf("initializing repository: %w", err)
	}
	s.repo = repo

	// Initialize schema manager
	s.schema = data.NewSchemaManager(conn)

	// Run schema migrations
	if err := s.schema.InitializeSchema(ctx); err != nil {
		s.cleanup(ctx)
		return fmt.Errorf("initializing schema: %w", err)
	}

	s.isRunning = true
	s.logger.Info("Database service started successfully")
	return nil
}

// Stop closes database connections
func (s *Service) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return nil
	}

	s.cleanup(ctx)
	s.isRunning = false
	s.logger.Info("Database service stopped")
	return nil
}

// GetRepository returns the data repository
func (s *Service) GetRepository() data.Repository {
	return s.repo
}

// IsHealthy checks database health
func (s *Service) IsHealthy() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isRunning {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.pool.Ping(ctx) == nil
}

// Internal methods

func (s *Service) connect(ctx context.Context) (*pgx.Conn, error) {
	conn, err := pgx.Connect(ctx, s.config.URL)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	// Test connection
	if err := conn.Ping(ctx); err != nil {
		conn.Close(ctx)
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return conn, nil
}

func (s *Service) createPool(ctx context.Context) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(s.config.URL)
	if err != nil {
		return nil, fmt.Errorf("parsing pool config: %w", err)
	}

	// Set pool configuration
	poolConfig.MaxConns = int32(s.config.MaxConnections)
	poolConfig.MinConns = int32(s.config.MinConnections)
	poolConfig.MaxConnLifetime = s.config.MaxConnLifetime
	poolConfig.MaxConnIdleTime = s.config.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = 30 * time.Second

	// Create pool
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	// Test pool
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pinging connection pool: %w", err)
	}

	return pool, nil
}

func (s *Service) cleanup(ctx context.Context) {
	if s.pool != nil {
		s.pool.Close()
	}
	if s.conn != nil {
		s.conn.Close(ctx)
	}
}

// Config represents database configuration
func (s *Service) Config() *config.DatabaseConfig {
	return s.config
}
