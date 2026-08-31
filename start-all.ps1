# Start both Backend and Website concurrently in separate terminal windows
Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "         🎬 Dyuet All-In-One Launcher" -ForegroundColor Magenta
Write-Host "=========================================" -ForegroundColor Cyan

$backendPath = Join-Path $PSScriptRoot "backend"
$frontendPath = Join-Path $PSScriptRoot "frontend"

Write-Host "1. Launching Go Backend (http://localhost:8080)..." -ForegroundColor Green
Start-Process powershell -ArgumentList "-NoExit", "-Command", "Set-Location '$backendPath'; go run main.go"

Start-Sleep -Seconds 2

Write-Host "2. Launching Dyuet Website (http://localhost:5173)..." -ForegroundColor Cyan
Start-Process powershell -ArgumentList "-NoExit", "-Command", "Set-Location '$frontendPath'; npm run dev"

Start-Sleep -Seconds 3

Write-Host "3. Opening Dyuet Website in browser..." -ForegroundColor Yellow
Start-Process "http://localhost:5173"

Write-Host "`n✅ Dyuet is running! Enjoy your watch party." -ForegroundColor Green
