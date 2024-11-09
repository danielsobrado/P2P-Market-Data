# P2P Market Data

A decentralized platform for sharing and validating financial market data across a peer-to-peer network.

## Overview

P2P Market Data enables secure, distributed sharing of financial market data between trusted peers. It uses blockchain-inspired consensus mechanisms to validate data integrity and source reputation.

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
git clone https://github.com/yourusername/p2p_market_data.git
cd p2p_market_data

# Install dependencies
go mod download
cd frontend && npm install

# Setup database
psql -U postgres -f db/schema.sql

# Configure environment
cp .env.example .env
```

## Usage

```
# Development
wails dev

# Build
wails build
```

## Folders

├── cmd/           # Application entrypoints
├── pkg/
│   ├── config/    # Configuration handling
│   ├── data/      # Data access layer
│   ├── p2p/       # P2P networking
│   └── scripts/   # Data source scripts
└── frontend/      # React frontend

