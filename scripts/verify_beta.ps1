param(
    [switch]$RunDockerSmoke
)

$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot

function Invoke-Step {
    param(
        [string]$Name,
        [scriptblock]$Command
    )

    Write-Host ""
    Write-Host "==> $Name"
    & $Command
}

Set-Location $repoRoot

$foundationPackages = @(
    "./pkg/config",
    "./pkg/scheduler",
    "./pkg/security",
    "./pkg/pythonenv",
    "./pkg/scripts"
)

$workflowPackages = @(
    "./pkg/data",
    "./pkg/p2p/host",
    "./cmd/app",
    "./cmd/p2pnode"
)

Invoke-Step "Go foundation suites" {
    & go test @foundationPackages -count=1
}

Invoke-Step "Go data and P2P workflow suites" {
    & go test @workflowPackages -count=1
}

Invoke-Step "Frontend production build" {
    Push-Location (Join-Path $repoRoot "frontend")
    try {
        & npm run build
    } finally {
        Pop-Location
    }
}

Invoke-Step "Docker smoke prerequisites" {
    & docker --version | Out-Host
    & docker compose version | Out-Host
}

if ($RunDockerSmoke) {
    Invoke-Step "Two-node Docker P2P smoke" {
        & (Join-Path $PSScriptRoot "docker_p2p_smoke.ps1")
    }
}

Write-Host ""
Write-Host "Beta verification completed."
