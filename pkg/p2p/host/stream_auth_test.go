package host

import (
	"context"
	"testing"
	"time"

	"p2p_market_data/pkg/data"

	"github.com/stretchr/testify/require"
)

func TestDataRequestAuthRoundTripAndReplayProtection(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sender := newIntegrationHost(t, ctx)
	receiver := newIntegrationHost(t, ctx)
	req := data.DataRequest{
		RequestID:   "req-1",
		TransferID:  "tx-1",
		Type:        data.DataTypeEOD,
		Symbol:      "AAPL",
		StartDate:   "2026-01-01",
		EndDate:     "2026-01-31",
		Granularity: "DAILY",
		ChunkSize:   10,
	}

	require.NoError(t, sender.signDataRequest(&req))
	require.NoError(t, receiver.verifyDataRequest(sender.host.ID(), &req))
	require.ErrorContains(t, receiver.verifyDataRequest(sender.host.ID(), &req), "replayed request nonce")
}

func TestDataResponseAuthRejectsTampering(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	requester := newIntegrationHost(t, ctx)
	responder := newIntegrationHost(t, ctx)
	req := data.DataRequest{
		RequestID:   "req-2",
		TransferID:  "tx-2",
		Type:        data.DataTypeEOD,
		Symbol:      "MSFT",
		StartDate:   "2026-01-01",
		EndDate:     "2026-01-31",
		Granularity: "DAILY",
		ChunkSize:   10,
	}
	require.NoError(t, requester.signDataRequest(&req))

	resp := dataResponse{
		RequestID:   req.RequestID,
		TransferID:  req.TransferID,
		Type:        data.DataTypeEOD,
		Offset:      0,
		NextOffset:  1,
		ChunkSize:   10,
		TotalRows:   1,
		TotalChunks: 1,
		EOD: []data.EODData{
			{MarketDataBase: data.MarketDataBase{Symbol: "MSFT", DataType: data.DataTypeEOD}, Close: 100},
		},
	}
	require.NoError(t, responder.signDataResponse(&resp, req))
	require.NoError(t, requester.verifyDataResponse(responder.host.ID(), req, &resp))

	resp.TotalRows = 2
	require.ErrorContains(t, requester.verifyDataResponse(responder.host.ID(), req, &resp), "verifying response signature")
}
