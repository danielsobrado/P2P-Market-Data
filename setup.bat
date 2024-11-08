@echo off
setlocal enabledelayedexpansion

:: Create main directories
mkdir cmd\app
mkdir pkg\config
mkdir pkg\data
mkdir pkg\p2p
mkdir pkg\scheduler
mkdir pkg\scripts
mkdir pkg\security
mkdir pkg\utils
mkdir pkg\pythonenv
mkdir scripts
mkdir frontend

:: Create Go files in cmd
echo. > cmd\app\main.go

:: Create Go files in pkg\config
echo. > pkg\config\config.go
echo. > pkg\config\config_test.go

:: Create Go files in pkg\data
echo. > pkg\data\models.go
echo. > pkg\data\repository.go
echo. > pkg\data\repository_test.go

:: Create Go files in pkg\p2p
echo. > pkg\p2p\host.go
echo. > pkg\p2p\message.go
echo. > pkg\p2p\network.go
echo. > pkg\p2p\network_test.go
echo. > pkg\p2p\authority.go
echo. > pkg\p2p\voting.go

:: Create Go files in pkg\scheduler
echo. > pkg\scheduler\scheduler.go
echo. > pkg\scheduler\scheduler_test.go

:: Create Go files in pkg\scripts
echo. > pkg\scripts\executor.go
echo. > pkg\scripts\manager.go
echo. > pkg\scripts\executor_test.go
echo. > pkg\scripts\manager_test.go

:: Create Go files in pkg\security
echo. > pkg\security\cryptography.go
echo. > pkg\security\cryptography_test.go
echo. > pkg\security\reputation.go
echo. > pkg\security\reputation_test.go

:: Create Go files in pkg\utils
echo. > pkg\utils\logger.go
echo. > pkg\utils\helpers.go
echo. > pkg\utils\logger_test.go
echo. > pkg\utils\helpers_test.go

:: Create Go files in pkg\pythonenv
echo. > pkg\pythonenv\env.go
echo. > pkg\pythonenv\env_test.go

:: Create Python script
echo # Example Python script > scripts\example_script.py

:: Create initial go.mod and go.sum
echo module p2p_market_data > go.mod
echo. > go.sum

:: Print directory structure
echo Project structure created successfully!
tree /F