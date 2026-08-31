# Start Dyuet Go Backend
Write-Host "🚀 Starting Dyuet Go Backend on port 8080..." -ForegroundColor Green
Set-Location -Path "$PSScriptRoot\backend"
go run main.go
