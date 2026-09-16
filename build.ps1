# build.ps1 — Script de build pour Magenta
# Synchronise le dossier web et compile le binaire

param(
    [string]$Version = "dev",
    [string]$Os = $env:GOOS,
    [string]$Arch = $env:GOARCH
)

Write-Host "Building Magenta $Version..." -ForegroundColor Cyan

# Étape 1 : Synchronisation du dossier web dans cmd/mediaserver
Write-Host "Syncing web assets..." -ForegroundColor Yellow
if (Test-Path "cmd\mediaserver\web") {
    Remove-Item -Recurse -Force "cmd\mediaserver\web"
}
Copy-Item -Recurse "web" "cmd\mediaserver\web"

# Étape 2 : Compilation
Write-Host "Compiling..." -ForegroundColor Yellow
$ldflags = "-X main.version=$Version"
go build -ldflags $ldflags -o bin/magenta.exe ./cmd/mediaserver

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build successful: bin/magenta.exe" -ForegroundColor Green
} else {
    Write-Host "Build failed!" -ForegroundColor Red
    exit 1
}
