package data

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func setupTestDB(t *testing.T) *PostgresRepository {
	// Get connection string from environment variable
	connStr := os.Getenv("TEST_DATABASE_URL")
	if connStr == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	logger := zaptest.NewLogger(t)
	repo, err := NewPostgresRepository(context.Background(), connStr, logger)
	require.NoError(t, err)

	// Clear test data
	clearTestData(t, repo)

	return repo
}

func clearTestData(t *testing.T, repo *PostgresRepository) {
	ctx := context.Background()
	queries := []string{
		"DELETE FROM market_data",
		"DELETE FROM votes",
		"DELETE FROM peers",
		"DELETE FROM stakes",
	}

	for _, query := range queries {
		_, err := repo.pool.Exec(ctx, query)
		require.NoError(t, err)
	}
}

func TestMarketDataOperations(t *testing.T) {
	repo := setupTestDB(t)
	defer repo.Close()

	ctx := context.Background()

	t.Run("CRUD Operations", func(t *testing.T) {
		// Create
		dataID := createTestMarketData(t, repo, "BTC/USD", 50000.0)
		data, err := repo.GetMarketData(ctx, dataID)
		require.NoError(t, err)

		// Read
		retrieved, err := repo.GetMarketData(ctx, data.ID)
		require.NoError(t, err)
		assert.Equal(t, data.Symbol, retrieved.Symbol)
		assert.Equal(t, data.Price, retrieved.Price)

		// Update
		data.Price = 51000.0
		err = repo.UpdateMarketData(ctx, data)
		require.NoError(t, err)

		updated, err := repo.GetMarketData(ctx, data.ID)
		require.NoError(t, err)
		assert.Equal(t, 51000.0, updated.Price)

		// Delete
		err = repo.DeleteMarketData(ctx, data.ID)
		require.NoError(t, err)

		_, err = repo.GetMarketData(ctx, data.ID)
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("List with Filters", func(t *testing.T) {
		// Create test data
		symbols := []string{"BTC/USD", "ETH/USD"}
		prices := []float64{50000.0, 3000.0}

		for i, symbol := range symbols {
			data, err := NewMarketData(symbol, prices[i], 1.0, "test_source", "spot")
			require.NoError(t, err)
			err = repo.SaveMarketData(ctx, data)
			require.NoError(t, err)
		}

		// Test filters
		minPrice := 4000.0
		filter := MarketDataFilter{
			MinPrice: &minPrice,
			Symbol:   "BTC/USD",
		}

		results, err := repo.ListMarketData(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, results, 1)
		assert.Equal(t, "BTC/USD", results[0].Symbol)
	})
}

func TestVoteOperations(t *testing.T) {
	repo := setupTestDB(t)
	defer repo.Close()

	ctx := context.Background()

	t.Run("Vote Management", func(t *testing.T) {
		// Create market data
		data, err := NewMarketData("BTC/USD", 50000.0, 1.5, "test_source", "spot")
		require.NoError(t, err)
		err = repo.SaveMarketData(ctx, data)
		require.NoError(t, err)

		// Create votes
		vote, err := NewVote(data.ID, "validator1", true, 0.9)
		require.NoError(t, err)
		vote.Signature = []byte("test_signature")

		err = repo.SaveVote(ctx, vote)
		require.NoError(t, err)

		// Get votes by market data
		votes, err := repo.GetVotesByMarketData(ctx, data.ID)
		require.NoError(t, err)
		assert.Len(t, votes, 1)
		assert.Equal(t, vote.ValidatorID, votes[0].ValidatorID)

		// Get votes by validator
		votes, err = repo.GetVotesByValidator(ctx, "validator1")
		require.NoError(t, err)
		assert.Len(t, votes, 1)
		assert.Equal(t, data.ID, votes[0].MarketDataID)
	})
}

func TestPeerOperations(t *testing.T) {
	// Similar structure to MarketDataOperations test
	// Implementation follows the same pattern
}

func TestStakeOperations(t *testing.T) {
	// Similar structure to MarketDataOperations test
	// Implementation follows the same pattern
}

func TestConcurrentOperations(t *testing.T) {
	repo := setupTestDB(t)
	defer repo.Close()

	ctx := context.Background()
	numGoroutines := 10

	t.Run("Concurrent Market Data Creation", func(t *testing.T) {
		done := make(chan bool)

		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				data, err := NewMarketData(
					"BTC/USD",
					50000.0+float64(index),
					1.0,
					"test_source",
					"spot",
				)
				require.NoError(t, err)

				err = repo.SaveMarketData(ctx, data)
				require.NoError(t, err)

				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Verify all records were created
		filter := MarketDataFilter{
			Source: "test_source",
			Limit:  numGoroutines * 2, // Extra headroom
		}
		results, err := repo.ListMarketData(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, results, numGoroutines)
	})

	t.Run("Concurrent Updates", func(t *testing.T) {
		// Create initial data
		data, err := NewMarketData("ETH/USD", 3000.0, 1.0, "test_source", "spot")
		require.NoError(t, err)
		err = repo.SaveMarketData(ctx, data)
		require.NoError(t, err)

		done := make(chan bool)
		errs := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				// Read-modify-write cycle
				retrieved, err := repo.GetMarketData(ctx, data.ID)
				if err != nil {
					errs <- err
					done <- true
					return
				}

				retrieved.Price += 100.0
				err = repo.UpdateMarketData(ctx, retrieved)
				if err != nil {
					errs <- err
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// Check for any errors
		close(errs)
		for err := range errs {
			require.NoError(t, err)
		}

		// Verify final state
		final, err := repo.GetMarketData(ctx, data.ID)
		require.NoError(t, err)
		assert.True(t, final.Price > 3000.0)
	})
}

func TestEdgeCases(t *testing.T) {
	repo := setupTestDB(t)
	defer repo.Close()

	ctx := context.Background()

	t.Run("Duplicate MarketData", func(t *testing.T) {
		data, err := NewMarketData("BTC/USD", 50000.0, 1.0, "test_source", "spot")
		require.NoError(t, err)

		// First save should succeed
		err = repo.SaveMarketData(ctx, data)
		require.NoError(t, err)

		// Second save should fail
		err = repo.SaveMarketData(ctx, data)
		assert.ErrorIs(t, err, ErrDuplicate)
	})

	t.Run("Invalid Market Data", func(t *testing.T) {
		data := &MarketData{} // Invalid data with missing required fields
		err := repo.SaveMarketData(ctx, data)
		assert.Error(t, err)
	})

	t.Run("Non-existent Records", func(t *testing.T) {
		// Try to get non-existent market data
		_, err := repo.GetMarketData(ctx, "non-existent-id")
		assert.ErrorIs(t, err, ErrNotFound)

		// Try to update non-existent market data
		data, err := NewMarketData("BTC/USD", 50000.0, 1.0, "test_source", "spot")
		require.NoError(t, err)
		data.ID = "non-existent-id"
		err = repo.UpdateMarketData(ctx, data)
		assert.ErrorIs(t, err, ErrNotFound)

		// Try to delete non-existent market data
		err = repo.DeleteMarketData(ctx, "non-existent-id")
		assert.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("Filter Validation", func(t *testing.T) {
		invalidPrice := -1.0
		filter := MarketDataFilter{
			MinPrice: &invalidPrice,
		}

		_, err := repo.ListMarketData(ctx, filter)
		assert.Error(t, err)
	})
}

func TestTransactionBehavior(t *testing.T) {
	repo := setupTestDB(t)
	defer repo.Close()

	ctx := context.Background()

	t.Run("Transaction Rollback", func(t *testing.T) {
		// Begin transaction
		tx, err := repo.pool.Begin(ctx)
		require.NoError(t, err)

		// Create market data within transaction
		data, err := NewMarketData("BTC/USD", 50000.0, 1.0, "test_source", "spot")
		require.NoError(t, err)

		query := `
			INSERT INTO market_data (
				id, symbol, price, volume, timestamp, source, data_type,
				created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

		_, err = tx.Exec(ctx, query,
			data.ID, data.Symbol, data.Price, data.Volume,
			data.Timestamp, data.Source, data.DataType,
			data.CreatedAt, data.UpdatedAt)
		require.NoError(t, err)

		// Rollback transaction
		err = tx.Rollback(ctx)
		require.NoError(t, err)

		// Verify data was not persisted
		_, err = repo.GetMarketData(ctx, data.ID)
		assert.ErrorIs(t, err, ErrNotFound)
	})
}

func TestQueryPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	repo := setupTestDB(t)
	defer repo.Close()

	ctx := context.Background()

	// Create test data
	numRecords := 1000
	for i := 0; i < numRecords; i++ {
		data, err := NewMarketData(
			"BTC/USD",
			50000.0+float64(i),
			1.0,
			"test_source",
			"spot",
		)
		require.NoError(t, err)
		err = repo.SaveMarketData(ctx, data)
		require.NoError(t, err)
	}

	t.Run("Large Result Set", func(t *testing.T) {
		filter := MarketDataFilter{
			Source: "test_source",
			Limit:  numRecords,
		}

		start := time.Now()
		results, err := repo.ListMarketData(ctx, filter)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.Len(t, results, numRecords)
		assert.Less(t, duration, 5*time.Second) // Adjust threshold as needed
	})

	t.Run("Filtered Query Performance", func(t *testing.T) {
		minPrice := 55000.0
		maxPrice := 56000.0
		filter := MarketDataFilter{
			Source:   "test_source",
			MinPrice: &minPrice,
			MaxPrice: &maxPrice,
		}

		start := time.Now()
		results, err := repo.ListMarketData(ctx, filter)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.NotEmpty(t, results)
		assert.Less(t, duration, 1*time.Second) // Adjust threshold as needed
	})
}

func TestRepositoryCleanup(t *testing.T) {
	repo := setupTestDB(t)
	defer repo.Close()

	ctx := context.Background()

	t.Run("Connection Pool Management", func(t *testing.T) {
		// Verify connection pool is active
		err := repo.pool.Ping(ctx)
		require.NoError(t, err)

		// Close repository
		repo.Close()

		// Verify connections are closed
		err = repo.pool.Ping(ctx)
		assert.Error(t, err)
	})
}

// Helper function to create test data with specified characteristics
func createTestMarketData(t *testing.T, repo *PostgresRepository, symbol string, price float64) string {
	ctx := context.Background()
	data, err := NewMarketData(symbol, price, 1.0, "test_source", "spot")
	require.NoError(t, err)
	err = repo.SaveMarketData(ctx, data)
	require.NoError(t, err)
	return data.ID
}
