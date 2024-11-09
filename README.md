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

## Architecture

```plaintext
├── cmd/                    # Application entrypoints
│   └── app/               # Main application
├── pkg/                   # Core packages
│   ├── config/           # Configuration management 
│   │   └── loader.go     # YAML config loader
│   ├── data/             # Data persistence
│   │   ├── models/       # Data models
│   │   ├── postgres/     # PostgreSQL implementation
│   │   └── repository.go # Repository interfaces
│   ├── p2p/              # P2P networking
│   │   ├── host/         # libp2p host implementation
│   │   ├── message/      # P2P message definitions
│   │   └── protocol/     # Protocol handlers
│   ├── scheduler/        # Data collection scheduling
│   ├── scripts/          # Data source scripts
│   └── utils/            # Common utilities
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

### Storage Flow
- Validated market data stored in PostgreSQL
- Peer network information persisted in PostgreSQL
- System configuration managed via YAML files
