# Production Beta Runbook

## Support Matrix

- OS: Windows 10/11 for the Wails desktop beta.
- Database: embedded PostgreSQL starts on the configured local port, default `5433`.
- Python: a usable Python 3 runtime is required for scripts and virtual environments. The app verifies the interpreter and skips Windows Store shims.
- Node: frontend build uses the `frontend/package.json` toolchain.
- P2P: controlled beta assumes trusted peers on reachable local or private-network addresses.

## Verification

Run the beta verification command from the repository root:

```powershell
.\scripts\verify_beta.ps1
```

Run the two-node Docker exchange smoke when Docker is available:

```powershell
.\scripts\verify_beta.ps1 -RunDockerSmoke
```

Expected checks:

- Go foundation packages pass: config, scheduler, security, Python env, scripts.
- Go data/P2P workflow packages pass.
- Frontend `npm run build` succeeds.
- Docker and Docker Compose are available for P2P smoke.

## First-Run Checklist

- Start the app with `wails dev` or a Wails build artifact.
- Confirm the dashboard opens without schema errors.
- Confirm `GetHealthDiagnostics` reports required tables: `market_data`, `data_sources`, `splits`, `transfers`, `scripts`, and `peers`.
- Upload EOD, dividend, and split rows from Market Data.
- Search by symbol/type/date range and confirm bounded real results.
- Open Transfers and confirm persisted history appears after refresh.
- Upload, install, run, uninstall, and restart-check a script.
- Connect two beta nodes and request market data from the remote peer.

## Known Beta Limits

- Transfers are request/response operations, not resumable chunked downloads.
- Authentication is suitable for controlled beta peers only; stronger auth is post-beta.
- Progress is lifecycle-based and byte-level progress is deferred.
- Windows is the primary supported desktop target for beta packaging.
- Broader observability and alerting are deferred to post-beta.

## Troubleshooting

- Python failures: set `PYTHON_PATH` to a real `python.exe`, not the Windows Store shim.
- Embedded database failures: verify the configured port is free and delete only local beta data after backing up anything needed.
- P2P failures: confirm both peers are reachable and use the full multiaddress from the host diagnostics.
- Script failures: inspect captured stdout/stderr and confirm imports are allowed by configuration.
