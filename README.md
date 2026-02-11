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

# Setup database schema (if using external Postgres)
for f in sql/schema/*.sql; do psql -U postgres -d market_data -f "$f"; done
```

## Usage

```
# Development
wails dev

# Build
wails build
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
