# P2P Market Data

A decentralized platform for sharing and validating financial market data across a peer-to-peer network.

Still **WORK IN PROGRESS**

## Overview

P2P Market Data enables secure, distributed sharing of financial market data between trusted peers. It uses blockchain-inspired consensus mechanisms to validate data integrity and source reputation.

![P2P Market Data](https://github.com/danielsobrado/p2p_market_data/blob/main/images/P2P_MD_Screenshot.png)

## Features

- Decentralized P2P network for data sharing
- Real-time market data validation
- Reputation-based peer scoring
- Data integrity verification
- Configurable data sources
- Dark/light theme support

## Tech Stack

- Frontend: React + TypeScript + Vite
- Backend: Go
- UI Components: shadcn/ui
- P2P Network: libp2p
- Desktop: Wails
- Database: PostgreSQL

## Installation

```bash
# Clone repository
git clone https://github.com/danielsobrado/p2p_market_data.git
cd p2p_market_data

# Install dependencies
go mod download
cd frontend && npm install

# Build frontend assets (required by Go embed)
cd frontend && npm install && npm run build && cd ..

# Setup database schema (only needed if you are managing Postgres outside the app)
for f in sql/schema/*.sql; do psql -U postgres -d market_data -f "$f"; done
```

## Usage

```
# Development
wails dev

# Build
wails build
```

When the Wails app starts its embedded PostgreSQL instance, it now initializes the required schema automatically.

## Production Beta Verification

Run the local beta verification script before packaging or handing a build to controlled users:

```powershell
.\scripts\verify_beta.ps1
```

This runs the critical Go foundation suites, data/P2P workflow suites, the frontend production build, and Docker smoke prerequisites. Add `-RunDockerSmoke` to run the two-node P2P container exchange. See [docs/beta-runbook.md](docs/beta-runbook.md) for the support matrix, first-run checklist, and known beta limits.

## Docker P2P Smoke Test

The repository includes a headless node for testing P2P market-data exchange between containers.

```powershell
.\scripts\docker_p2p_smoke.ps1
```

The script builds two Docker services, connects `p2p-node-2` to `p2p-node-1`, publishes a `BTCUSD` market-data record on node 1, and verifies that node 2 receives and stores it over libp2p pubsub.

Manual flow:

```powershell
docker compose -f docker-compose.p2p.yml up -d --build
$node1 = Invoke-RestMethod http://localhost:18080/status
$addr = "/dns4/p2p-node-1/tcp/9000/p2p/$($node1.peer_id)"
Invoke-RestMethod http://localhost:18081/connect -Method Post -ContentType "application/json" -Body (@{addr=$addr} | ConvertTo-Json)
Invoke-RestMethod http://localhost:18080/market-data -Method Post -ContentType "application/json" -Body '{"symbol":"BTCUSD","price":50000,"volume":12,"source":"manual","data_type":"EOD"}'
Invoke-RestMethod "http://localhost:18081/market-data?symbol=BTCUSD"
```

## Directory Structure

```plaintext
├── cmd/                    # Application entrypoints
│   └── app/               # Main application
├── pkg/                   # Core packages
│   ├── config/            # Configuration management
│   ├── data/              # Data models/repository/schema
│   ├── database/          # Database service lifecycle
│   ├── p2p/               # P2P networking (host/voting/authority)
│   ├── scripts/           # Script execution and management
│   └── utils/             # Common utilities
└── frontend/             # React frontend
    ├── src/              # Frontend source code
    │   ├── components/   # React components
    │   ├── hooks/        # Custom React hooks
    │   └── types/        # TypeScript definitions
    └── public/           # Static assets
```

## Process Details

### Data Flow
- External market data sources feed by using scrapping scripts (Can be shared)
- Scheduler orchestrates data collection timing
- Repository handles persistence and retrieval
- P2P network enables decentralized data sharing
- Frontend displays data through Backend API

### Validation Flow
- Collected data is broadcast to P2P network
- Peers validate data authenticity and accuracy (Voting)
- Consensus mechanism confirms validation
- Peer reputation scores updated based on accuracy

### Storage
- Validated market data stored in PostgreSQL
- Peer network information persisted in PostgreSQL
- System configuration managed via YAML files
