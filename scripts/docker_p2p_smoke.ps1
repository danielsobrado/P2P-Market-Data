$ErrorActionPreference = "Stop"

$composeFile = "docker-compose.p2p.yml"

$dockerConfig = Join-Path (Get-Location) ".docker-test-config"
New-Item -ItemType Directory -Force $dockerConfig | Out-Null
Set-Content -Path (Join-Path $dockerConfig "config.json") -Value "{}"
$env:DOCKER_CONFIG = $dockerConfig
if (Test-Path "\\.\pipe\dockerDesktopLinuxEngine") {
    $env:DOCKER_HOST = "npipe:////./pipe/dockerDesktopLinuxEngine"
}

$binaryDir = Join-Path (Get-Location) "build\docker"
New-Item -ItemType Directory -Force $binaryDir | Out-Null

$env:CGO_ENABLED = "0"
$env:GOOS = "linux"
$env:GOARCH = "amd64"
$env:GOMAXPROCS = "2"
$env:GOGC = "50"
go build -p 1 -o (Join-Path $binaryDir "p2pnode") ./cmd/p2pnode
if ($LASTEXITCODE -ne 0) {
    throw "go build failed"
}

docker compose -f $composeFile down --volumes --remove-orphans
if ($LASTEXITCODE -ne 0) {
    throw "docker compose cleanup failed. Is Docker Desktop running?"
}

docker compose -f $composeFile up -d --build --force-recreate
if ($LASTEXITCODE -ne 0) {
    throw "docker compose failed. Is Docker Desktop running?"
}

function Wait-Json($Url) {
    $deadline = (Get-Date).AddSeconds(60)
    do {
        try {
            return Invoke-RestMethod -Uri $Url -TimeoutSec 3
        } catch {
            Start-Sleep -Seconds 1
        }
    } while ((Get-Date) -lt $deadline)

    throw "Timed out waiting for $Url"
}

Wait-Json "http://localhost:18080/health" | Out-Null
Wait-Json "http://localhost:18081/health" | Out-Null

$node1 = Wait-Json "http://localhost:18080/status"
$node1Addr = "/dns4/p2p-node-1/tcp/9000/p2p/$($node1.peer_id)"

Invoke-RestMethod `
    -Uri "http://localhost:18081/connect" `
    -Method Post `
    -ContentType "application/json" `
    -Body (@{ addr = $node1Addr } | ConvertTo-Json) | Out-Null

1..3 | ForEach-Object {
    $payload = @{
        symbol = "BTCUSD"
        price = 50000 + $_
        volume = 12 + $_
        source = "docker-smoke"
        data_type = "EOD"
    } | ConvertTo-Json

    Invoke-RestMethod `
        -Uri "http://localhost:18080/market-data" `
        -Method Post `
        -ContentType "application/json" `
        -Body $payload | Out-Null
}

$requestPayload = @{
    peer_id = $node1.peer_id
    type = "EOD"
    symbol = "BTCUSD"
    start_date = (Get-Date).AddDays(-1).ToUniversalTime().ToString("yyyy-MM-dd")
    end_date = (Get-Date).AddDays(1).ToUniversalTime().ToString("yyyy-MM-dd")
    granularity = "DAILY"
    chunk_size = 1
} | ConvertTo-Json

Invoke-RestMethod `
    -Uri "http://localhost:18081/request-data" `
    -Method Post `
    -ContentType "application/json" `
    -Body $requestPayload | Out-Null

$deadline = (Get-Date).AddSeconds(30)
do {
    $records = Invoke-RestMethod -Uri "http://localhost:18081/market-data?symbol=BTCUSD" -TimeoutSec 3
    if ($records.Count -ge 3) {
        Write-Host "P2P Docker smoke test passed. Node 2 received $($records.Count) BTCUSD record(s)."
        $records | ConvertTo-Json -Depth 8
        if ($env:KEEP_P2P_SMOKE -ne "1") {
            docker compose -f $composeFile down --volumes --remove-orphans | Out-Host
        }
        exit 0
    }
    Start-Sleep -Seconds 1
} while ((Get-Date) -lt $deadline)

throw "Node 2 did not receive market data from node 1"
