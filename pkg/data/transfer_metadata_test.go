package data

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransferMetadataRoundTrip(t *testing.T) {
	transfer := &DataTransfer{
		RequestID:       "req-1",
		StartDate:       "2026-01-01",
		EndDate:         "2026-01-31",
		Granularity:     "DAILY",
		ChunkSize:       250,
		TotalRows:       1200,
		TotalChunks:     5,
		CompletedChunks: 3,
		ResumeOffset:    750,
	}

	content, err := marshalTransferMetadata(transfer)
	require.NoError(t, err)

	var restored DataTransfer
	require.NoError(t, unmarshalTransferMetadata(content, &restored))
	assert.Equal(t, transfer.RequestID, restored.RequestID)
	assert.Equal(t, transfer.StartDate, restored.StartDate)
	assert.Equal(t, transfer.EndDate, restored.EndDate)
	assert.Equal(t, transfer.Granularity, restored.Granularity)
	assert.Equal(t, transfer.ChunkSize, restored.ChunkSize)
	assert.Equal(t, transfer.TotalRows, restored.TotalRows)
	assert.Equal(t, transfer.TotalChunks, restored.TotalChunks)
	assert.Equal(t, transfer.CompletedChunks, restored.CompletedChunks)
	assert.Equal(t, transfer.ResumeOffset, restored.ResumeOffset)
}
