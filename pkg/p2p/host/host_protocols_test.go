package host

import (
	"testing"
	"time"

	"p2p_market_data/pkg/data"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChunkDataResponseSplitsRowsAndAddsResumeMetadata(t *testing.T) {
	rows := make([]data.EODData, 5)
	for i := range rows {
		rows[i] = data.EODData{
			MarketDataBase: data.MarketDataBase{Symbol: "AAPL", DataType: data.DataTypeEOD},
			Date:           time.Date(2026, time.January, i+1, 0, 0, 0, 0, time.UTC),
			Close:          float64(100 + i),
		}
	}

	resp, err := chunkDataResponse(dataResponse{
		RequestID:  "req-1",
		TransferID: "tx-1",
		Type:       data.DataTypeEOD,
		EOD:        rows,
	}, data.DataRequest{
		RequestID:  "req-1",
		TransferID: "tx-1",
		Type:       data.DataTypeEOD,
		Offset:     2,
		ChunkSize:  2,
	})

	require.NoError(t, err)
	assert.Equal(t, "req-1", resp.RequestID)
	assert.Equal(t, "tx-1", resp.TransferID)
	assert.Equal(t, 2, resp.Offset)
	assert.Equal(t, 4, resp.NextOffset)
	assert.Equal(t, 1, resp.ChunkIndex)
	assert.Equal(t, 2, resp.ChunkSize)
	assert.Equal(t, 5, resp.TotalRows)
	assert.Equal(t, 3, resp.TotalChunks)
	assert.True(t, resp.HasMore)
	require.Len(t, resp.EOD, 2)
	assert.Equal(t, 102.0, resp.EOD[0].Close)
	require.NotEmpty(t, resp.Checksum)
	require.NoError(t, verifyResponseChecksum(resp))
}

func TestChunkDataResponseFinalChunkAndChecksumTamper(t *testing.T) {
	resp, err := chunkDataResponse(dataResponse{
		Type: data.DataTypeSplit,
		Splits: []data.SplitData{
			{MarketDataBase: data.MarketDataBase{Symbol: "MSFT", DataType: data.DataTypeSplit}, SplitRatio: 2},
			{MarketDataBase: data.MarketDataBase{Symbol: "MSFT", DataType: data.DataTypeSplit}, SplitRatio: 3},
			{MarketDataBase: data.MarketDataBase{Symbol: "MSFT", DataType: data.DataTypeSplit}, SplitRatio: 4},
		},
	}, data.DataRequest{
		Type:      data.DataTypeSplit,
		Offset:    2,
		ChunkSize: 2,
	})

	require.NoError(t, err)
	assert.Equal(t, 2, resp.Offset)
	assert.Equal(t, 3, resp.NextOffset)
	assert.Equal(t, 1, resp.ChunkIndex)
	assert.Equal(t, 3, resp.TotalRows)
	assert.Equal(t, 2, resp.TotalChunks)
	assert.False(t, resp.HasMore)
	require.Len(t, resp.Splits, 1)

	resp.Splits[0].SplitRatio = 5
	require.ErrorContains(t, verifyResponseChecksum(resp), "response checksum mismatch")
}

func TestNormalizeChunkSizeBounds(t *testing.T) {
	assert.Equal(t, 100, normalizeChunkSize(0))
	assert.Equal(t, 100, normalizeChunkSize(-10))
	assert.Equal(t, 250, normalizeChunkSize(250))
	assert.Equal(t, 1000, normalizeChunkSize(5000))
}

func TestValidateChunkResponseRejectsMalformedOffsets(t *testing.T) {
	request := data.DataRequest{Type: data.DataTypeEOD, ChunkSize: 1}
	resp, err := chunkDataResponse(dataResponse{
		Type: data.DataTypeEOD,
		EOD: []data.EODData{
			{MarketDataBase: data.MarketDataBase{Symbol: "AAPL", DataType: data.DataTypeEOD}},
			{MarketDataBase: data.MarketDataBase{Symbol: "AAPL", DataType: data.DataTypeEOD}},
		},
	}, request)
	require.NoError(t, err)

	resp.NextOffset = resp.Offset
	require.ErrorContains(t, validateChunkResponse(resp, request), "response row count mismatch")
}
